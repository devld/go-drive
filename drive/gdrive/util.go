package gdrive

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common/drive_util"
	"go-drive/common/types"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gOauth "google.golang.org/api/oauth2/v1"
	"google.golang.org/api/option"
)

const (
	typeFolder   = "application/vnd.google-apps.folder"
	typeShortcut = "application/vnd.google-apps.shortcut"

	typeGoogleAppPrefix = "application/vnd.google-apps."
)

// see https://developers.google.com/drive/api/v3/ref-export-formats
// and https://developers.google.com/drive/api/v3/mime-types
var exportMimeTypeMap = map[string]string{
	"application/vnd.google-apps.document":     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.google-apps.spreadsheet":  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"application/vnd.google-apps.presentation": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"application/vnd.google-apps.drawing":      "image/svg+xml",
	"application/vnd.google-apps.script":       "application/vnd.google-apps.script+json",
}
var mimeTypeExtensionsMap = map[string]string{
	"application/vnd.google-apps.document":     "docx",
	"application/vnd.google-apps.spreadsheet":  "xlsx",
	"application/vnd.google-apps.presentation": "pptx",
	"application/vnd.google-apps.drawing":      "svg",
	"application/vnd.google-apps.script":       "json",
}

var oauth = drive_util.OAuthRequest{
	Endpoint:       google.Endpoint,
	RedirectURL:    drive_util.CommonRedirectURL,
	Scopes:         []string{"https://www.googleapis.com/auth/drive", "https://www.googleapis.com/auth/userinfo.profile"},
	Text:           "Connect to Google Drive",
	AutoCodeOption: []oauth2.AuthCodeOption{oauth2.AccessTypeOffline},
}

func InitConfig(config drive_util.DriveConfig, utils drive_util.DriveUtils) (*drive_util.DriveInitConfig, error) {
	initConfig, resp, e := drive_util.OAuthInitConfig(oauth, config, utils.Data)
	if e != nil {
		return nil, e
	}
	if resp == nil {
		return initConfig, nil
	}
	httpClient := resp.Client(nil)
	service, e := gOauth.NewService(context.Background(), option.WithHTTPClient(httpClient))
	if e != nil {
		return nil, e
	}

	// get user info
	user, e := service.Userinfo.V2.Me.Get().Do()
	initConfig.Configured = e == nil
	if e == nil {
		initConfig.OAuth.Principal = fmt.Sprintf("%s", user.Name)
	}

	return initConfig, nil
}

func Init(data types.SM, config drive_util.DriveConfig, utils drive_util.DriveUtils) error {
	_, e := drive_util.OAuthInit(oauth, data, config, utils.Data)
	return e
}

func (g *GDrive) deserializeEntry(dat string) (types.IEntry, error) {
	ci, e := drive_util.DeserializeEntry(dat)
	if e != nil {
		return nil, e
	}
	id := ci.Data["i"]
	if id == "" {
		return nil, errors.New("")
	}
	return &gdriveEntry{
		id: id, mime: ci.Data["m"], path: ci.Path, isDir: ci.Type.IsDir(),
		size: ci.Size, modTime: ci.ModTime, d: g,
		targetId: ci.Data["ti"], targetMime: ci.Data["tm"],
		thumbnail: ci.Data["th"],
	}, nil
}
