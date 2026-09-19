package utils

import "sync"

type KeyLock struct {
	m   map[string]*keyLockEntry
	mux sync.Mutex
}

type keyLockEntry struct {
	mu   sync.Mutex
	refs int
}

func NewKeyLock(size int) *KeyLock {
	return &KeyLock{m: make(map[string]*keyLockEntry, size)}
}

func (kl *KeyLock) getLock(key string) *keyLockEntry {
	kl.mux.Lock()
	defer kl.mux.Unlock()
	l, exists := kl.m[key]
	if !exists {
		l = &keyLockEntry{}
		kl.m[key] = l
	}
	l.refs++
	return l
}

func (kl *KeyLock) Lock(key string) {
	kl.getLock(key).mu.Lock()
}

func (kl *KeyLock) TryLock(key string) bool {
	l := kl.getLock(key)
	if l.mu.TryLock() {
		return true
	}
	kl.releaseRef(key, l)
	return false
}

func (kl *KeyLock) Unlock(key string) {
	kl.mux.Lock()
	l, exists := kl.m[key]
	kl.mux.Unlock()
	if !exists {
		panic("unlock of unknown key")
	}

	l.mu.Unlock()
	kl.releaseRef(key, l)
}
func (kl *KeyLock) releaseRef(key string, l *keyLockEntry) {
	kl.mux.Lock()
	l.refs--
	if l.refs == 0 && kl.m[key] == l {
		delete(kl.m, key)
	}
	kl.mux.Unlock()
}

// UnLock is kept for compatibility with the original misspelled method.
func (kl *KeyLock) UnLock(key string) {
	kl.Unlock(key)
}
