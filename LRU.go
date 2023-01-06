package cache

import "container/list"

// Cache is a LRU cache. It is not safe for concurrent access.
type Cache struct {
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

// Value use Len to count how many bytes it takes
type Val interface {
	Len() int
}

// New To initialize LRU
func New(maxBytes int64, onEvicted func(string, Val)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		list:      list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

func (cache *Cache) Get(key string) (value Val, ok bool) {
	if item, ok := cache.cache[key]; ok {
		cache.list.MoveToFront(item)
		mapEntry := item.Value.(*entry)
		return mapEntry.value, true
	}
	return nil, false
}

func (cache *Cache) Remove() {
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

func (cache *Cache) Add(key string, value Val) {
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
