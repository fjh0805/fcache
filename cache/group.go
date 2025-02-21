package cache

import (
	"fmt"
	"log"
	"sync"

	cachepb "github.com/limerence-yu/fcache/cachepb"
	"github.com/limerence-yu/fcache/singleflight.go"
)

type Group struct {
	name      string
	mainCache Cache
	getter    Getter
	peers     PeerPicker
	loader    *singleflight.Group
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
		mainCache: Cache{
			Name:       "lru",
			CacheBytes: cacheBytes,
		},
		getter: getter,
		loader: &singleflight.Group{},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is empty")
	}
	if v, ok := g.mainCache.Get(key); ok {
		log.Printf("get key %s in cache", key)
		return v, nil
	}
	return g.load(key)
}

func (g *Group) load(key string) (ByteView, error) {
	v, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				log.Printf("pick peer %v", peer)

				req := &cachepb.Request{
					Group: g.name,
					Key: key,
				}
				res := &cachepb.Response{}
				
				err := peer.Get(req, res)
				if err == nil {
					return ByteView{res.Value}, nil
				}
				return ByteView{}, err
			}
		}
		return g.getLocally(key)
	})	
	if err == nil {
		return v.(ByteView), err
	}
	return ByteView{}, err
}

func (g *Group) getLocally(key string) (ByteView, error) {
	v, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{append([]byte{}, v...)}

	g.mainCache.Put(key, value)
	return ByteView{v}, nil
}
