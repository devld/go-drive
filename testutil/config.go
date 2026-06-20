package testutil

import (
	"fmt"
	"go-drive/common"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	sharedOnce    sync.Once
	sharedConfig  common.Config
	sharedCleanup func()
)

// GetSharedTestConfig returns a test config that uses a temporary directory for
// DataDir and TempDir (e.g. os.TempDir()/go-drive-ut-<timestamp>/data and .../temp).
// The same config is shared for the whole test process (e.g. all tests in a package).
// The caller (e.g. TestMain) must call the returned cleanup after tests finish.
func GetSharedTestConfig() (common.Config, func()) {
	sharedOnce.Do(func() {
		base := filepath.Join(os.TempDir(), fmt.Sprintf("go-drive-ut-%d", time.Now().UnixNano()))
		dataDir := filepath.Join(base, "data")
		tempDir := filepath.Join(base, "temp")
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			panic("testutil: create data dir: " + err.Error())
		}
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			_ = os.RemoveAll(base)
			panic("testutil: create temp dir: " + err.Error())
		}
		sharedConfig = defaultTestConfigWithDirs(dataDir, tempDir)
		sharedCleanup = func() { _ = os.RemoveAll(base) }
	})
	return sharedConfig, sharedCleanup
}

func defaultTestConfigWithDirs(dataDir, tempDir string) common.Config {
	// DB file lives under DataDir, cleaned with the temp root
	return common.Config{
		Listen:      common.DefaultListen,
		APIPath:     common.DefaultAPIPath,
		WebPath:     common.DefaultWebPath,
		DataDir:     dataDir,
		TempDir:     tempDir,
		WebDir:      common.DefaultWebDir,
		LangDir:     common.DefaultLangDir,
		DefaultLang: common.DefaultLang,

		DrivesDir:          common.DefaultDrivesDir,
		DriveUploadersDir:  common.DefaultDriveUploadersDir,
		DriveRepositoryURL: common.DefaultDriveRepositoryURL,

		OAuthRedirectURI:  common.DefaultOAuthRedirectURI,
		MaxConcurrentTask: common.DefaultMaxConcurrentTask,
		FreeFs:            common.DefaultFreeFs,
		Thumbnail: common.ThumbnailConfig{
			TTL: common.DefaultThumbnailTTL,
		},
		Auth: common.AuthConfig{
			Validity:    common.DefaultAuthValidity,
			AutoRefresh: common.DefaultAuthAutoRefresh,
		},
		SignatureTTL: common.DefaultSignatureTTL,
		WebDav: common.WebDavConfig{
			Enabled:       false,
			Prefix:        common.DefaultWebDavPrefix,
			MaxCacheItems: common.DefaultWebDavMaxCacheItems,
		},
		Search: common.SearchConfig{
			Type: common.DefaultSearcher,
		},
		Cache: common.CacheConfig{
			Type:        common.DefaultCacheType,
			CleanPeriod: common.DefaultCacheCleanPeriod,
		},

		Version: common.Version,
		RevHash: common.RevHash,
		BuildAt: common.BuildAt,

		Db: common.DbConfig{
			Type: "sqlite",
			Name: "file::memory:?cache=shared",
		},
	}
}

// DefaultTestConfig returns a fixed config for unit tests (shared temp dir when
// GetSharedTestConfig was used in TestMain). DB is test.db under DataDir.
// For per-package shared config and cleanup, use GetSharedTestConfig in TestMain
// and use the returned config in tests.
func DefaultTestConfig() common.Config {
	cfg, _ := GetSharedTestConfig()
	return cfg
}
