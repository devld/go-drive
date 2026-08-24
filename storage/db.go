package storage

import (
	"go-drive/common"
	"go-drive/common/logging"
	"go-drive/common/registry"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var dbLog = logging.For("db")

func NewDB(config common.Config, ch *registry.ComponentsHolder) (*DB, error) {
	dbLog.Debugf("opening database type=%s", config.Db.Type)
	dialect := config.GetDB()
	gormConfig := logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	}
	if logging.Enabled(logging.DebugLevel) {
		gormConfig.LogLevel = logger.Info
		gormConfig.IgnoreRecordNotFoundError = false
	}
	dbConfig := gorm.Config{Logger: logging.NewGormLogger("db", gormConfig)}

	db, e := gorm.Open(dialect, &dbConfig)
	if e != nil {
		return nil, e
	}

	if e := migrateAll(db); e != nil {
		closeDb(db)
		return nil, e
	}
	dbLog.Debugf("database ready type=%s", config.Db.Type)

	d := &DB{db: db}
	ch.Add(registry.KeyDB, d)
	return d, nil
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
