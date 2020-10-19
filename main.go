package main

import (
	"go-drive/common"
	"go-drive/common/task"
	"go-drive/drive"
	"go-drive/server"
	"go-drive/storage"
	"log"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	common.InitConfig()
	config := common.GetConfig()

	componentsHolder := initComponentsHolder(config)

	engine, e := server.InitServer(componentsHolder, config.GetResDir())
	common.IfFatalError(e)

	log.Fatalln(engine.Run(config.GetListen()))
}

func initComponentsHolder(config common.Config) *server.ComponentsHolder {
	dbDialect, dbArg := config.GetDB()

	db, e := storage.InitDB(dbDialect, dbArg)
	common.IfFatalError(e)

	tokenStore := server.NewMemTokenStore(12*time.Hour, true, 1*time.Hour)

	requestSigner := common.NewSigner(common.RandString(32))

	driveStorage, e := storage.NewDriveStorage(db)
	common.IfFatalError(e)
	userStorage, e := storage.NewUserStorage(db)
	common.IfFatalError(e)
	groupStorage, e := storage.NewGroupStorage(db)
	common.IfFatalError(e)
	permissionStorage, e := storage.NewPathPermissionStorage(db)
	common.IfFatalError(e)
	pathMountStorage, e := storage.NewPathMountStorage(db)
	common.IfFatalError(e)
	driveDataStorage, e := storage.NewDriveDataStorage(db)
	common.IfFatalError(e)
	driveCacheStorage, e := storage.NewDriveCacheStorage(db)
	common.IfFatalError(e)
	rootDrive, e := drive.NewRootDrive(
		driveStorage, pathMountStorage,
		driveDataStorage, driveCacheStorage,
	)
	common.IfFatalError(e)

	chunksTempDir, e := config.GetDir("upload_temp", true)
	common.IfFatalError(e)
	chunkUploader, e := server.NewChunkUploader(chunksTempDir)
	common.IfFatalError(e)

	return &server.ComponentsHolder{
		TokenStore:    tokenStore,
		RequestSigner: requestSigner,

		RootDrive: rootDrive,

		DriveStorage: driveStorage,
		UserStorage:  userStorage,
		GroupStorage: groupStorage,

		PermissionStorage: permissionStorage,
		PathMountStorage:  pathMountStorage,
		DriveCacheStorage: driveCacheStorage,
		DriveDataStorage:  driveDataStorage,

		TaskRunner:    task.NewTunnyRunner(100),
		ChunkUploader: chunkUploader,
	}
}
