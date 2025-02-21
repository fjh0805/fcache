package cache

import (
	"sync"

	"github.com/limerence-yu/fcache/strategies"
)

type Cache struct {
	mu         sync.Mutex
	Strategy   strategies.CacheStrategy
	CacheBytes int64
	Name       string
}

func (c *Cache) Get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Strategy == nil {
		return
	}
	if v, ok := c.Strategy.Get(key); ok {
		return v.(ByteView), ok
	}
	return
}

func (c *Cache) Put(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Strategy == nil {
		c.Strategy = strategies.New(c.Name, c.CacheBytes, nil)
	}
	c.Strategy.Put(key, value)
}
