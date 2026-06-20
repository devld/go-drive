package storage

import (
	"go-drive/common/registry"
	"go-drive/common/types"
)

// SessionDAO persists login sessions (types.Session). Only the SHA-256 hash of
// the token is stored, never the raw token.
type SessionDAO struct {
	db *DB
}

func NewSessionDAO(db *DB, ch *registry.ComponentsHolder) *SessionDAO {
	dao := &SessionDAO{db: db}
	ch.Add(registry.KeySessionDAO, dao)
	return dao
}

// Create inserts a new session row.
func (d *SessionDAO) Create(session types.Session) error {
	return d.db.C().Create(&session).Error
}

// GetByHash returns the session identified by its token hash.
func (d *SessionDAO) GetByHash(tokenHash string) (types.Session, error) {
	session := types.Session{}
	e := d.db.C().Where("token_hash = ?", tokenHash).Take(&session).Error
	return session, e
}

// UpdateExpiresAt extends (or shortens) the expiry of a session.
func (d *SessionDAO) UpdateExpiresAt(tokenHash string, expiresAt int64) error {
	return d.db.C().Model(&types.Session{}).
		Where("token_hash = ?", tokenHash).
		Update("expires_at", expiresAt).Error
}

// DeleteByHash removes a single session.
func (d *SessionDAO) DeleteByHash(tokenHash string) error {
	return d.db.C().Where("token_hash = ?", tokenHash).Delete(&types.Session{}).Error
}

// DeleteExpired removes all sessions that expired before the given unix time.
func (d *SessionDAO) DeleteExpired(now int64) error {
	return d.db.C().Where("expires_at < ?", now).Delete(&types.Session{}).Error
}

// Count returns the total number of sessions and how many are still active
// (not expired) at the given unix time.
func (d *SessionDAO) Count(now int64) (total int64, active int64, e error) {
	if e = d.db.C().Model(&types.Session{}).Count(&total).Error; e != nil {
		return 0, 0, e
	}
	if e = d.db.C().Model(&types.Session{}).
		Where("expires_at >= ?", now).Count(&active).Error; e != nil {
		return 0, 0, e
	}
	return total, active, nil
}
