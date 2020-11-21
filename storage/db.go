package storage

import (
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"go-drive/common"
	"go-drive/common/types"
)

const DbOrder = -4096

func init() {
	common.R().Register("db", func(c *common.ComponentRegistry) interface{} {
		dialect, args := c.Get("config").(common.Config).GetDB()

		db, e := gorm.Open(dialect, args)
		if e != nil {
			panic(e)
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

		return &DB{db: db}
	}, DbOrder)
}

type DB struct {
	db *gorm.DB
}

func (d *DB) C() *gorm.DB {
	return d.db
}
