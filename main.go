package main

import (
	"go-drive/common"
	"go-drive/server"
	"go-drive/storage"
	"go-drive/storage/db"
)

func main() {
	common.InitConfig()

	common.PanicIfError(db.InitDB())
	common.PanicIfError(storage.InitRootDrive())
	common.PanicIfError(server.InitServer())
}
