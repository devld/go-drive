package db

import (
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"go-drive/common"
)

var db *gorm.DB

func InitDB() error {
	db_, e := gorm.Open("sqlite3", common.GetDBFile())
	if e != nil {
		return e
	}
	db = db_

	db.AutoMigrate(&User{}, &Group{}, &Drive{})

	return nil
}

func GetDB() *gorm.DB {
	common.RequireNotNil(db, "database not initialized")
	return db
}
