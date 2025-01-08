package cache

import (
	"fmt"
	"log"
	"sync"
)

type Group struct {
	name      string
	mainCache cache
	getter    Getter
}

var mu sync.RWMutex
var groups = make(map[string]*Group)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("getter nil")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name: name,
		mainCache: cache{
			name: "lru",
			cacheBytes: cacheBytes,	
		},
		getter: getter,
	}
	groups[name] = g
	return g
}

func getGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is empty")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Printf("get key %s in cache", key)
		return v, nil
	}
	return g.load(key)
}

func (g *Group) load(key string) (ByteView, error) {
	return g.getLocally(key)
}

func (g *Group) getLocally(key string) (ByteView, error) {
	v, err := g.getter.Get(key); 
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{append([]byte{}, v...)}
	
	g.mainCache.put(key, value)
	return ByteView{v}, nil
}