package strategies

import "strings"

type CacheStrategy interface {
	Get(string) (Value, bool)
	Put(string, Value)
	Len() int
}

type Value interface {
	Len() int // 返回值所占用的内存大小
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

