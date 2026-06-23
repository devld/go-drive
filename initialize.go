package main

import (
	"context"
	"go-drive/common"
	"go-drive/common/event"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/server"
	"go-drive/server/job"
	"go-drive/server/search"
	"go-drive/server/thumbnail"
	"go-drive/storage"

	"github.com/gin-gonic/gin"
)

func Initialize(ctx context.Context, ch *registry.ComponentsHolder) (*gin.Engine, error) {
	config, err := common.InitConfig(ch)
	if err != nil {
		return nil, err
	}
	bus := event.NewBus(ch)
	db, err := storage.NewDB(config, ch)
	if err != nil {
		return nil, err
	}
	driveDAO := storage.NewDriveDAO(db, ch)
	pathMountDAO := storage.NewPathMountDAO(db, ch)
	driveDataDAO := storage.NewDriveDataDAO(db, ch)
	driveCacheDAO := storage.NewDriveCacheDAO(db, ch)
	rootDrive, err := drive.NewRootDrive(ctx, config, driveDAO, pathMountDAO, driveDataDAO, driveCacheDAO, ch)
	if err != nil {
		return nil, err
	}
	pathPermissionDAO := storage.NewPathPermissionDAO(db, ch)
	optionsDAO := storage.NewOptionsDAO(db, ch)
	pathMetaDAO := storage.NewPathMetaDAO(db, ch)
	access, err := drive.NewAccess(ch, rootDrive, pathPermissionDAO, optionsDAO, pathMetaDAO, bus)
	if err != nil {
		return nil, err
	}
	runner := task.NewPondRunner(config, ch)
	service, err := search.NewService(ch, config, optionsDAO, rootDrive, runner, bus)
	if err != nil {
		return nil, err
	}
	userDAO := storage.NewUserDAO(db, ch)
	sessionDAO := storage.NewSessionDAO(db, ch)
	dbTokenStore, err := server.NewDBTokenStore(sessionDAO, userDAO, config, ch)
	if err != nil {
		return nil, err
	}
	maker, err := thumbnail.NewMaker(config, optionsDAO, ch)
	if err != nil {
		return nil, err
	}
	signer := utils.NewSigner()
	chunkUploader, err := server.NewChunkUploader(config)
	if err != nil {
		return nil, err
	}
	groupDAO := storage.NewGroupDAO(db, ch)
	jobDAO := storage.NewJobDAO(db, ch)
	fileBucketDAO := storage.NewFileBucketDAO(db, ch)
	jobExecutor, err := job.NewJobExecutor(jobDAO, ch)
	if err != nil {
		return nil, err
	}
	fileMessageSource, err := i18n.NewFileMessageSource(langResourceFS())
	if err != nil {
		return nil, err
	}
	engine, err := server.InitServer(config, ch, bus, rootDrive, access,
		service, dbTokenStore, maker, signer, chunkUploader, runner,
		optionsDAO, userDAO, groupDAO, driveDAO, driveDataDAO, pathPermissionDAO,
		pathMountDAO, pathMetaDAO, jobDAO, fileBucketDAO,
		jobExecutor, fileMessageSource, webResourceFS())
	if err != nil {
		return nil, err
	}
	return engine, nil
}
