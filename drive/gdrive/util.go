package gdrive

import (
	"context"
	"errors"
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/i18n"
	"go-drive/common/req"
	"go-drive/common/types"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	gOauth "google.golang.org/api/oauth2/v1"
	"google.golang.org/api/option"
)

var t = i18n.TPrefix("drive.gdrive.")

const (
	typeFolder = "application/vnd.google-apps.folder"

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

func oauthReq(c common.Config) *driveutil.OAuthRequest {
	return &driveutil.OAuthRequest{
		Endpoint:       google.Endpoint,
		RedirectURL:    c.OAuthRedirectURI,
		Scopes:         []string{"https://www.googleapis.com/auth/drive", "https://www.googleapis.com/auth/userinfo.profile"},
		Text:           t("oauth_text"),
		AutoCodeOption: []oauth2.AuthCodeOption{oauth2.AccessTypeOffline, oauth2.ApprovalForce},
	}
}

func InitConfig(ctx context.Context, config types.SM,
	env driveutil.DriveEnv) (*driveutil.DriveInitConfig, error) {
	initConfig, oauthHolder, e := driveutil.OAuthInitConfig(*oauthReq(env.Config),
		driveutil.OAuthCredentials{
			ClientID:     config["client_id"],
			ClientSecret: config["client_secret"],
		}, env.Data)
	if e != nil {
		return nil, e
	}
	if oauthHolder == nil {
		return initConfig, nil
	}
	httpClient := req.WithDefaultRoundTripper(oauthHolder.Client())
	service, e := gOauth.NewService(ctx, option.WithHTTPClient(httpClient))
	if e != nil {
		return nil, e
	}

	// get user info
	user, e := service.Userinfo.V2.Me.Get().Context(ctx).Do()
	initConfig.Configured = e == nil
	if e == nil {
		initConfig.OAuth.Principal = user.Name
		if e := buildInitForm(ctx, oauthHolder, env, initConfig); e != nil {
			return nil, e
		}
	}

	return initConfig, nil
}

func buildInitForm(ctx context.Context, oauthHolder *driveutil.OAuthHolder,
	driveEnv driveutil.DriveEnv, initConfig *driveutil.DriveInitConfig) error {
	// get shared drives
	driveSrv, e := drive.NewService(ctx, option.WithHTTPClient(req.WithDefaultRoundTripper(oauthHolder.Client())))
	if e != nil {
		return e
	}
	drivesResp, e := driveSrv.Drives.List().Context(ctx).Do()
	if e != nil {
		return e
	}

	params, e := driveEnv.Data.Load("drive_id")
	if e != nil {
		return e
	}

	opts := make([]types.FormItemOption, 0, len(drivesResp.Drives)+1)
	opts = append(opts, types.FormItemOption{
		Name:  t("my_drive_name"),
		Title: t("my_drive_name"),
		Value: "",
	})
	for _, d := range drivesResp.Drives {
		opts = append(opts, types.FormItemOption{
			Name:  d.Name,
			Title: d.Name,
			Value: d.Id,
		})
	}

	initConfig.Form = []types.FormItem{
		{Label: t("drive_label"), Type: "select", Field: "drive_id", Options: &opts, DefaultValue: ""},
	}
	initConfig.Value = types.SM{"drive_id": params["drive_id"]}

	return nil
}

func Init(ctx context.Context, data types.SM,
	config types.SM, env driveutil.DriveEnv) error {
	if e := env.Data.Save(types.SM{"drive_id": data["drive_id"]}); e != nil {
		return e
	}
	_, e := driveutil.OAuthInit(ctx, *oauthReq(env.Config), data,
		driveutil.OAuthCredentials{
			ClientID:     config["client_id"],
			ClientSecret: config["client_secret"],
		}, env.Data)
	return e
}

func (g *GDrive) deserializeEntry(ci driveutil.EntryCacheItem) (types.IEntry, error) {
	id := ci.Data["i"]
	if id == "" {
		return nil, errors.New("")
	}
	return &gdriveEntry{
		id: id, mime: ci.Data["m"], path: ci.Path, isDir: ci.Type.IsDir(),
		size: ci.Size, modTime: ci.ModTime, d: g,
		targetID: ci.Data["ti"], targetMime: ci.Data["tm"],
		thumbnail: ci.Data["th"],
	}, nil
}
