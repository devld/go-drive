package main

import (
	"context"
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/event"
	"go-drive/common/i18n"
	"go-drive/common/logging"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/server"
	"go-drive/server/job"
	"go-drive/server/search"
	"go-drive/server/thumbnail"
	"go-drive/storage"
	"time"

	"github.com/gin-gonic/gin"
)

func initPhase(name string) func(error) error {
	started := time.Now()
	log := logging.For("start")
	log.Debugf("initialization started phase=%s", name)
	return func(e error) error {
		if e != nil {
			log.Errorf("initialization failed phase=%s duration=%s: %v", name, time.Since(started), e)
			return e
		}
		log.Debugf("initialization completed phase=%s duration=%s", name, time.Since(started))
		return nil
	}
}

func Initialize(ctx context.Context, ch *registry.ComponentsHolder) (*gin.Engine, error) {
	started := time.Now()
	log := logging.For("start")
	phase := initPhase("config")
	config, err := common.InitConfig(ch)
	if err := phase(err); err != nil {
		return nil, err
	}

	phase = initPhase("drive registry")
	driveutil.NewDriveRegistry(ch)
	if err := phase(nil); err != nil {
		return nil, err
	}

	phase = initPhase("drive registration")
	if err := drive.RegisterAllDrives(ctx, config, ch); err != nil {
		return nil, phase(err)
	}
	if err := phase(nil); err != nil {
		return nil, err
	}
	bus := event.NewBus(ch)

	phase = initPhase("database")
	db, err := storage.NewDB(config, ch)
	if err := phase(err); err != nil {
		return nil, err
	}
	driveDAO := storage.NewDriveDAO(db, ch)
	pathMountDAO := storage.NewPathMountDAO(db, ch)
	driveDataDAO := storage.NewDriveDataDAO(db, ch)
	driveCacheDAO := storage.NewDriveCacheDAO(db, ch)

	phase = initPhase("root drive")
	rootDrive, err := drive.NewRootDrive(ctx, config, driveDAO, pathMountDAO, driveDataDAO, driveCacheDAO, ch)
	if err := phase(err); err != nil {
		return nil, err
	}
	pathPermissionDAO := storage.NewPathPermissionDAO(db, ch)
	optionsDAO := storage.NewOptionsDAO(db, ch)
	pathMetaDAO := storage.NewPathMetaDAO(db, ch)
	phase = initPhase("drive access")
	access, err := drive.NewAccess(ch, rootDrive, pathPermissionDAO, optionsDAO, pathMetaDAO, bus)
	if err := phase(err); err != nil {
		return nil, err
	}

	phase = initPhase("task runner")
	runner := task.NewPondRunner(config, ch)
	if err := phase(nil); err != nil {
		return nil, err
	}

	phase = initPhase("search")
	service, err := search.NewService(ch, config, optionsDAO, rootDrive, runner, bus)
	if err := phase(err); err != nil {
		return nil, err
	}
	userDAO := storage.NewUserDAO(db, ch)
	sessionDAO := storage.NewSessionDAO(db, ch)
	phase = initPhase("token store")
	dbTokenStore, err := server.NewDBTokenStore(sessionDAO, userDAO, config, ch)
	if err := phase(err); err != nil {
		return nil, err
	}

	phase = initPhase("thumbnail")
	maker, err := thumbnail.NewMaker(config, optionsDAO, ch)
	if err := phase(err); err != nil {
		return nil, err
	}
	signer := utils.NewSigner()
	phase = initPhase("chunk uploader")
	chunkUploader, err := server.NewChunkUploader(config)
	if err := phase(err); err != nil {
		return nil, err
	}
	groupDAO := storage.NewGroupDAO(db, userDAO, ch)
	jobDAO := storage.NewJobDAO(db, ch)
	fileBucketDAO := storage.NewFileBucketDAO(db, ch)
	phase = initPhase("job executor")
	jobExecutor, err := job.NewJobExecutor(jobDAO, ch)
	if err := phase(err); err != nil {
		return nil, err
	}

	phase = initPhase("messages")
	fileMessageSource, err := i18n.NewFileMessageSource(langResourceFS())
	if err := phase(err); err != nil {
		return nil, err
	}

	phase = initPhase("server")
	engine, err := server.InitServer(config, ch, bus, rootDrive, access,
		service, dbTokenStore, maker, signer, chunkUploader, runner,
		optionsDAO, userDAO, groupDAO, driveDAO, driveDataDAO, pathPermissionDAO,
		pathMountDAO, pathMetaDAO, jobDAO, fileBucketDAO,
		jobExecutor, fileMessageSource, webResourceFS())
	if err := phase(err); err != nil {
		return nil, err
	}
	log.Infof("initialization completed duration=%s", time.Since(started))
	return engine, nil
}
