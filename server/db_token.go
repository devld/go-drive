package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/logging"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/storage"
	"time"

	"github.com/google/uuid"
)

const sessionCacheMaxEntries = 1000

// DBTokenStore is a server-side opaque token store backed by the database.
//
// The session row only stores the username (not a permission snapshot): the
// current user (groups, root path, ...) is resolved from UserDAO on every
// validation, so permission/disable/password changes take effect immediately.
//
// Only the SHA-256 hash of the token is stored. A short-lived in-memory cache
// fronts the reads, and the sliding-expiry refresh is throttled to avoid a
// database write on every request.
type DBTokenStore struct {
	sessionDAO *storage.SessionDAO
	userDAO    *storage.UserDAO

	validity    time.Duration
	autoRefresh bool

	cache       *utils.KVCache[dbTokenCacheItem]
	stopCleaner func()
}

type dbTokenCacheItem struct {
	username  string
	expiresAt int64
}

func NewDBTokenStore(sessionDAO *storage.SessionDAO, userDAO *storage.UserDAO,
	config common.Config, ch *registry.ComponentsHolder) (*DBTokenStore, error) {
	authConfig := config.Auth
	ts := &DBTokenStore{
		sessionDAO:  sessionDAO,
		userDAO:     userDAO,
		validity:    authConfig.Validity,
		autoRefresh: authConfig.AutoRefresh,
		cache:       utils.NewKVCache[dbTokenCacheItem](sessionCacheMaxEntries, 0),
	}
	ts.stopCleaner = utils.TimeTick(ts.clean, authConfig.Validity)
	ch.Add(registry.KeyTokenStore, ts)
	return ts, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (ts *DBTokenStore) invalidTokenError() error {
	return err.NewUnauthorizedError(i18n.T("api.db_token.invalid_token"))
}

func (ts *DBTokenStore) Create(value types.Principal) (types.Token, error) {
	token := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(ts.validity).Unix()
	row := types.Session{
		TokenHash: hashToken(token),
		Username:  value.User.Username,
		CreatedAt: now.Unix(),
		ExpiresAt: expiresAt,
	}
	if e := ts.sessionDAO.Create(row); e != nil {
		logging.For("auth").Errorf("session creation failed user=%s: %v", logging.Sanitize(row.Username), e)
		return types.Token{}, e
	}
	ts.setCached(row.TokenHash, dbTokenCacheItem{
		username:  row.Username,
		expiresAt: row.ExpiresAt,
	})
	logging.For("auth").Debugf("session created user=%s expires=%s", logging.Sanitize(row.Username),
		time.Unix(expiresAt, 0).Format(time.RFC3339))
	return types.Token{Token: token, Value: value, ExpiredAt: expiresAt}, nil
}

func (ts *DBTokenStore) Validate(token string) (types.Token, error) {
	if token == "" {
		return types.Token{}, ts.invalidTokenError()
	}
	hash := hashToken(token)

	item, ok := ts.getCached(hash)
	if !ok {
		logging.For("auth").Debugf("session cache miss")
		row, e := ts.sessionDAO.GetByHash(hash)
		if e != nil {
			logging.For("auth").Debugf("session validation failed reason=not_found_or_storage: %v", e)
			return types.Token{}, ts.invalidTokenError()
		}
		item = dbTokenCacheItem{username: row.Username, expiresAt: row.ExpiresAt}
		ts.setCached(hash, item)
	} else {
		logging.For("auth").Debugf("session cache hit user=%s", logging.Sanitize(item.username))
	}

	now := time.Now()
	if item.expiresAt <= now.Unix() {
		logging.For("auth").Debugf("session expired user=%s", logging.Sanitize(item.username))
		if e := ts.revokeHash(hash); e != nil {
			logging.For("auth").Warnf("expired session cleanup failed user=%s: %v", logging.Sanitize(item.username), e)
		}
		return types.Token{}, ts.invalidTokenError()
	}

	expiresAt := item.expiresAt
	if ts.autoRefresh {
		var exists bool
		expiresAt, exists = ts.maybeRefresh(hash, item, now)
		if !exists {
			logging.For("auth").Debugf("session validation failed reason=revoked user=%s", logging.Sanitize(item.username))
			return types.Token{}, ts.invalidTokenError()
		}
	}

	principal := types.Principal{}
	if item.username != "" {
		user, e := ts.userDAO.GetUser(item.username)
		if e != nil {
			// the user has been removed; invalidate the token
			logging.For("auth").Debugf("session invalidated reason=user_missing user=%s", logging.Sanitize(item.username))
			if revokeErr := ts.revokeHash(hash); revokeErr != nil {
				logging.For("auth").Warnf("invalid session cleanup failed user=%s: %v", logging.Sanitize(item.username), revokeErr)
			}
			return types.Token{}, ts.invalidTokenError()
		}
		principal = types.Principal{User: user, AuthType: types.AuthTypeToken}
	}

	return types.Token{Token: token, Value: principal, ExpiredAt: expiresAt}, nil
}

// maybeRefresh extends the token expiry only when it has passed half of its
// validity. It returns the effective expiry and whether the session still
// exists. The cache itself is concurrency-safe. Refresh and revoke are not
// serialized; RowsAffected detects a session removed before this update.
func (ts *DBTokenStore) maybeRefresh(hash string, item dbTokenCacheItem, now time.Time) (int64, bool) {
	half := int64(ts.validity.Seconds()) / 2
	if item.expiresAt-now.Unix() > half {
		return item.expiresAt, true
	}

	if current, ok := ts.getCached(hash); ok && current.expiresAt > item.expiresAt {
		return current.expiresAt, true
	}
	newExp := now.Add(ts.validity).Unix()
	if newExp <= item.expiresAt {
		return item.expiresAt, true
	}
	updated, e := ts.sessionDAO.UpdateExpiresAt(hash, newExp)
	if e != nil {
		logging.For("auth").Warnf("session refresh failed user=%s: %v", logging.Sanitize(item.username), e)
		return item.expiresAt, true
	}
	if !updated {
		logging.For("auth").Debugf("session refresh lost race user=%s", logging.Sanitize(item.username))
		ts.removeCached(hash)
		return item.expiresAt, false
	}
	item.expiresAt = newExp
	ts.setCached(hash, item)
	logging.For("auth").Debugf("session refreshed user=%s expires=%s", logging.Sanitize(item.username),
		time.Unix(newExp, 0).Format(time.RFC3339))
	return newExp, true
}

func (ts *DBTokenStore) Revoke(token string) error {
	return ts.revokeHash(hashToken(token))
}

func (ts *DBTokenStore) revokeHash(hash string) error {
	ts.removeCached(hash)
	return ts.sessionDAO.DeleteByHash(hash)
}

func (ts *DBTokenStore) getCached(hash string) (dbTokenCacheItem, bool) {
	return ts.cache.Get(hash)
}

func (ts *DBTokenStore) setCached(hash string, item dbTokenCacheItem) {
	ttl := time.Until(time.Unix(item.expiresAt, 0))
	if ttl <= 0 {
		return
	}
	ts.cache.Set(hash, item, ttl)
}

func (ts *DBTokenStore) removeCached(hash string) {
	ts.cache.Remove(hash)
}

func (ts *DBTokenStore) clean() {
	if e := ts.sessionDAO.DeleteExpired(time.Now().Unix()); e != nil {
		logging.For("auth").Errorf("error when cleaning expired sessions: %v", e)
		return
	}
	logging.For("auth").Debugf("expired sessions cleaned")
}

func (ts *DBTokenStore) Status() (string, types.SM, error) {
	total, active, e := ts.sessionDAO.Count(time.Now().Unix())
	if e != nil {
		return "", nil, e
	}
	return "Session", types.SM{
		"Total":  fmt.Sprintf("%d", total),
		"Active": fmt.Sprintf("%d", active),
	}, nil
}

func (ts *DBTokenStore) Dispose() error {
	ts.stopCleaner()
	ts.cache.Clear()
	return nil
}
