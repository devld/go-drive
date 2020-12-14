package drive_util

import (
	"context"
	"go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
	"time"
)

type OAuthRequest struct {
	Endpoint       oauth2.Endpoint
	RedirectURL    string
	Scopes         []string
	Text           string
	AutoCodeOption []oauth2.AuthCodeOption
}

type OAuthResponse struct {
	Token  *oauth2.Token
	Config *oauth2.Config
}

func (o *OAuthResponse) Client(ctx context.Context) *http.Client {
	if ctx == nil {
		ctx = context.Background()
	}
	return o.Config.Client(ctx, o.Token)
}

func (o *OAuthResponse) TokenSource(ctx context.Context) oauth2.TokenSource {
	if ctx == nil {
		ctx = context.Background()
	}
	return o.Config.TokenSource(ctx, o.Token)
}

func getOAuthConfig(o OAuthRequest, config DriveConfig) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     config["client_id"],
		ClientSecret: config["client_secret"],
		Endpoint:     o.Endpoint,
		RedirectURL:  o.RedirectURL,
		Scopes:       o.Scopes,
	}
}

func oauthGet(o OAuthRequest, config DriveConfig, ds DriveDataStore) (*OAuthResponse, error) {
	params, e := ds.Load("token", "token_type", "expires_at", "refresh_token")
	if e != nil {
		return nil, e
	}
	expiresAt := time.Unix(utils.ToInt64(params["expires_at"], -1), 0)
	t := &oauth2.Token{
		AccessToken:  params["token"],
		TokenType:    params["token_type"],
		RefreshToken: params["refresh_token"],
		Expiry:       expiresAt,
	}
	if t.AccessToken == "" {
		t = nil
	}
	if t != nil && t.RefreshToken == "" && expiresAt.Before(time.Now()) {
		t = nil
	}
	return &OAuthResponse{Config: getOAuthConfig(o, config), Token: t}, nil
}

func OAuthInitConfig(o OAuthRequest, config DriveConfig,
	ds DriveDataStore) (*DriveInitConfig, *OAuthResponse, error) {
	resp, e := oauthGet(o, config, ds)
	if e != nil {
		return nil, nil, e
	}

	state := utils.RandString(6)
	if e := ds.Save(types.SM{"state": state}); e != nil {
		return nil, nil, e
	}
	initConfig := &DriveInitConfig{
		OAuth: &OAuthConfig{
			Url:  resp.Config.AuthCodeURL(state, o.AutoCodeOption...),
			Text: o.Text,
		},
	}
	if resp.Token != nil {
		initConfig.Configured = true
		return initConfig, resp, nil
	}

	return initConfig, resp, nil
}

func OAuthInit(ctx context.Context, o OAuthRequest, data types.SM,
	config DriveConfig, ds DriveDataStore) (*OAuthResponse, error) {
	code := data["code"]
	state := data["state"]

	if code == "" {
		return nil, nil
	}

	oauthConf := getOAuthConfig(o, config)

	params, e := ds.Load("state")
	if e != nil {
		return nil, e
	}
	if state != params["state"] {
		return nil, err.NewNotAllowedMessageError(i18n.T("oauth.state_mismatch"))
	}
	t, e := oauthConf.Exchange(ctx, code)
	if e != nil {
		return nil, e
	}
	return &OAuthResponse{Config: oauthConf, Token: t},
		ds.Save(types.SM{
			"state":         "",
			"token":         t.AccessToken,
			"token_type":    t.TokenType,
			"refresh_token": t.RefreshToken,
			"expires_at":    strconv.FormatInt(t.Expiry.Unix(), 10),
		})
}

func OAuthGet(o OAuthRequest, config DriveConfig, ds DriveDataStore) (*OAuthResponse, error) {
	resp, e := oauthGet(o, config, ds)
	if e != nil {
		return nil, e
	}
	if resp.Token == nil {
		return nil, err.NewNotAllowedMessageError("drive.not_configured")
	}
	return resp, e
}
