package strategies

import (
	"container/list"
	"log"
	"sync"
	"time"
)

// 先进先出
type cacheFIFO struct {
	mu        sync.Mutex
	maxBytes  int64
	nBytes    int64
	ll        *list.List // 存储数据的双向链表
	storage   map[string]*list.Element
	onEvicted func(key string, value Value)
}

func NewFIFO(maxBytes int64, onEvicted func(key string, value Value)) *cacheFIFO {
	c := &cacheFIFO{
		maxBytes:  maxBytes,
		ll:        list.New(),
		storage:   make(map[string]*list.Element),
		onEvicted: onEvicted,
	}
	go func() {
		tiker := time.NewTicker(2 * time.Minute)
		defer tiker.Stop()
		for {
			<-tiker.C
			c.Cleanup()
			log.Println("定期处理过期缓存")
		}
	}()
	return c
}

func (c *cacheFIFO) Get(key string) (value Value, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, ok := c.storage[key]; ok {
		kv := ele.Value.(*entry)
		kv.TimeUpdate()
		return kv.value, ok
	}
	return nil, false
}

func (c *cacheFIFO) Put(key string, value Value) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, ok := c.storage[key]; ok {
		kv := ele.Value.(*entry)
		c.nBytes = c.nBytes - int64(kv.value.Len()) + int64(value.Len())
		kv.value = value
		kv.TimeUpdate()
	} else {
		kv := &entry{key: key, value: value}
		kv.TimeUpdate()
		ele := c.ll.PushBack(kv)
		c.storage[key] = ele
		c.nBytes = c.nBytes + int64(kv.value.Len()) + int64(len(kv.key))
	}
	for c.maxBytes != 0 && c.nBytes > c.maxBytes {
		c.RemoveOldest()
	}
}

func (c *cacheFIFO) RemoveOldest() {
	ele := c.ll.Front()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.storage, kv.key)
		c.nBytes -= int64(kv.value.Len()) + int64(len(kv.key))
		if c.onEvicted != nil {
			c.onEvicted(kv.key, kv.value)
		}
	}
}

func (c *cacheFIFO) Len() int {
	return c.ll.Len()
}

func (c *cacheFIFO) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for e := c.ll.Front(); e != nil; {
		next := e.Next()
		kv := e.Value.(*entry)
		if kv.Expired() {
			c.ll.Remove(e)
			delete(c.storage, kv.key)
			c.nBytes -= int64(kv.value.Len()) + int64(len(kv.key))
			if c.onEvicted != nil {
				c.onEvicted(kv.key, kv.value)
			}
		}
		e = next
	}
}
