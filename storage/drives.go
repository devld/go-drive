package storage

import (
	"encoding/json"
	"go-drive/common"
	"go-drive/drive"
	"go-drive/storage/db"
	"log"
)

var (
	rootDrive = drive.NewDrive()
)

type DriveCreator = func(map[string]string) (common.IDrive, error)

var drivesFactory = map[string]DriveCreator{
	"fs": drive.NewFsDrive,
}

func InitRootDrive() error {
	return ReloadDrive()
}

func GetRootDrive() common.IDrive {
	common.RequireNotNil(rootDrive, "drives not initialized")
	return rootDrive
}

func ReloadDrive() error {
	var drivesConfig []db.Drive
	if e := db.GetDB().Find(&drivesConfig).Error; e != nil {
		return e
	}
	drives := make(map[string]common.IDrive)
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
	rootDrive.SetDrives(drives)
	return nil
}
