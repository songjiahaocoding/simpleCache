package cache

import (
	pb "cache/cachePB/cachepb"
	"fmt"
	"log"
	"sync"
)

// A Getter loads data for a key.
// The callback designed for cache misses
// It should specify what need to be done when cache miss happens
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc implements Getter with a function.
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter interface
func (get GetterFunc) Get(key string) ([]byte, error) {
	return get(key)
}

// A Group is a cache namespace and associated data loaded spread over
type Group struct {
	name       string
	getter     Getter
	mainCache  cache
	peerPicker PeerPicker
	// To ensure that each key request is invoked only once
	loader *RequestGroup
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup create a new instance of Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &RequestGroup{},
	}
	groups[name] = g
	return g
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

func (g *Group) Get(key string) (*ByteView, error) {
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[Cache] hit")
		return v, nil
	}

	return g.load(key)
}

// Get the data from local memory and populate it to the cache
func (g *Group) getLocally(key string) (*ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return nil, err
	}
	value := ByteView{bytes: cloneBytes(bytes)}
	g.populateCache(key, &value)
	return &value, nil
}

// populateCache Populate new data into the cache
func (g *Group) populateCache(key string, value *ByteView) {
	g.mainCache.add(key, value)
}

// load Cache miss; Load data from remote peer or local memory
// try to load from remote peers first
func (g *Group) load(key string) (value *ByteView, err error) {
	view, err := g.loader.CallOnce(key, func() (interface{}, error) {
		if g.peerPicker != nil {
			if peer, ok := g.peerPicker.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})

	if err == nil {
		return view.(*ByteView), nil
	}
	return nil, err
}

// Retrieve data from the remote peer
func (g *Group) getFromPeer(peer PeerGetter, key string) (*ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return &ByteView{}, err
	}
	return &ByteView{bytes: res.Value}, nil
}

// RegisterPeers registers a PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peerPicker PeerPicker) {
	if g.peerPicker != nil {
		panic("There is already a Peer Picker")
	}
	g.peerPicker = peerPicker
}
