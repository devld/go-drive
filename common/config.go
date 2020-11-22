package common

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path"
	"time"
)

const (
	DbType                     = "sqlite3"
	DbFilename                 = "data.db"
	LocalFsDir                 = "local"
	DefaultMaxProxySize        = 1 * 1024 * 1024
	DefaultMaxThumbnailSize    = 16
	DefaultThumbnailCacheTTL   = "12h"
	DefaultThumbnailConcurrent = 50
)

func init() {
	R().Register("config", func(c *ComponentRegistry) interface{} {
		config := Config{}
		flag.StringVar(&config.listen, "l", ":8089", "port listen on")
		flag.StringVar(&config.dataDir, "d", "./", "path to the db files dir")
		flag.StringVar(&config.resDir, "s", "", "path to the static files")
		flag.BoolVar(&config.freeFs, "f", false, "enable unlimited local fs drive(absolute path)")
		flag.Int64Var(&config.maxProxySize, "max-proxy-size", DefaultMaxProxySize, "maximum file size that can be proxied")
		flag.Int64Var(&config.maxThumbnailSize, "max-thumbnail-size", DefaultMaxThumbnailSize, "maximum file size to create thumbnail")
		flag.IntVar(&config.ThumbnailConcurrent, "thumbnail-concurrent", DefaultThumbnailConcurrent, "maximum number of concurrent creation of thumbnails")
		var tcTtl string
		flag.StringVar(&tcTtl, "thumbnail-cache-ttl", DefaultThumbnailCacheTTL, "thumbnail cache validity, valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\".")

		flag.Parse()

		duration, _ := time.ParseDuration(tcTtl)
		if duration <= 0 {
			duration = 12 * time.Hour
		}
		config.ThumbnailCacheTTl = duration
		if config.ThumbnailConcurrent <= 0 {
			config.ThumbnailConcurrent = DefaultThumbnailConcurrent
		}

		if exists, _ := FileExists(config.dataDir); !exists {
			panic(fmt.Sprintf("dataDir '%s' does not exist", config.dataDir))
		}
		return config
	}, math.MinInt32)
}

func Conf() Config {
	return R().Get("config").(Config)
}

type Config struct {
	dataDir string
	listen  string
	// static files(web) dir
	resDir string
	// unlimited fs drive path,
	// fs drive path will be limited in dataDir/local if freeFs is false
	freeFs bool
	// maxProxySize is the maximum file size can be proxied when
	// the API call explicitly specifies
	// that it needs to be proxied.
	// The size is unlimited when maxProxySize is <= 0
	maxProxySize int64

	// maxThumbnailSize is the maximum file size(MB) to create thumbnail
	maxThumbnailSize    int64
	ThumbnailCacheTTl   time.Duration
	ThumbnailConcurrent int
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

func (c Config) GetMaxProxySize() int64 {
	return c.maxProxySize
}

func (c Config) GetMaxThumbnailSize() int64 {
	return c.maxThumbnailSize * 1024 * 1024
}
