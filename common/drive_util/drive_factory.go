package drive_util

import (
	"go-drive/common"
	"go-drive/common/types"
)

type DriveUtils struct {
	Data        DriveDataStore
	CreateCache DriveCacheFactory
	Config      common.Config
}

type DriveCacheFactory = func(EntryDeserialize, EntrySerialize) DriveCache

type DriveInitConfig struct {
	Configured bool             `json:"configured"`
	OAuth      *OAuthInitConfig `json:"oauth"`

	Form  []types.FormItem `json:"form"`
	Value types.SM         `json:"value"`
}

type OAuthInitConfig struct {
	Url  string `json:"url"`
	Text string `json:"text"`

	Principal string `json:"principal"`
}

// DriveDataStore is a place to store drive's runtime data, such as token, refresh token.
type DriveDataStore interface {
	Save(types.SM) error
	Load(...string) (types.SM, error)
}

type DriveConfig = types.SM

type DriveFactory struct {
	// Create creates a drive instance by config map
	Create func(DriveConfig, DriveUtils) (types.IDrive, error)
	// InitConfig gets the initialization information.
	InitConfig func(DriveConfig, DriveUtils) (DriveInitConfig, error)
	// Init configures a drive's initial data.
	Init func(types.SM, DriveConfig, DriveUtils) error
}
