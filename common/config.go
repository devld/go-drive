package common

import (
	"errors"
	"flag"
	"fmt"
	"go-drive/common/registry"
	"os"
	"path"
	"path/filepath"
	"time"
)

var (
	version = "unknown"
	hash    = "unknown"
	build   = "unknown"
)

const (
	DbType     = "sqlite3"
	DbFilename = "data.db"
	LocalFsDir = "local"
	Listen     = ":8089"
)

func InitConfig(ch *registry.ComponentsHolder) (Config, error) {
	config := Config{}

	var v bool
	flag.BoolVar(&v, "v", false, "print version")

	flag.StringVar(&config.Listen, "l", Listen, "address listen on")
	flag.StringVar(&config.dataDir, "d", "./", "path to the data dir")
	flag.StringVar(&config.resDir, "s", "", "path to the static files")
	flag.BoolVar(&config.freeFs, "f", false, "enable unlimited local fs drive(absolute path)")
	flag.StringVar(&config.DefaultLang, "lang", "en-US", "default language code")

	flag.Int64Var(&config.ProxyMaxSize, "proxy-max-size", 1*1024*1024, "maximum file size that can be proxied")

	flag.Int64Var(&config.ThumbnailMaxSize, "thumbnail-max-size", 16*1024*1024, "maximum file size to create thumbnail")
	flag.IntVar(&config.ThumbnailMaxPixels, "thumbnail-max-pixels", 22369621, "maximum pixels(W*H) of original image to thumbnails")
	flag.IntVar(&config.ThumbnailConcurrent, "thumbnail-concurrent", 16, "maximum number of concurrent creation of thumbnails")
	flag.DurationVar(&config.ThumbnailCacheTTl, "thumbnail-cache-ttl", 48*time.Hour, "thumbnail cache validity")

	flag.IntVar(&config.MaxConcurrentTask, "max-concurrent-task", 100, "maximum concurrent task(copy, move, upload, delete files)")

	flag.DurationVar(&config.TokenValidity, "token-validity", 2*time.Hour, "token validity")
	flag.BoolVar(&config.TokenRefresh, "token-refresh", true, "enable auto refresh token")

	flag.Parse()

	if v {
		fmt.Printf("%s %s build-%s", version, hash, build)
		os.Exit(0)
	}

	if _, e := os.Stat(config.dataDir); os.IsNotExist(e) {
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

	// DefaultLang is the default language
	DefaultLang string

	TempDir string

	// ProxyMaxSize is the maximum file size can be proxied when
	// the API call explicitly specifies
	// that it needs to be proxied.
	// The size is unlimited when maxProxySize is <= 0
	ProxyMaxSize int64

	// ThumbnailMaxSize is the maximum file size(MB) to create thumbnail
	ThumbnailMaxSize    int64
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
	name = filepath.Join(c.dataDir, name)
	if create {
		if _, e := os.Stat(name); os.IsNotExist(e) {
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
