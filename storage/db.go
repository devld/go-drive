package storage

import (
	"go-drive/common"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

var initSQL = []string{
	"INSERT INTO users(username, password) VALUES ('admin', '$2y$10$Xqn8qV2D2KY2ceI5esM/JOiKTPKJFbkSzzuhce89BxygvCqnhyk3m')", // 123456
	"INSERT INTO groups(name) VALUES ('admin')",
	"INSERT INTO user_groups(username, group_name) VALUES ('admin', 'admin')",
	"INSERT INTO path_permissions(path, subject, permission, policy) VALUES ('', 'ANY', 1, 1)",
	"INSERT INTO path_permissions(path, subject, permission, policy) VALUES ('', 'g:admin', 3, 1)",
}

func NewDB(config common.Config, ch *registry.ComponentsHolder) (*DB, error) {
	dialect := config.GetDB()
	dbConfig := gorm.Config{}

	if utils.IsDebugOn() {
		dbConfig.Logger = logger.New(
			log.New(os.Stdout, "\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: false,
				Colorful:                  true,
			},
		)
	}

	db, e := gorm.Open(dialect, &dbConfig)
	if e != nil {
		return nil, e
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
	); e != nil {
		closeDb(db)
		return nil, e
	}

	if e := tryInitDbData(db); e != nil {
		closeDb(db)
		return nil, e
	}

	d := &DB{db: db}
	ch.Add("db", d)
	return d, nil
}

func tryInitDbData(db *gorm.DB) error {
	var n int64 = 0
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
	closeDb(d.db)
	return nil
}

func closeDb(db *gorm.DB) {
	if db != nil {
		sqlDb, e := db.DB()
		if e == nil {
			_ = sqlDb.Close()
		}
	}
}

func (d *DB) C() *gorm.DB {
	return d.db
}
