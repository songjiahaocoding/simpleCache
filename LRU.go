package cache

import "container/list"

// LRUCache is an LRU cache. It is not safe for concurrent access.
type LRUCache struct {
	maxBytes int64
	nbytes   int64
	list     *list.List
	cache    map[string]*list.Element
	// optional and executed when an entry is purged.
	OnEvicted func(key string, value Val)
}

type entry struct {
	key   string
	value Val
}

// Val use Len() to count how many bytes it takes
type Val interface {
	Len() int
}

// New To initialize LRU
func New(maxBytes int64, onEvicted func(string, Val)) *LRUCache {
	return &LRUCache{
		maxBytes:  maxBytes,
		list:      list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

func (cache *LRUCache) Get(key string) (value Val, ok bool) {
	if item, ok := cache.cache[key]; ok {
		cache.list.MoveToFront(item)
		mapEntry := item.Value.(*entry)
		return mapEntry.value, true
	}
	return nil, false
}

func (cache *LRUCache) Remove() {
	item := cache.list.Back()
	if item != nil {
		cache.list.Remove(item)
		mapEntry := item.Value.(*entry)
		delete(cache.cache, mapEntry.key)
		cache.nbytes -= int64(len(mapEntry.key)) + int64(mapEntry.value.Len())
		if cache.OnEvicted != nil {
			cache.OnEvicted(mapEntry.key, mapEntry.value)
		}
	}
}

// Add : Add new entry into the cache
// If the key is already in the cache, update the value and move it to the front
// else insert the key and pop the least recently used key if necessary
func (cache *LRUCache) Add(key string, value Val) {
	if ele, ok := cache.cache[key]; ok {
		cache.list.MoveToFront(ele)
		mapEntry := ele.Value.(*entry)
		cache.nbytes += int64(value.Len()) - int64(mapEntry.value.Len())
		mapEntry.value = value
	} else {
		item := cache.list.PushFront(&entry{key, value})
		cache.cache[key] = item
		cache.nbytes += int64(len(key)) + int64(value.Len())
	}

	for cache.maxBytes != 0 && cache.maxBytes < cache.nbytes {
		cache.Remove()
	}
}

// Len : Get the number of entries in the cache
func (cache *LRUCache) Len() int {
	return cache.list.Len()
}
