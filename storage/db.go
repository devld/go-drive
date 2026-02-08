package storage

import (
	"go-drive/common"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"log"
	"os"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(config common.Config, ch *registry.ComponentsHolder) (*DB, error) {
	dialect := config.GetDB()
	dbConfig := gorm.Config{}

	if utils.IsDebugOn {
		dbConfig.Logger = logger.New(
			log.New(os.Stdout, "\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: false,
				Colorful:                  true,
			},
		)
	} else {
		dbConfig.Logger = logger.New(
			log.New(os.Stdout, "\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
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
		&types.Option{},
		&types.Job{},
		&types.JobExecution{},
		&types.PathMeta{},
		&types.FileBucket{},
	); e != nil {
		closeDb(db)
		return nil, e
	}

	if e := tryInitDbData(db); e != nil {
		closeDb(db)
		return nil, e
	}

	// Run migrations
	if e := migrateJobScheduleToTriggers(db); e != nil {
		closeDb(db)
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
