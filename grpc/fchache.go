package grpc

import (
	"fmt"
	"log"
	"sync"

	"github.com/limerence-yu/fcache/cache"
	"github.com/limerence-yu/fcache/singleflight.go"
)

type Group struct {
	name      string
	mainCache cache.Cache
	getter    cache.Getter
	servers   Picker
	loader    *singleflight.Group
}

var mu sync.RWMutex
var groups = make(map[string]*Group)

func NewGroup(name string, cacheBytes int64, getter cache.Getter) *Group {
	if getter == nil {
		panic("getter nil")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name: name,
		mainCache: cache.Cache{
			Name:       "lru",
			CacheBytes: cacheBytes,
		},
		getter: getter,
		loader: &singleflight.Group{},
	}
	groups[name] = g
	log.Printf("group: %s", g.name)
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

func (g *Group) RegisterSvr(peers Picker) {
	if g.servers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.servers = peers
}

func (g *Group) Get(key string) (cache.ByteView, error) {
	if key == "" {
		return cache.ByteView{}, fmt.Errorf("key is empty")
	}
	if v, ok := g.mainCache.Get(key); ok {
		log.Printf("get key %s in cache", key)
		return v, nil
	}
	return g.load(key)
}

func (g *Group) load(key string) (cache.ByteView, error) {
	v, err := g.loader.Do(key, func() (interface{}, error) {
		if g.servers != nil {
			if server, ok := g.servers.Pick(key); ok {
				log.Printf("[%s] attempting to fetch from peer for key %s", g.name, key)
				bytes, err := server.Fetch(g.name, key)
				if err == nil {
					return cache.ByteView{Bytes: append([]byte{}, bytes...)}, nil
				}
				log.Printf("fail to get key *%s* from peer, %s.\n", key, err.Error())
			}
		}
		return g.getLocally(key)
	})
	if err == nil {
		return v.(cache.ByteView), err
	}
	return cache.ByteView{}, err
}

func (g *Group) getLocally(key string) (cache.ByteView, error) {
	v, err := g.getter.Get(key)
	if err != nil {
		return cache.ByteView{}, err
	}
	value := cache.ByteView{Bytes: append([]byte{}, v...)}

	g.mainCache.Put(key, value)
	return value, nil
}

func DestroyGroup(name string) {
	g := GetGroup(name)
	if g != nil {
		svr := g.servers.(*Server)
		svr.Stop()
		delete(groups, name)
		log.Printf("Destroy cache [%s %s]", name, svr.Addr)
	}
}
