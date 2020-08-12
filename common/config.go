package common

import (
	"flag"
	"path"
)

const (
	DbFilename = "data.db"
)

var (
	dataDir string
	listen  string
)

func InitConfig() {
	flag.StringVar(&listen, "l", ":8089", "port listen on")
	flag.StringVar(&dataDir, "d", "./", "path to the db files dir")

	flag.Parse()

	if exists, _ := FileExists(dataDir); !exists {
		panic("dataDir '" + dataDir + "' does not exist")
	}
}

func GetListen() string {
	return listen
}

func GetDBFile() string {
	return path.Join(dataDir, DbFilename)
}
