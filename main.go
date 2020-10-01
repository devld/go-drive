package main

import (
	"go-drive/common"
	"go-drive/common/task"
	"go-drive/drive"
	"go-drive/server"
	"go-drive/storage"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	config := common.InitConfig()
	componentsHolder := initComponentsHolder(config)

	engine, e := server.InitServer(componentsHolder, config.GetResDir())
	common.PanicIfError(e)

	panic(engine.Run(config.GetListen()))
}

func initComponentsHolder(config common.Config) *server.ComponentsHolder {
	dbDialect, dbArg := config.GetDB()

	db, e := storage.InitDB(dbDialect, dbArg)
	common.PanicIfError(e)

	tokenStore := server.NewMemTokenStore(12*time.Hour, true, 1*time.Hour)

	requestSigner := common.NewSigner(common.RandString(32))

	driveStorage, e := storage.NewDriveStorage(db)
	common.PanicIfError(e)
	userStorage, e := storage.NewUserStorage(db)
	common.PanicIfError(e)
	groupStorage, e := storage.NewGroupStorage(db)
	common.PanicIfError(e)
	permissionStorage, e := storage.NewPathPermissionStorage(db)
	common.PanicIfError(e)
	pathMountStorage, e := storage.NewPathMountStorage(db)
	common.PanicIfError(e)
	rootDrive, e := drive.NewRootDrive(driveStorage, pathMountStorage)
	common.PanicIfError(e)

	chunksTempDir, e := config.GetDir("upload_temp", true)
	common.PanicIfError(e)
	chunkUploader, e := server.NewChunkUploader(chunksTempDir)
	common.PanicIfError(e)

	return &server.ComponentsHolder{
		TokenStore:        tokenStore,
		RootDrive:         rootDrive,
		DriveStorage:      driveStorage,
		UserStorage:       userStorage,
		GroupStorage:      groupStorage,
		PermissionStorage: permissionStorage,
		PathMountStorage:  pathMountStorage,
		RequestSigner:     requestSigner,
		TaskRunner:        task.NewTunnyRunner(100),
		ChunkUploader:     chunkUploader,
	}
}
