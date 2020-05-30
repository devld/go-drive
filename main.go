package main

import (
	"github.com/gin-gonic/gin"
	"go-drive/config"
	"go-drive/drive"
	"go-drive/server"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		_, _ = os.Stderr.WriteString("Usage: " + os.Args[0] + " <config file>\n")
		os.Exit(1)
	}
	conf, e := config.LoadConfig(os.Args[1])
	if e != nil {
		panic(e)
	}

	d := drive.NewDrive()

	for _, local := range conf.Locals {
		log.Println("Loading local drive [" + local.Name + "] " + local.Path)
		localDrive, e := drive.NewFsDrive(local.Path)
		if e != nil {
			panic(e)
		}
		d.AddDrive(local.Name, localDrive)
	}

	dr := server.NewDriveRoute(d)

	r := gin.Default()
	dr.Init(r)

	panic(r.Run(conf.Listen))

}
