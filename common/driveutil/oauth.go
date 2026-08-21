package driveutil

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
	// OAuth helper data is private drive data and uses a reserved namespace so
	// it cannot collide with drive-specific configuration, runtime data, or
	// initialization callback data.
	DsKeyCode         = "__oauth_code"
	DsKeyToken        = "__oauth_token"
	DsKeyTokenType    = "__oauth_token_type"
	DsKeyExpiresAt    = "__oauth_expires_at"
	DsKeyRefreshToken = "__oauth_refresh_token"
	DsKeyState        = "__oauth_state"
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

// OAuthHolder holds a drive's persisted OAuth token and refreshes it as needed.
type OAuthHolder struct {
	conf *oauth2.Config
	ds   DriveDataStore

	mu sync.Mutex
	t  *oauth2.Token
}

func newOAuthHolder(config *oauth2.Config, ds DriveDataStore, token *oauth2.Token) *OAuthHolder {
	return &OAuthHolder{conf: config, ds: ds, t: token}
}

func (o *OAuthHolder) Client() *http.Client {
	return &http.Client{Transport: &oauthTransport{o: o}}
}

func (o *OAuthHolder) Token(ctx context.Context) (*oauth2.Token, error) {
	if ctx == nil {
		panic("driveutil.OAuthHolder.Token: nil context")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if o.t.Valid() {
		tok := *o.t
		return &tok, nil
	}

	newTok, e := o.conf.TokenSource(ctx, o.t).Token()
	if e != nil {
		return nil, e
	}
	if e := o.persistLocked(newTok); e != nil {
		return nil, e
	}
	tok := *newTok
	return &tok, nil
}

// Refresh forces a token endpoint exchange even if the cached access token is
// still valid. Use this from onInterval; request paths should keep calling
// Token. There must be a refresh token. The in-memory cache and data store are
// updated on the same holder so concurrent Token calls see the new value.
func (o *OAuthHolder) Refresh(ctx context.Context) (*oauth2.Token, error) {
	if ctx == nil {
		panic("driveutil.OAuthHolder.Refresh: nil context")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if o.t == nil || o.t.RefreshToken == "" {
		return nil, err.NewNotAllowedMessageError("OAuth refresh token is not available")
	}

	stale := &oauth2.Token{RefreshToken: o.t.RefreshToken}
	newTok, e := o.conf.TokenSource(ctx, stale).Token()
	if e != nil {
		return nil, e
	}
	if newTok.RefreshToken == "" {
		copied := *newTok
		copied.RefreshToken = o.t.RefreshToken
		newTok = &copied
	}
	if e := o.persistLocked(newTok); e != nil {
		return nil, e
	}
	tok := *newTok
	return &tok, nil
}

func (o *OAuthHolder) persistLocked(token *oauth2.Token) error {
	changed := o.t == nil ||
		o.t.AccessToken != token.AccessToken ||
		(token.RefreshToken != "" && token.RefreshToken != o.t.RefreshToken)
	copied := *token
	o.t = &copied
	if !changed {
		return nil
	}
	return storeToken(o.ds, token)
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
	ds DriveDataStore) (*DriveInitConfig, *OAuthHolder, error) {

	c := oAuthConfig(o, cred)
	t := loadToken(ds)

	// use a cryptographically-strong, non-guessable state to mitigate CSRF
	state := utils.Base64URLEncode(utils.RandSecret(16))
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

	var oauthHolder *OAuthHolder
	if t != nil {
		oauthHolder = newOAuthHolder(c, ds, t)
	}

	return initConfig, oauthHolder, nil
}

func OAuthInit(ctx context.Context, o OAuthRequest, data types.SM,
	cred OAuthCredentials, ds DriveDataStore) (*OAuthHolder, error) {
	code := data[DsKeyCode]
	state := data[DsKeyState]

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
	return newOAuthHolder(oauthConf, ds, t), storeToken(ds, t)
}

func OAuthLoad(o OAuthRequest, cred OAuthCredentials, ds DriveDataStore) (*OAuthHolder, error) {
	t := loadToken(ds)
	if t == nil {
		return nil, err.NewNotAllowedMessageError(i18n.T("drive.not_configured"))
	}
	return newOAuthHolder(oAuthConfig(o, cred), ds, t), nil
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

type oauthTransport struct {
	base http.RoundTripper
	o    *OAuthHolder
}

func (t *oauthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqBodyClosed := false
	if req.Body != nil {
		defer func() {
			if !reqBodyClosed {
				_ = req.Body.Close()
			}
		}()
	}

	tok, e := t.o.Token(req.Context())
	if e != nil {
		return nil, e
	}
	req2 := req.Clone(req.Context())
	tok.SetAuthHeader(req2)
	reqBodyClosed = true
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req2)
}
