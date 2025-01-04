package strategies

import "container/list"

//最长未使用
//用双向队列实现
type cacheLRU struct {
	maxBytes  int64      // 最大内存
	nBytes    int64      // 已使用内存
	ll        *list.List // 存储数据的双向链表
	storage   map[string]*list.Element
	onEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int // 返回值所占用的内存大小
}

func New(maxBytes int64, onEvicted func(key string, value Value)) *cacheLRU {
	return &cacheLRU{
		maxBytes: maxBytes,
		ll:       list.New(),
		storage:  make(map[string]*list.Element),
		onEvicted: onEvicted,
	}
}

func (c *cacheLRU) Get(key string) (value Value, ok bool) {
	if ele, ok := c.storage[key]; ok {
		c.ll.MoveToBack(ele)
		kv := ele.Value.(*entry)
		return kv.value, ok
	}
	return
}

func (c *cacheLRU) Put(key string, value Value) {
	if ele, ok := c.storage[key]; ok {
		c.ll.MoveToBack(ele)
		kv := ele.Value.(*entry)
		c.nBytes = c.nBytes - int64(kv.value.Len()) + int64(value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushBack(&entry{key, value})
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
	return c.ll.Len()
}
