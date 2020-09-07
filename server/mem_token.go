package server

import (
	"github.com/google/uuid"
	cmap "github.com/orcaman/concurrent-map"
	"go-drive/common"
	"log"
	"sync"
	"time"
)

type MemTokenStore struct {
	store cmap.ConcurrentMap

	validity    time.Duration
	autoRefresh bool

	mux *sync.Mutex

	ticker  *time.Ticker
	dispose chan bool
}

// NewMemTokenStore creates a MemTokenStore
//
// params:
//
// - autoRefresh: refresh token by adding `validity` after each token access
//
// - cleanupDuration: cleanup invalid token each `cleanupDuration`
func NewMemTokenStore(validity time.Duration, autoRefresh bool, cleanupDuration time.Duration) *MemTokenStore {
	if cleanupDuration <= 0 {
		panic("invalid cleanupDuration")
	}
	ticker := time.NewTicker(cleanupDuration)
	dispose := make(chan bool)
	tokenStore := &MemTokenStore{
		store:       cmap.New(),
		validity:    validity,
		autoRefresh: autoRefresh,
		mux:         &sync.Mutex{},
		ticker:      ticker,
		dispose:     dispose,
	}
	go func() {
		for {
			select {
			case <-dispose:
				return
			case <-ticker.C:
				tokenStore.clean()
			}
		}
	}()
	return tokenStore
}

func (m *MemTokenStore) Create(value interface{}) (Token, error) {
	key := uuid.New().String()
	var expiredAt int64 = -1
	if m.validity > 0 {
		expiredAt = time.Now().Add(m.validity).Unix()
	}
	token := Token{key, value, expiredAt}
	m.store.Set(key, token)
	return token, nil
}

func (m *MemTokenStore) Update(token string, value interface{}) (Token, error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	t, ok := m.store.Get(token)
	if !ok {
		return Token{}, common.NewUnauthorizedError("invalid token '" + token + "'")
	}
	tt := t.(Token)
	tt.Value = value
	if m.refreshEnabled() {
		tt.ExpiredAt = time.Now().Add(m.validity).Unix()
	}
	m.store.Set(token, tt)
	return tt, nil
}

func (m *MemTokenStore) Validate(token string) (interface{}, error) {
	if m.refreshEnabled() {
		m.mux.Lock()
		defer m.mux.Unlock()
	}
	t, ok := m.store.Get(token)
	if !ok {
		return nil, common.NewUnauthorizedError("invalid token '" + token + "'")
	}
	tt := t.(Token)
	if !m.isValid(tt) {
		return nil, common.NewUnauthorizedError("token expired '" + token + "'")
	}
	if m.refreshEnabled() {
		tt.ExpiredAt = time.Now().Add(m.validity).Unix()
		m.store.Set(token, tt)
	}
	return tt, nil
}

func (m *MemTokenStore) Revoke(token string) error {
	m.store.Remove(token)
	return nil
}

func (m *MemTokenStore) isValid(token Token) bool {
	return token.ExpiredAt <= 0 || token.ExpiredAt > time.Now().Unix()
}

func (m *MemTokenStore) refreshEnabled() bool {
	return m.autoRefresh && m.validity > 0
}

func (m *MemTokenStore) clean() {
	keys := make([]string, 0)
	m.store.IterCb(func(key string, v interface{}) {
		if !m.isValid(v.(Token)) {
			keys = append(keys, key)
		}
	})
	for _, key := range keys {
		_ = m.Revoke(key)
	}
	log.Printf("%d expired tokens cleaned", len(keys))
}

func (m *MemTokenStore) Dispose() error {
	m.dispose <- true
	m.ticker.Stop()
	return nil
}
