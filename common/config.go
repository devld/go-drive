package common

import (
	"flag"
	"log"
	"os"
	"path"
)

const (
	DbType     = "sqlite3"
	DbFilename = "data.db"
	LocalFsDir = "local"
)

var config *Config

func GetConfig() Config {
	if config == nil {
		log.Fatalln("Configuration is not initialized")
	}
	return *config
}

func InitConfig() {
	if config != nil {
		log.Fatalln("Configuration has been initialized")
	}
	c := Config{}
	flag.StringVar(&c.listen, "l", ":8089", "port listen on")
	flag.StringVar(&c.dataDir, "d", "./", "path to the db files dir")
	flag.StringVar(&c.resDir, "s", "", "path to the static files")
	flag.BoolVar(&c.freeFs, "f", false, "enable unlimited local fs drive(absolute path)")

	flag.Parse()

	if exists, _ := FileExists(c.dataDir); !exists {
		log.Fatalf("dataDir '%s' does not exist", c.dataDir)
	}

	config = &c
}

type Config struct {
	dataDir string
	listen  string
	resDir  string
	freeFs  bool
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

func (c Config) GetLocalFsDir() (string, error) {
	if c.freeFs {
		return "", nil
	}
	return c.GetDir(LocalFsDir, true)
}
