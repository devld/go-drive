//+build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"go-drive/common"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/drive"
	"go-drive/server"
	"go-drive/storage"
)

func Initialize(ch *common.ComponentsHolder) (*gin.Engine, error) {
	wire.Build(
		common.InitConfig,
		storage.NewDB,
		storage.NewUserDAO,
		storage.NewPathPermissionDAO,
		storage.NewDriveCacheDAO,
		storage.NewGroupDAO,
		storage.NewPathMountDAO,
		storage.NewDriveDAO,
		storage.NewDriveDataDAO,
		wire.Bind(new(task.Runner), new(*task.TunnyRunner)),
		task.NewTunnyRunner,
		common.NewSigner,
		wire.Bind(new(types.TokenStore), new(*server.FileTokenStore)),
		server.NewFileTokenStore,
		server.NewChunkUploader,
		server.NewThumbnail,
		drive.NewRootDrive,
		server.InitServer,
	)
	return &gin.Engine{}, nil
}
