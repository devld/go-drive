package common

import (
	"errors"
	"flag"
	"fmt"
	"go-drive/common/registry"
	"go-drive/common/types"
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	version = "unknown"
	revHash = "unknown"
	buildAt = "unknown"
)

const (
	HeaderAuth        = "Authorization"
	SignatureQueryKey = "_k"
)

const (
	DbFilename = "data.db"
	LocalFsDir = "local"

	TempDir = "temp"

	DefaultListen            = ":8089"
	DefaultAPIPath           = ""
	DefaultWebPath           = ""
	DefaultDataDir           = "./"
	DefaultWebDir            = "./web"
	DefaultLangDir           = "./lang"
	DefaultLang              = "en-US"
	DefaultOAuthRedirectURI  = "https://go-drive.top/oauth_callback"
	DefaultMaxConcurrentTask = 100
	DefaultFreeFs            = false
	DefaultThumbnailTTL      = 30 * 24 * time.Hour
	DefaultAuthValidity      = 2 * time.Hour
	DefaultAuthAutoRefresh   = true
	DefaultWebDavPrefix      = "/dav"
	DefaultSearcher          = "bleve"

	DefaultConfigFile = "config.yml"
)

type Config struct {
	Listen string `yaml:"listen"`

	Db DbConfig `yaml:"db"`

	APIPath string `yaml:"api-path"`
	WebPath string `yaml:"web-path"`

	// all data will be stored in DataDir
	DataDir string `yaml:"data-dir"`
	TempDir string `yaml:"temp-dir"`
	// WebDir is the web ui static files dir
	WebDir string `yaml:"web-dir"`
	// LangDir is the i18n files dir
	LangDir string `yaml:"lang-dir"`
	// DefaultLang is the default language code
	DefaultLang string `yaml:"default-lang"`

	OAuthRedirectURI string `yaml:"oauth-redirect-uri"`

	MaxConcurrentTask int `yaml:"max-concurrent-task"`

	// unlimited fs drive path,
	// fs drive path will be limited in dataDir/local if freeFs is false
	FreeFs bool `yaml:"free-fs"`

	Thumbnail ThumbnailConfig `yaml:"thumbnail"`
	Auth      AuthConfig      `yaml:"auth"`

	WebDav WebDavConfig `yaml:"web-dav"`

	Search SearchConfig `yaml:"search"`

	Version string
	RevHash string
	BuildAt string
}

type DbConfig struct {
	Type     string   `yaml:"type"`
	Host     string   `yaml:"host"`
	Port     int      `yaml:"port"`
	User     string   `yaml:"user"`
	Password string   `yaml:"password"`
	Name     string   `yaml:"name"`
	Config   types.SM `yaml:"config"`
}

type ThumbnailConfig struct {
	TTL        time.Duration          `yaml:"ttl"`
	Concurrent int                    `yaml:"concurrent"`
	Handlers   []ThumbnailHandlerItem `yaml:"handlers"`
}

type ThumbnailHandlerItem struct {
	// Name is the unique name of the thumbnail handler
	Tags string `yaml:"tags"`
	// Type is handler type, available type are image, text, shell
	Type string `yaml:"type"`
	// FileTypes is supported file extensions separate by comm, folder type is /
	FileTypes string   `yaml:"file-types"`
	Config    types.SM `yaml:"config"`
}

type AuthConfig struct {
	Validity    time.Duration `yaml:"validity"`
	AutoRefresh bool          `yaml:"auto-refresh"`
}

type WebDavConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Prefix         string `yaml:"prefix"`
	AllowAnonymous bool   `yaml:"allow-anonymous"`
}

type SearchConfig struct {
	Enabled bool     `yaml:"enabled"`
	Type    string   `yaml:"type"`
	Config  types.SM `yaml:"config"`
}

func InitConfig(ch *registry.ComponentsHolder) (Config, error) {
	config := Config{
		Listen:            DefaultListen,
		APIPath:           DefaultAPIPath,
		WebPath:           DefaultWebPath,
		DataDir:           DefaultDataDir,
		WebDir:            DefaultWebDir,
		LangDir:           DefaultLangDir,
		DefaultLang:       DefaultLang,
		OAuthRedirectURI:  DefaultOAuthRedirectURI,
		MaxConcurrentTask: DefaultMaxConcurrentTask,
		FreeFs:            DefaultFreeFs,
		Thumbnail: ThumbnailConfig{
			TTL: DefaultThumbnailTTL,
		},
		Auth: AuthConfig{
			Validity:    DefaultAuthValidity,
			AutoRefresh: DefaultAuthAutoRefresh,
		},
		WebDav: WebDavConfig{
			Enabled: false,
			Prefix:  DefaultWebDavPrefix,
		},
		Search: SearchConfig{
			Type: DefaultSearcher,
		},

		Version: version,
		RevHash: revHash,
		BuildAt: buildAt,
	}

	v := flag.Bool("v", false, "print version")
	configFile := flag.String("c", "", "configuration file")
	showConfig := flag.Bool("show-config", false, "show parsed config")
	flag.Parse()

	if *v {
		fmt.Printf("%s rev %s build at %s\n", version, revHash, buildAt)
		os.Exit(0)
	}

	if *configFile == "" {
		_, e := os.Stat(DefaultConfigFile)
		if e == nil {
			*configFile = DefaultConfigFile
		}
	}

	if *configFile != "" {
		configBytes, e := ioutil.ReadFile(*configFile)
		if e != nil {
			return config, e
		}
		e = yaml.Unmarshal(configBytes, &config)
		if e != nil {
			return config, e
		}
	}

	if _, e := os.Stat(config.DataDir); os.IsNotExist(e) {
		return config, errors.New(fmt.Sprintf("data dir '%s' does not exist", config.DataDir))
	}

	if config.Thumbnail.Concurrent <= 0 {
		config.Thumbnail.Concurrent = int(math.Max(float64(runtime.NumCPU()/2), 1))
	}

	e := parseDbConfig(&config.Db)
	if e != nil {
		return config, e
	}

	if *showConfig {
		resolvedConf, _ := yaml.Marshal(config)
		fmt.Println(string(resolvedConf))
		os.Exit(0)
	}

	if config.TempDir == "" {
		tempDir, e := config.GetDir(TempDir, true)
		if e != nil {
			return config, e
		}
		config.TempDir = tempDir
	}

	ch.Add("config", config)
	ch.Add("versionSysConfig", versionSysConfig{})
	return config, nil
}

func parseDbConfig(c *DbConfig) error {
	if c.Type == "" {
		c.Type = "sqlite"
	}
	switch c.Type {
	case "sqlite":
		if c.Name == "" {
			c.Name = DbFilename
		}
	case "mysql":
		if c.Port <= 0 {
			c.Port = 3306
		}
		if c.Host == "" {
			return errors.New("mysql host is required")
		}
		if c.Name == "" {
			return errors.New("mysql database name is required")
		}
	default:
		return errors.New("unsupported db type: " + c.Type)
	}
	return nil
}

func (c Config) GetDB() gorm.Dialector {
	db := c.Db
	var d gorm.Dialector = nil
	switch db.Type {
	case "sqlite":
		d = sqlite.Open(path.Join(c.DataDir, db.Name))
	case "mysql":
		params, _ := url.ParseQuery("charset=utf8mb4&parseTime=True&loc=Local")
		if db.Config != nil {
			for k, v := range db.Config {
				params.Set(k, v)
			}
		}
		d = mysql.Open(fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?%s",
			db.User, db.Password, db.Host, db.Port, db.Name, params.Encode(),
		))
	default:
		panic("invalid db type: " + db.Type)
	}
	return d
}

func (c Config) GetDir(name string, create bool) (string, error) {
	name = filepath.Join(c.DataDir, name)
	if create {
		if _, e := os.Stat(name); os.IsNotExist(e) {
			if e := os.Mkdir(name, 0755); e != nil {
				return "", e
			}
		}
	}
	return name, nil
}

func (c Config) GetTempDir(name string, create bool) (string, error) {
	name = filepath.Join(c.TempDir, name)
	if create {
		if _, e := os.Stat(name); os.IsNotExist(e) {
			if e := os.Mkdir(name, 0755); e != nil {
				return "", e
			}
		}
	}
	return name, nil
}

func (c Config) GetLocalFsDir() (string, error) {
	if c.FreeFs {
		return "", nil
	}
	return c.GetDir(LocalFsDir, true)
}

type versionSysConfig struct {
}

func (v versionSysConfig) SysConfig() (string, types.M, error) {
	return "version", types.M{
		"version": version,
		"rev":     revHash,
		"buildAt": buildAt,
	}, nil
}
