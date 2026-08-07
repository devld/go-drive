package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"net/url"
	pathpkg "path"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	wopiSessionTTL = 10 * time.Hour
	wopiLockTTL    = 30 * time.Minute
)

type wopiSession struct {
	id          string
	tokenHash   [sha256.Size]byte
	username    string
	path        string
	origin      string
	resourceKey string
	writable    bool
	expiresAt   time.Time
}

type wopiLock struct {
	value     string
	version   string
	expiresAt time.Time
}

type wopiService struct {
	config    common.Config
	access    wopiDriveAccess
	userDAO   wopiUserStore
	discovery *wopiDiscoveryClient

	sessionsMu sync.RWMutex
	sessions   map[string]wopiSession

	locksMu      sync.Mutex
	locks        map[string]wopiLock
	resourceLock *utils.KeyLock

	stopCleaner func()
}

type wopiDriveAccess interface {
	GetDrive(types.Principal) (types.IDrive, error)
}

type wopiUserStore interface {
	GetUser(string) (types.User, error)
}

func newWOPIService(config common.Config, access wopiDriveAccess, userDAO wopiUserStore,
	ch *registry.ComponentsHolder) (*wopiService, error) {
	discovery, e := newWOPIDiscoveryClient(config.WOPI.DiscoveryURL)
	if e != nil {
		return nil, e
	}
	s := &wopiService{
		config:       config,
		access:       access,
		userDAO:      userDAO,
		discovery:    discovery,
		sessions:     make(map[string]wopiSession),
		locks:        make(map[string]wopiLock),
		resourceLock: utils.NewKeyLock(32),
	}
	s.stopCleaner = utils.TimeTick(s.clean, time.Minute)
	ch.Add(registry.KeyWOPI, s)
	return s, nil
}

func (s *wopiService) Dispose() error {
	s.stopCleaner()
	s.sessionsMu.Lock()
	clear(s.sessions)
	s.sessionsMu.Unlock()
	s.locksMu.Lock()
	clear(s.locks)
	s.locksMu.Unlock()
	return nil
}

func (s *wopiService) SysConfig() (string, types.M, error) {
	discovery, e := s.discovery.get(context.Background())
	if e != nil {
		return "", nil, e
	}
	return "wopi", discovery.sysConfig(), nil
}

func newWOPIToken() (string, [sha256.Size]byte, error) {
	raw := make([]byte, 32)
	if _, e := rand.Read(raw); e != nil {
		return "", [sha256.Size]byte{}, e
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, sha256.Sum256([]byte(token)), nil
}

func (s *wopiService) createSession(username, path, origin, resourceKey string,
	writable bool, notAfter time.Time) (wopiSession, string, error) {
	if notAfter.IsZero() || time.Until(notAfter) > wopiSessionTTL {
		notAfter = time.Now().Add(wopiSessionTTL)
	}
	token, tokenHash, e := newWOPIToken()
	if e != nil {
		return wopiSession{}, "", e
	}
	session := wopiSession{
		id:          uuid.NewString(),
		tokenHash:   tokenHash,
		username:    username,
		path:        utils.CleanPath(path),
		origin:      origin,
		resourceKey: resourceKey,
		writable:    writable,
		expiresAt:   notAfter,
	}
	s.sessionsMu.Lock()
	s.sessions[session.id] = session
	s.sessionsMu.Unlock()
	return session, token, nil
}

func (s *wopiService) validateSession(id, token string) (wopiSession, bool) {
	if id == "" || token == "" {
		return wopiSession{}, false
	}
	s.sessionsMu.RLock()
	session, ok := s.sessions[id]
	s.sessionsMu.RUnlock()
	if !ok || !session.expiresAt.After(time.Now()) {
		if ok {
			s.sessionsMu.Lock()
			delete(s.sessions, id)
			s.sessionsMu.Unlock()
		}
		return wopiSession{}, false
	}
	hash := sha256.Sum256([]byte(token))
	if subtle.ConstantTimeCompare(hash[:], session.tokenHash[:]) != 1 {
		return wopiSession{}, false
	}
	return session, true
}

func (s *wopiService) deleteSession(id string) {
	s.sessionsMu.Lock()
	delete(s.sessions, id)
	s.sessionsMu.Unlock()
}

func (s *wopiService) principal(session wopiSession) (types.Principal, error) {
	user, e := s.userDAO.GetUser(session.username)
	if e != nil {
		return types.Principal{}, e
	}
	return types.Principal{User: user, AuthType: types.AuthTypeToken}, nil
}

func canonicalWOPIResourceKey(entry types.IEntry) string {
	dispatched := driveutil.GetIEntry(entry, func(candidate types.IEntry) bool {
		_, ok := candidate.(types.IDispatcherEntry)
		return ok
	})
	if dispatched != nil {
		return dispatched.(types.IDispatcherEntry).GetRealPath()
	}
	unwrapped := driveutil.UnwrapIEntry(entry)
	return fmt.Sprintf("%T:%p:%s", unwrapped.Drive(), unwrapped.Drive(), unwrapped.Path())
}

func wopiVersion(entry types.IEntry) string {
	data := fmt.Sprintf("%s\x00%d\x00%d", canonicalWOPIResourceKey(entry), entry.ModTime(), entry.Size())
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:16])
}

func validateWOPIOrigin(rawOrigin, requestHost string) (string, error) {
	u, e := url.Parse(rawOrigin)
	if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
		u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("invalid request origin")
	}
	if !strings.EqualFold(u.Host, requestHost) {
		return "", fmt.Errorf("request origin does not match host")
	}
	u.Path = ""
	return strings.TrimSuffix(u.String(), "/"), nil
}

func (s *wopiService) fileURL(origin, id string) string {
	apiPath := strings.Trim(s.config.APIPath, "/")
	parts := []string{origin}
	if apiPath != "" {
		parts = append(parts, apiPath)
	}
	parts = append(parts, "wopi", "files", url.PathEscape(id))
	return strings.Join(parts, "/")
}

func (s *wopiService) relativePath(session wopiSession, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name != pathpkg.Base(name) || name == "." || name == ".." || strings.ContainsAny(name, `/\\`) {
		return "", fmt.Errorf("invalid relative file name")
	}
	return pathpkg.Join(pathpkg.Dir(session.path), name), nil
}

func (s *wopiService) currentLock(resourceKey string, now time.Time) (wopiLock, bool) {
	s.locksMu.Lock()
	defer s.locksMu.Unlock()
	lock, ok := s.locks[resourceKey]
	if ok && !lock.expiresAt.After(now) {
		delete(s.locks, resourceKey)
		return wopiLock{}, false
	}
	return lock, ok
}

func (s *wopiService) setLock(resourceKey string, lock wopiLock) {
	s.locksMu.Lock()
	s.locks[resourceKey] = lock
	s.locksMu.Unlock()
}

func (s *wopiService) deleteLock(resourceKey string) {
	s.locksMu.Lock()
	delete(s.locks, resourceKey)
	s.locksMu.Unlock()
}

func (s *wopiService) clean() {
	now := time.Now()
	s.sessionsMu.Lock()
	for id, session := range s.sessions {
		if !session.expiresAt.After(now) {
			delete(s.sessions, id)
		}
	}
	s.sessionsMu.Unlock()
	s.locksMu.Lock()
	for key, lock := range s.locks {
		if !lock.expiresAt.After(now) {
			delete(s.locks, key)
		}
	}
	s.locksMu.Unlock()
}
