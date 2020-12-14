package drive_util

import (
	"context"
	"go-drive/common"
	"go-drive/common/types"
)

type DriveFactoryConfig struct {
	Type        string           `json:"type"`
	DisplayName string           `json:"display_name" i18n:""`
	README      string           `json:"readme" i18n:""`
	ConfigForm  []types.FormItem `json:"config_form"`
	Factory     DriveFactory     `json:"-"`
}

type DriveInitConfig struct {
	Configured bool         `json:"configured"`
	OAuth      *OAuthConfig `json:"oauth"`

	Form  []types.FormItem `json:"form"`
	Value types.SM         `json:"value"`
}

type OAuthConfig struct {
	Url  string `json:"url"`
	Text string `json:"text" i18n:""`

	Principal string `json:"principal"`
}

type DriveConfig = types.SM

type DriveCacheFactory = func(EntryDeserialize, EntrySerialize) DriveCache

// DriveDataStore is a place to store drive's runtime data, such as token, refresh token.
type DriveDataStore interface {
	Save(types.SM) error
	Load(...string) (types.SM, error)
}

type DriveUtils struct {
	Data        DriveDataStore
	CreateCache DriveCacheFactory
	Config      common.Config
}

type DriveFactory struct {
	// InitConfig gets the initialization information.
	InitConfig func(context.Context, DriveConfig, DriveUtils) (*DriveInitConfig, error)
	// Init configures a drive's initial data.
	Init func(context.Context, types.SM, DriveConfig, DriveUtils) error
	// Create creates a drive instance by config map
	Create func(context.Context, DriveConfig, DriveUtils) (types.IDrive, error)
}
