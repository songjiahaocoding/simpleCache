package cache

import (
	"sync"
)

// cache Encapsulated operations exposed to users
type cache struct {
	mu         sync.Mutex
	lru        *LRUCache
	cacheBytes int64
}

func (cache *cache) add(key string, bv *ByteView) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.lru == nil {
		cache.lru = New(cache.cacheBytes, nil)
	}
	cache.lru.Add(key, bv)
}

func (cache *cache) get(key string) (bv *ByteView, ok bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.lru == nil {
		return nil, false
	}

	if v, ok := cache.lru.Get(key); ok {
		return v.(*ByteView), ok
	}

	return nil, false
}
