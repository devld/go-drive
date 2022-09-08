package utils

import "sync"

type KeyLock struct {
	m   map[string]*sync.Mutex
	mux sync.Mutex
}

func NewKeyLock(size int) *KeyLock {
	return &KeyLock{m: make(map[string]*sync.Mutex, size)}
}

func (kl *KeyLock) getLock(key string) *sync.Mutex {
	kl.mux.Lock()
	defer kl.mux.Unlock()
	l, exists := kl.m[key]
	if !exists {
		l = &sync.Mutex{}
		kl.m[key] = l
	}
	return l
}

func (kl *KeyLock) Lock(key string) {
	kl.getLock(key).Lock()
}

func (kl *KeyLock) TryLock(key string) {
	kl.getLock(key).TryLock()
}

func (kl *KeyLock) UnLock(key string) {
	kl.getLock(key).Unlock()
}
