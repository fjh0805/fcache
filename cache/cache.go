package cache

import (
	"sync"

	"github.com/limerence-yu/fcache/strategies"
)

type cache struct {
	mu         sync.Mutex
	strategy   strategies.CacheStrategy
	cacheBytes int64
	name       string
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.strategy == nil {
		return
	}
	if v, ok := c.strategy.Get(key); ok {
		return v.(ByteView), ok
	}
	return
}

func (c *cache) put(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.strategy == nil {
		c.strategy = strategies.New(c.name, c.cacheBytes, nil)
	}
	c.strategy.Put(key, value)
}
