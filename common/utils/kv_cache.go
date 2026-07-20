package utils

import (
	"go-drive/common/types"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	cmap "github.com/orcaman/concurrent-map/v2"
)

// NewKVCache creates a KV cache. When maxSize > 0 the cache uses LRU eviction;
// otherwise it is unbounded. cleanInterval controls periodic removal of
// TTL-expired items; 0 disables it.
func NewKVCache[T any](maxSize int, cleanInterval time.Duration) *KVCache[T] {
	kv := &KVCache[T]{}
	if maxSize > 0 {
		c, e := lru.New[string, *kvCacheItem[T]](maxSize)
		if e != nil {
			panic(e)
		}
		kv.lruCache = c
	} else {
		kv.cmapCache = cmap.New[*kvCacheItem[T]]()
	}
	if cleanInterval > 0 {
		kv.timerStop = TimeTick(kv.evict, cleanInterval)
	}
	return kv
}

type KVCache[T any] struct {
	lruCache  *lru.Cache[string, *kvCacheItem[T]]
	cmapCache cmap.ConcurrentMap[string, *kvCacheItem[T]]
	timerStop func()
}

func (kv *KVCache[T]) Get(key string) (ret T, ok bool) {
	var item *kvCacheItem[T]
	if kv.lruCache != nil {
		item, ok = kv.lruCache.Get(key)
	} else {
		item, ok = kv.cmapCache.Get(key)
	}
	if !ok {
		return
	}
	if item.isExpired() {
		kv.Remove(key)
		ok = false
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
	if kv.lruCache != nil {
		kv.lruCache.Add(key, item)
	} else {
		kv.cmapCache.Set(key, item)
	}
}

func (kv *KVCache[T]) Remove(key string) {
	if kv.lruCache != nil {
		kv.lruCache.Remove(key)
	} else {
		kv.cmapCache.Remove(key)
	}
}

func (kv *KVCache[T]) Clear() {
	if kv.lruCache != nil {
		kv.lruCache.Purge()
	} else {
		kv.cmapCache.Clear()
	}
}

func (kv *KVCache[T]) Len() int {
	if kv.lruCache != nil {
		return kv.lruCache.Len()
	}
	return kv.cmapCache.Count()
}

func (kv *KVCache[T]) evict() {
	if kv.lruCache != nil {
		for _, key := range kv.lruCache.Keys() {
			item, ok := kv.lruCache.Peek(key)
			if ok && item.isExpired() {
				kv.lruCache.Remove(key)
			}
		}
	} else {
		keys := make([]string, 0)
		kv.cmapCache.IterCb(func(key string, v *kvCacheItem[T]) {
			if v.isExpired() {
				keys = append(keys, key)
			}
		})
		for _, key := range keys {
			kv.cmapCache.Remove(key)
		}
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
