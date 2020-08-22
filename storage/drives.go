package storage

import (
	"encoding/json"
	"go-drive/common/types"
	"go-drive/drive"
	"log"
)

var drivesFactory = map[string]types.DriveCreator{
	"fs": drive.NewFsDrive,
}

type DriveStorage struct {
	db   *DB
	root *drive.DispatcherDrive
}

func NewDriveStorage(db *DB) (*DriveStorage, error) {
	ds := DriveStorage{root: drive.NewDispatcherDrive(), db: db}
	return &ds, ds.ReloadDrive()
}

func (d *DriveStorage) GetRootDrive() types.IDrive {
	return d.root
}

func (d *DriveStorage) ReloadDrive() error {
	var drivesConfig []types.Drive
	if e := d.db.C().Find(&drivesConfig).Error; e != nil {
		return e
	}
	drives := make(map[string]types.IDrive)
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
