package mega

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/types"

	megaapi "go-drive/drive/mega/internal/gomega"
)

const (
	sessionSIDKey         = "mega_sid"
	sessionMasterKeyKey   = "mega_master_key"
	sessionEmailKey       = "mega_email"
	sessionPasswordTagKey = "mega_password_tag"
)

// authenticator is the MEGA login surface used to restore or create a session.
type authenticator interface {
	LoginWithKeys(sessionID string, masterKey []byte) error
	MultiFactorLogin(email, password, multiFactor string) error
	GetSessionID() string
	GetMasterKey() []byte
}

func passwordTag(email, password string) string {
	sum := sha256.Sum256([]byte(email + "\n" + password))
	return hex.EncodeToString(sum[:])
}

// restoreOrLogin reuses a saved session when the account credentials still match.
// MEGA session tokens avoid asking for a single-use multi-factor code on every restart.
func restoreOrLogin(auth authenticator, email, password, multiFactor string, data driveutil.DriveDataStore) error {
	tag := passwordTag(email, password)
	if data != nil {
		stored, loadErr := data.Load(sessionSIDKey, sessionMasterKeyKey, sessionEmailKey, sessionPasswordTagKey)
		if loadErr == nil && stored[sessionEmailKey] == email && stored[sessionPasswordTagKey] == tag && stored[sessionSIDKey] != "" {
			masterKey, decodeErr := base64.StdEncoding.DecodeString(stored[sessionMasterKeyKey])
			if decodeErr == nil && len(masterKey) > 0 {
				loginErr := auth.LoginWithKeys(stored[sessionSIDKey], masterKey)
				if loginErr == nil {
					return nil
				}
				if !sessionRejected(loginErr) {
					return mapLoginError(loginErr)
				}
			}
		}
	}

	if loginErr := auth.MultiFactorLogin(email, password, multiFactor); loginErr != nil {
		return mapLoginError(loginErr)
	}
	if data == nil {
		return nil
	}
	return data.SaveEncrypted(types.SM{
		sessionSIDKey:         auth.GetSessionID(),
		sessionMasterKeyKey:   base64.StdEncoding.EncodeToString(auth.GetMasterKey()),
		sessionEmailKey:       email,
		sessionPasswordTagKey: tag,
	})
}

// mfaRequiredError keeps the HTTP status of an unauthorized login and still
// matches MEGA's multi-factor sentinel.
type mfaRequiredError struct {
	err.UnauthorizedError
}

func (mfaRequiredError) Unwrap() error { return megaapi.EMFAREQUIRED }

// probeSession reports whether a saved session or a password login is enough.
// A completed login is configured and has nothing left to enter.
// A multi-factor challenge leaves the session store unchanged and asks for a code.
func probeSession(auth authenticator, email, password string, data driveutil.DriveDataStore) (*driveutil.DriveInitConfig, error) {
	loginErr := restoreOrLogin(auth, email, password, "", data)
	if errors.Is(loginErr, megaapi.EMFAREQUIRED) {
		return &driveutil.DriveInitConfig{
			Form: []types.FormItem{{
				Field:       "mfa",
				Label:       t("form.mfa.label"),
				Type:        "password",
				Required:    true,
				Description: t("form.mfa.description"),
			}},
		}, nil
	}
	if loginErr != nil {
		return nil, loginErr
	}
	return &driveutil.DriveInitConfig{Configured: true}, nil
}

// loginWithCode completes a challenged login. The code is not written to the store.
func loginWithCode(auth authenticator, email, password, code string, data driveutil.DriveDataStore) error {
	return restoreOrLogin(auth, email, password, strings.TrimSpace(code), data)
}

func sessionRejected(loginErr error) bool {
	return errors.Is(loginErr, megaapi.ESID) || errors.Is(loginErr, megaapi.EKEY) || errors.Is(loginErr, megaapi.ENOENT)
}

func mapLoginError(loginErr error) error {
	switch {
	case loginErr == nil:
		return nil
	case errors.Is(loginErr, megaapi.ENOENT), errors.Is(loginErr, megaapi.EKEY):
		return err.NewUnauthorizedError(t("bad_credentials"))
	case errors.Is(loginErr, megaapi.EMFAREQUIRED):
		return mfaRequiredError{UnauthorizedError: err.NewUnauthorizedError(t("mfa_required"))}
	case errors.Is(loginErr, megaapi.EBLOCKED):
		return err.NewNotAllowedMessageError(t("blocked"))
	case errors.Is(loginErr, megaapi.EOVERQUOTA), errors.Is(loginErr, megaapi.EGOINGOVERQUOTA):
		return err.NewNotAllowedMessageError(t("quota"))
	default:
		return loginErr
	}
}
