package main

import (
	"context"
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/event"
	"go-drive/common/i18n"
	"go-drive/common/logging"
	"go-drive/common/registry"
	"go-drive/common/secretbox"
	"go-drive/common/task"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/server"
	artifactinit "go-drive/server/artifact/builtin"
	"go-drive/server/auth"
	"go-drive/server/job"
	"go-drive/server/search"
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

func Initialize(ctx context.Context, ch *registry.ComponentsHolder) (*gin.Engine, common.Config, error) {
	started := time.Now()
	log := logging.For("start")
	phase := initPhase("config")
	config, err := common.InitConfig()
	if err := phase(err); err != nil {
		return nil, config, err
	}
	ch.Add(common.NewVersionSysConfig())

	phase = initPhase("drive registry")
	driveRegistry := driveutil.NewDriveRegistry()
	if err := phase(nil); err != nil {
		return nil, config, err
	}

	phase = initPhase("drive registration")
	if err := drive.RegisterAllDrives(ctx, config, driveRegistry); err != nil {
		return nil, config, phase(err)
	}
	if err := phase(nil); err != nil {
		return nil, config, err
	}
	bus := event.NewBus()

	phase = initPhase("encryption key")
	secrets, err := secretbox.Open(config.DataDir)
	if err := phase(err); err != nil {
		return nil, config, err
	}

	phase = initPhase("database")
	db, err := storage.NewDB(config, secrets)
	if err := phase(err); err != nil {
		return nil, config, err
	}
	ch.Add(db)
	driveDAO := storage.NewDriveDAO(db, secrets)
	pathMountDAO := storage.NewPathMountDAO(db)
	driveDataDAO := storage.NewDriveDataDAO(db, secrets)
	driveCacheDAO := storage.NewDriveCacheDAO(db)
	ch.Add(driveCacheDAO)

	phase = initPhase("root drive")
	rootDrive, err := drive.NewRootDrive(ctx, config, driveDAO, pathMountDAO, driveDataDAO, driveCacheDAO, driveRegistry)
	if err := phase(err); err != nil {
		return nil, config, err
	}
	ch.Add(rootDrive)
	pathPermissionDAO := storage.NewPathPermissionDAO(db)
	optionsDAO := storage.NewOptionsDAO(db)
	ch.Add(optionsDAO)
	pathMetaDAO := storage.NewPathMetaDAO(db)
	phase = initPhase("drive access")
	access, err := drive.NewAccess(rootDrive, pathPermissionDAO, optionsDAO, pathMetaDAO, bus)
	if err := phase(err); err != nil {
		return nil, config, err
	}

	phase = initPhase("task runner")
	runner := task.NewTaskRunner(config)
	ch.Add(runner)
	if err := phase(nil); err != nil {
		return nil, config, err
	}

	phase = initPhase("search")
	service, err := search.NewService(config, optionsDAO, rootDrive, runner, bus)
	if err := phase(err); err != nil {
		return nil, config, err
	}
	ch.Add(service)
	userDAO := storage.NewUserDAO(db)
	ch.Add(userDAO)
	sessionDAO := storage.NewSessionDAO(db)
	phase = initPhase("token store")
	dbTokenStore, err := server.NewDBTokenStore(sessionDAO, userDAO, config)
	if err := phase(err); err != nil {
		return nil, config, err
	}
	ch.Add(dbTokenStore)

	phase = initPhase("drive fs")
	driveFS, err := driveutil.NewDriveFS(config)
	if err := phase(err); err != nil {
		return nil, config, err
	}
	ch.Add(driveFS)

	phase = initPhase("artifact previews")
	artifactService, err := artifactinit.Initialize(config, optionsDAO, runner, driveFS)
	if err := phase(err); err != nil {
		return nil, config, err
	}
	ch.Add(artifactService)
	signer := utils.NewSigner()
	phase = initPhase("chunk uploader")
	chunkUploader, err := server.NewChunkUploader(config)
	if err := phase(err); err != nil {
		return nil, config, err
	}
	groupDAO := storage.NewGroupDAO(db, userDAO)
	jobDAO := storage.NewJobDAO(db)
	fileBucketDAO := storage.NewFileBucketDAO(db)
	ch.Add(fileBucketDAO)
	phase = initPhase("job executor")
	jobExecutor, err := job.NewJobExecutor(jobDAO, runner, bus, access)
	if err := phase(err); err != nil {
		return nil, config, err
	}
	ch.Add(jobExecutor)

	phase = initPhase("messages")
	fileMessageSource, err := i18n.NewFileMessageSource(langResourceFS())
	if err := phase(err); err != nil {
		return nil, config, err
	}

	phase = initPhase("server")
	userAuth, err := auth.NewUserAuth(config.Auth.Providers, userDAO, groupDAO)
	if err := phase(err); err != nil {
		return nil, config, err
	}
	ch.Add(userAuth)
	failBanGroup := server.NewFailBanGroup(10 * time.Minute)
	ch.Add(failBanGroup)
	ch.Add(server.NewRuntimeStat())
	engine, err := server.InitServer(config, ch, driveRegistry, userAuth, failBanGroup, bus, rootDrive, access, driveFS,
		service, dbTokenStore, artifactService, signer, chunkUploader, runner,
		optionsDAO, userDAO, groupDAO, driveDAO, driveDataDAO, pathPermissionDAO,
		pathMountDAO, pathMetaDAO, jobDAO, fileBucketDAO,
		jobExecutor, fileMessageSource, webResourceFS())
	if err := phase(err); err != nil {
		return nil, config, err
	}
	log.Infof("initialization completed duration=%s", time.Since(started))
	return engine, config, nil
}
