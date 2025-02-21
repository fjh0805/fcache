package strategies

import (
	"container/list"
	"log"
	"sync"
	"time"
)

// 最长未使用 很自然的，淘汰缓存从最后开始淘汰，遇到没过期的直接break就可以了
// 用双向队列实现
type cacheLRU struct {
	mu        sync.Mutex
	maxBytes  int64      // 最大内存
	nBytes    int64      // 已使用内存
	ll        *list.List // 存储数据的双向链表
	storage   map[string]*list.Element
	onEvicted func(key string, value Value)
}

func NewLRU(maxBytes int64, onEvicted func(key string, value Value)) *cacheLRU {
	c := &cacheLRU{
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

func (c *cacheLRU) Get(key string) (value Value, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, ok := c.storage[key]; ok {
		c.ll.MoveToBack(ele)
		kv := ele.Value.(*entry)
		kv.TimeUpdate() //更新过期时间
		return kv.value, ok
	}
	return
}

func (c *cacheLRU) Put(key string, value Value) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, ok := c.storage[key]; ok {
		c.ll.MoveToBack(ele)
		kv := ele.Value.(*entry)
		kv.TimeUpdate()
		c.nBytes = c.nBytes - int64(kv.value.Len()) + int64(value.Len())
		kv.value = value
	} else {
		kv := &entry{key: key, value: value}
		kv.TimeUpdate()
		ele := c.ll.PushBack(kv)
		c.storage[key] = ele
		c.nBytes = c.nBytes + int64(value.Len()) + int64(len(key))
	}
	for c.maxBytes != 0 && c.nBytes > c.maxBytes {
		c.RemoveOldest()
	}
}

func (c *cacheLRU) RemoveOldest() {
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

func (c *cacheLRU) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

func (c *cacheLRU) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	//从最久的开始删除
	e := c.ll.Front()
	for e != nil {
		next := e.Next()
		if entry, ok := e.Value.(*entry); ok && entry != nil {
			if entry.Expired() {
				c.ll.Remove(e)
				delete(c.storage, entry.key)
				c.nBytes -= int64(entry.value.Len()) + int64(len(entry.key))
				if c.onEvicted != nil {
					c.onEvicted(entry.key, entry.value)
				}
			} else {
				break
			}
		}
		e = next
	}
}
