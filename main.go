package main

import (
	"go-drive/common"
	"go-drive/server"
	"go-drive/storage"
	"time"
)

func main() {
	config := common.InitConfig()
	dbDialect, dbArg := config.GetDB()

	db, e := storage.InitDB(dbDialect, dbArg)
	common.PanicIfError(e)

	tokenStore := server.NewMemTokenStore(12*time.Hour, true, 1*time.Hour)

	driveStorage, e := storage.NewDriveStorage(db)
	common.PanicIfError(e)
	userStorage, e := storage.NewUserStorage(db)
	common.PanicIfError(e)

	engine, e := server.InitServer(
		tokenStore,
		driveStorage,
		userStorage,
	)
	common.PanicIfError(e)

	panic(engine.Run(config.GetListen()))
}
