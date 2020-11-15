package storage

import (
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"go-drive/common"
	"go-drive/common/types"
)

type DB struct {
	db *gorm.DB
}

func InitDB(dialect string, args ...interface{}) (*DB, error) {
	db, e := gorm.Open(dialect, args...)
	if e != nil {
		return nil, e
	}

	if common.IsDebugOn() {
		db.LogMode(true)
	}

	db.AutoMigrate(
		&types.User{},
		&types.Group{},
		&types.UserGroup{},
		&types.Drive{},
		&types.PathMount{},
		&types.DriveData{},
		&types.DriveCache{},
	)

	return &DB{db: db}, nil
}

func (d *DB) C() *gorm.DB {
	return d.db
}
