package storage

import (
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"go-drive/common"
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

	return &DB{db: db}, nil
}

func (d *DB) C() *gorm.DB {
	return d.db
}
