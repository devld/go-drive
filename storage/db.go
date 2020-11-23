package storage

import (
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"go-drive/common"
	"go-drive/common/types"
)

func NewDB(config common.Config, ch *common.ComponentsHolder) (*DB, error) {
	dialect, args := config.GetDB()

	db, e := gorm.Open(dialect, args)
	if e != nil {
		return nil, e
	}

	if common.IsDebugOn() {
		db.LogMode(true)
	}

	if e := db.AutoMigrate(
		&types.User{},
		&types.Group{},
		&types.UserGroup{},
		&types.Drive{},
		&types.PathMount{},
		&types.DriveData{},
		&types.DriveCache{},
	).Error; e != nil {
		_ = db.Close()
		return nil, e
	}

	d := &DB{db: db}
	ch.Add("db", d)
	return d, nil
}

type DB struct {
	db *gorm.DB
}

func (d *DB) Dispose() error {
	return d.db.Close()
}

func (d *DB) C() *gorm.DB {
	return d.db
}
