package strategies

import (
	"math/rand"
	"strings"
	"time"
)

const (
	maxTimeout = 31
	minTimeout = 29
)

var ttl = time.Duration(minTimeout+rand.Intn(maxTimeout-minTimeout)) * time.Second

type CacheStrategy interface {
	Get(string) (Value, bool)
	Put(string, Value)
	Cleanup()
	Len() int
}

type Value interface {
	Len() int // 返回值所占用的内存大小
}

// 增加缓存过期功能
type entry struct {
	key     string
	value   Value
	timeout time.Time // 过期时间点
}

func (e *entry) Expired() bool {
	if e.timeout.IsZero() {
		return false
	}
	return time.Now().After(e.timeout)
}

func (e *entry) TimeUpdate() {
	e.timeout = time.Now().Add(ttl) // 直接设为当前时间 + ttl

}

func New(name string, maxBytes int64, onEvicted func(string, Value)) CacheStrategy {
	name = strings.ToLower(name)
	switch name {
	case "fifo":
		return NewFIFO(maxBytes, onEvicted)
	case "lru":
		return NewLRU(maxBytes, onEvicted)
	case "lfu":
		return NewLFU(maxBytes, onEvicted)
	}
	return nil
}
