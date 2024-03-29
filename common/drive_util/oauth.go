package drive_util

import (
	"context"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

const (
	DsKeyToken        = "token"
	DsKeyTokenType    = "token_type"
	DsKeyExpiresAt    = "expires_at"
	DsKeyRefreshToken = "refresh_token"
	DsKeyState        = "state"
)

type OAuthCredentials struct {
	ClientID     string
	ClientSecret string
}

type OAuthRequest struct {
	Endpoint       oauth2.Endpoint
	RedirectURL    string
	Scopes         []string
	Text           string
	AutoCodeOption []oauth2.AuthCodeOption
}

type OAuthResponse struct {
	ts oauth2.TokenSource
}

func newOAuthResponse(config *oauth2.Config, ds DriveDataStore, token *oauth2.Token) *OAuthResponse {
	ts := &tokenSource{
		ts: config.TokenSource(context.Background(), token),
		ds: ds,
		mu: sync.Mutex{},
	}
	return &OAuthResponse{ts}
}

func (o *OAuthResponse) Client() *http.Client {
	return oauth2.NewClient(context.Background(), o.TokenSource())
}

func (o *OAuthResponse) TokenSource() oauth2.TokenSource {
	return o.ts
}

func (o *OAuthResponse) Token() (*oauth2.Token, error) {
	return o.ts.Token()
}

func oAuthConfig(o OAuthRequest, cred OAuthCredentials) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cred.ClientID,
		ClientSecret: cred.ClientSecret,
		Endpoint:     o.Endpoint,
		RedirectURL:  o.RedirectURL,
		Scopes:       o.Scopes,
	}
}

func OAuthInitConfig(o OAuthRequest, cred OAuthCredentials,
	ds DriveDataStore) (*DriveInitConfig, *OAuthResponse, error) {

	c := oAuthConfig(o, cred)
	t := loadToken(ds)

	state := utils.RandString(6)
	if e := ds.Save(types.SM{DsKeyState: state}); e != nil {
		return nil, nil, e
	}
	initConfig := &DriveInitConfig{
		Configured: t != nil,
		OAuth: &OAuthConfig{
			URL:  c.AuthCodeURL(state, o.AutoCodeOption...),
			Text: o.Text,
		},
	}

	var resp *OAuthResponse
	if t != nil {
		resp = newOAuthResponse(c, ds, t)
	}

	return initConfig, resp, nil
}

func OAuthInit(ctx context.Context, o OAuthRequest, data types.SM,
	cred OAuthCredentials, ds DriveDataStore) (*OAuthResponse, error) {
	code := data["code"]
	state := data["state"]

	if code == "" {
		return nil, nil
	}

	oauthConf := oAuthConfig(o, cred)

	params, e := ds.Load(DsKeyState)
	if e != nil {
		return nil, e
	}
	if state != params[DsKeyState] {
		return nil, err.NewNotAllowedMessageError(i18n.T("oauth.state_mismatch"))
	}
	t, e := oauthConf.Exchange(ctx, code)
	if e != nil {
		return nil, e
	}
	return newOAuthResponse(oauthConf, ds, t), storeToken(ds, t)
}

func OAuthGet(o OAuthRequest, cred OAuthCredentials, ds DriveDataStore) (*OAuthResponse, error) {
	t := loadToken(ds)
	if t == nil {
		return nil, err.NewNotAllowedMessageError(i18n.T("drive.not_configured"))
	}
	return newOAuthResponse(oAuthConfig(o, cred), ds, t), nil
}

func loadToken(ds DriveDataStore) *oauth2.Token {
	params, e := ds.Load(DsKeyToken, DsKeyTokenType, DsKeyExpiresAt, DsKeyRefreshToken)
	if e != nil {
		return nil
	}
	expiresAt := params.GetUnixTime(DsKeyExpiresAt, nil)
	token := &oauth2.Token{
		AccessToken:  params[DsKeyToken],
		TokenType:    params[DsKeyTokenType],
		RefreshToken: params[DsKeyRefreshToken],
		Expiry:       expiresAt,
	}
	if token.AccessToken == "" {
		token = nil
	}
	if token != nil && token.RefreshToken == "" && expiresAt.Before(time.Now()) {
		token = nil
	}
	return token
}

func storeToken(ds DriveDataStore, token *oauth2.Token) error {
	return ds.Save(types.SM{
		DsKeyToken:        token.AccessToken,
		DsKeyTokenType:    token.TokenType,
		DsKeyRefreshToken: token.RefreshToken,
		DsKeyExpiresAt:    strconv.FormatInt(token.Expiry.Unix(), 10),
	})
}

type tokenSource struct {
	ts oauth2.TokenSource
	// t is used to store current token value
	t oauth2.Token

	ds DriveDataStore
	mu sync.Mutex
}

func (t *tokenSource) storeToken(token *oauth2.Token) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.t.AccessToken != token.AccessToken ||
		(token.RefreshToken != "" && token.RefreshToken != t.t.RefreshToken) {
		t.t = *token
		if e := storeToken(t.ds, token); e != nil {
			return e
		}
	}

	return nil
}

func (t *tokenSource) Token() (*oauth2.Token, error) {
	token, e := t.ts.Token()
	if e != nil {
		return nil, e
	}

	if e := t.storeToken(token); e != nil {
		return nil, e
	}

	return token, nil
}
