package strategies

import "container/list"

// 先进先出
type cacheFIFO struct {
	maxBytes  int64
	nBytes    int64
	ll        *list.List // 存储数据的双向链表
	storage   map[string]*list.Element
	onEvicted func(key string, value Value)
}

func NewFIFO(maxBytes int64, onEvicted func(key string, value Value)) *cacheFIFO {
	return &cacheFIFO{
		maxBytes: maxBytes,
		ll:       list.New(),
		storage:  make(map[string]*list.Element),
		onEvicted: onEvicted,
	}
}

func (c *cacheFIFO) Get(key string) (value Value, ok bool){
	if ele, ok := c.storage[key]; ok {
		kv := ele.Value.(*entry)
		return kv.value, ok
	}
	return nil, false
}

func (c *cacheFIFO) Put(key string, value Value) {
	if ele, ok := c.storage[key]; ok {
		kv := ele.Value.(*entry)
		c.nBytes = c.nBytes - int64(kv.value.Len()) + int64(value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushBack(&entry{key, value})
		c.storage[key] = ele
		kv := ele.Value.(*entry)
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

