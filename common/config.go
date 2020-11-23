package common

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"time"
)

const (
	DbType     = "sqlite3"
	DbFilename = "data.db"
	LocalFsDir = "local"
	Listen     = ":8089"

	DefaultMaxProxySize = 1 * 1024 * 1024

	DefaultThumbnailMaxSize    = 16
	DefaultThumbnailCacheTTL   = "48h"
	DefaultThumbnailConcurrent = 16
	DefaultThumbnailMaxPixels  = 22369621

	DefaultMaxConcurrentTask = 100

	DefaultTokenValidity = "2h"
)

func InitConfig(ch *ComponentsHolder) (Config, error) {
	config := Config{}
	flag.StringVar(&config.Listen, "l", Listen,
		"address listen on")
	flag.StringVar(&config.dataDir, "d", "./",
		"path to the data dir")
	flag.StringVar(&config.resDir, "s", "",
		"path to the static files")
	flag.BoolVar(&config.freeFs, "f", false,
		"enable unlimited local fs drive(absolute path)")

	flag.Int64Var(&config.ProxyMaxSize, "proxy-max-size", DefaultMaxProxySize,
		"maximum file size that can be proxied")

	flag.Int64Var(&config.maxThumbnailSize, "thumbnail-max-size", DefaultThumbnailMaxSize,
		"maximum file size to create thumbnail")
	flag.IntVar(&config.ThumbnailMaxPixels, "thumbnail-max-pixels", DefaultThumbnailMaxPixels,
		"maximum pixels(W*H) of original image to thumbnails")
	flag.IntVar(&config.ThumbnailConcurrent, "thumbnail-concurrent", DefaultThumbnailConcurrent,
		"maximum number of concurrent creation of thumbnails")
	var tcTtl string
	flag.StringVar(&tcTtl, "thumbnail-cache-ttl", DefaultThumbnailCacheTTL,
		"thumbnail cache validity, valid time units are \"s\", \"m\", \"h\".")

	flag.IntVar(&config.MaxConcurrentTask, "max-concurrent-task", DefaultMaxConcurrentTask,
		"maximum concurrent task(copy, move, upload, delete files)")

	var tokenTtl string
	flag.StringVar(&tokenTtl, "token-validity", DefaultTokenValidity, "token validity, valid time units are \"s\", \"m\", \"h\".")
	flag.BoolVar(&config.TokenRefresh, "token-refresh", true, "enable auto refresh token")

	flag.Parse()

	config.ThumbnailCacheTTl = parseDuration(tcTtl, 1, 48*time.Hour)
	config.ThumbnailConcurrent = parseInt(config.ThumbnailConcurrent, 1, DefaultThumbnailConcurrent)

	config.TokenValidity = parseDuration(tokenTtl, 1, 2*time.Hour)

	if exists, _ := FileExists(config.dataDir); !exists {
		return config, errors.New(fmt.Sprintf("dataDir '%s' does not exist", config.dataDir))
	}
	tempDir, e := config.GetDir("temp", true)
	if e != nil {
		return config, e
	}
	config.TempDir = tempDir

	ch.Add("config", config)
	return config, nil
}

type Config struct {
	Listen  string
	dataDir string
	// static files(web) dir
	resDir string
	// unlimited fs drive path,
	// fs drive path will be limited in dataDir/local if freeFs is false
	freeFs bool

	TempDir string

	// ProxyMaxSize is the maximum file size can be proxied when
	// the API call explicitly specifies
	// that it needs to be proxied.
	// The size is unlimited when maxProxySize is <= 0
	ProxyMaxSize int64

	// maxThumbnailSize is the maximum file size(MB) to create thumbnail
	maxThumbnailSize    int64
	ThumbnailCacheTTl   time.Duration
	ThumbnailConcurrent int
	ThumbnailMaxPixels  int

	MaxConcurrentTask int

	TokenValidity time.Duration
	TokenRefresh  bool
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

func (c Config) GetThumbnailMaxSize() int64 {
	return c.maxThumbnailSize * 1024 * 1024
}

func parseDuration(d string, min time.Duration, def time.Duration) time.Duration {
	r, e := time.ParseDuration(d)
	if e != nil || r < min {
		return def
	}
	return r
}

func parseInt(val int, min int, def int) int {
	if val < min {
		return def
	}
	return val
}
