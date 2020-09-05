package common

import (
	"flag"
	"path"
)

const (
	DbType     = "sqlite3"
	DbFilename = "data.db"
)

type Config struct {
	dataDir string
	listen  string
}

func InitConfig() Config {
	c := Config{}
	flag.StringVar(&c.listen, "l", ":8089", "port listen on")
	flag.StringVar(&c.dataDir, "d", "./", "path to the db files dir")

	flag.Parse()

	if exists, _ := FileExists(c.dataDir); !exists {
		panic("dataDir '" + c.dataDir + "' does not exist")
	}
	return c
}

func (c Config) GetListen() string {
	return c.listen
}

func (c Config) GetDB() (string, string) {
	return DbType, path.Join(c.dataDir, DbFilename)
}
