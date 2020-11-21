package main

import (
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"log"
	"math/rand"
	"time"

	_ "go-drive/common"
	_ "go-drive/common/task"
	_ "go-drive/server"
	_ "go-drive/storage"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	common.R().Init()

	config := common.R().Get("config").(common.Config)

	log.Fatalln(
		common.R().Get("httpServer").(*gin.Engine).Run(config.GetListen()),
	)
}
