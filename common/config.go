package common

import (
	"flag"
	"os"
	"path"
)

const (
	DbType     = "sqlite3"
	DbFilename = "data.db"
)

type Config struct {
	dataDir string
	listen  string
	resDir  string
}

func InitConfig() Config {
	c := Config{}
	flag.StringVar(&c.listen, "l", ":8089", "port listen on")
	flag.StringVar(&c.dataDir, "d", "./", "path to the db files dir")
	flag.StringVar(&c.resDir, "s", "", "path to the static files")

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

func (c Config) GetDir(name string, create bool) (string, error) {
	name = path.Join(c.dataDir, name)
	if create {
		exists, e := FileExists(name)
		if e != nil {
			return "", e
		}
		if !exists {
			if e := os.Mkdir(name, 0755); e != nil {
				return "", e
			}
		}
	}
	return name, nil
}

func (c Config) GetResDir() string {
	return c.resDir
}
