package utils

import (
	"go-drive/common/types"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
)

func NewKVCache[T any](clearInterval time.Duration) *KVCache[T] {
	kv := &KVCache[T]{cache: cmap.New[*kvCacheItem[T]]()}
	if clearInterval > 0 {
		kv.timerStop = TimeTick(kv.evict, clearInterval)
	}
	return kv
}

type KVCache[T any] struct {
	cache     cmap.ConcurrentMap[string, *kvCacheItem[T]]
	timerStop func()
}

func (kv *KVCache[T]) Get(key string) (ret T, ok bool) {
	var item *kvCacheItem[T]
	item, ok = kv.cache.Get(key)
	if !ok {
		return
	}
	if item.isExpired() {
		kv.Remove(key)
		return
	}
	return item.data, true
}

func (kv *KVCache[T]) Set(key string, value T, ttl time.Duration) {
	var exp *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		exp = &t
	}
	item := &kvCacheItem[T]{data: value, expiresAt: exp}
	kv.cache.Set(key, item)
}

func (kv *KVCache[T]) Remove(key string) {
	kv.cache.Remove(key)
}

func (kv *KVCache[T]) Clear() {
	kv.cache.Clear()
}

func (kv *KVCache[T]) evict() {
	keys := make([]string, 0)
	kv.cache.IterCb(func(key string, v *kvCacheItem[T]) {
		if v.isExpired() {
			keys = append(keys, key)
		}
	})
	for _, key := range keys {
		kv.cache.Remove(key)
	}
}

func (kv *KVCache[T]) Dispose() error {
	if kv.timerStop != nil {
		kv.timerStop()
	}
	return nil
}

type kvCacheItem[T any] struct {
	data      T
	expiresAt *time.Time
}

func (item *kvCacheItem[T]) isExpired() bool {
	return item.expiresAt != nil && item.expiresAt.Before(time.Now())
}

var _ types.IDisposable = (*KVCache[any])(nil)
