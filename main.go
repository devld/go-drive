package main

import (
	"github.com/gin-gonic/gin"
	"go-drive/drive"
	"go-drive/server"
)

func main() {

	localDrive, e := drive.NewFsDrive("D:\\data\\Temp\\test\\drive-local")
	localDrive2, e := drive.NewFsDrive("D:\\coding")

	if e != nil {
		panic(e)
	}
	d := drive.NewDrive()
	d.AddDrive("local", localDrive)
	d.AddDrive("local2", localDrive2)

	dr := server.NewDriveRoute(d)

	r := gin.Default()
	dr.Init(r)

	panic(r.Run(":8089"))

}
