package storage

import (
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"go-drive/common"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
)

var initSQL = []string{
	"INSERT INTO users(username, password) VALUES ('admin', '$2y$10$Xqn8qV2D2KY2ceI5esM/JOiKTPKJFbkSzzuhce89BxygvCqnhyk3m')", // 123456
	"INSERT INTO groups(name) VALUES ('admin')",
	"INSERT INTO user_groups(username, group_name) VALUES ('admin', 'admin')",
	"INSERT INTO path_permissions(path, subject, permission, policy, depth) VALUES ('', 'ANY', 1, 1, 0)",
	"INSERT INTO path_permissions(path, subject, permission, policy, depth) VALUES ('', 'g:admin', 3, 1, 0)",
}

func NewDB(config common.Config, ch *registry.ComponentsHolder) (*DB, error) {
	dialect, args := config.GetDB()

	db, e := gorm.Open(dialect, args)
	if e != nil {
		return nil, e
	}

	if utils.IsDebugOn() {
		db.LogMode(true)
	}

	if e := db.AutoMigrate(
		&types.User{},
		&types.Group{},
		&types.UserGroup{},
		&types.Drive{},
		&types.PathPermission{},
		&types.PathMount{},
		&types.DriveData{},
		&types.DriveCache{},
	).Error; e != nil {
		_ = db.Close()
		return nil, e
	}

	if e := tryInitDbData(db); e != nil {
		_ = db.Close()
		return nil, e
	}

	d := &DB{db: db}
	ch.Add("db", d)
	return d, nil
}

func tryInitDbData(db *gorm.DB) error {
	n := 0
	if e := db.Model(&types.User{}).Count(&n).Error; e != nil {
		return e
	}
	if n > 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, initSQL := range initSQL {
			if e := tx.Exec(initSQL).Error; e != nil {
				return e
			}
		}
		return nil
	})
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
