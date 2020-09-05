package drive

import (
	"encoding/json"
	"go-drive/common/types"
	"go-drive/storage"
	"log"
)

var drivesFactory = map[string]types.DriveCreator{
	"fs": NewFsDrive,
}

type RootDrive struct {
	root    *DispatcherDrive
	storage *storage.DriveStorage
}

func NewRootDrive(storage *storage.DriveStorage) (*RootDrive, error) {
	r := &RootDrive{
		root:    NewDispatcherDrive(),
		storage: storage,
	}
	e := r.ReloadDrive()
	return r, e
}

func (d *RootDrive) Get() types.IDrive {
	return d.root
}

func (d *RootDrive) ReloadDrive() error {
	drivesConfig, e := d.storage.GetDrives()
	if e != nil {
		return e
	}
	drives := make(map[string]types.IDrive, len(drivesConfig))
	for _, d := range drivesConfig {
		create, ok := drivesFactory[d.Type]
		if !ok {
			log.Printf("invalid drive type '%s'", d.Type)
			continue
		}
		config := make(map[string]string)
		e := json.Unmarshal([]byte(d.Config), &config)
		if e != nil {
			log.Printf("invalid drive config of '%s'", d.Name)
			continue
		}
		iDrive, e := create(config)
		if e != nil {
			log.Printf("error when creating drive '%s': %s", d.Name, e.Error())
			continue
		}
		drives[d.Name] = iDrive
		log.Printf("drive '%s' of type '%s' added", d.Name, d.Type)
	}
	d.root.SetDrives(drives)
	return nil
}
