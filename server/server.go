package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/storage"
)

func InitServer() error {
	engine := gin.New()

	newDriveRoute(storage.GetRootDrive()).init(engine)

	return engine.Run(common.GetListen())
}
