package strategies

import (
	"container/heap"
	"log"
	"sync"
	"time"
)

// 最少使用,给每个kv添加一个新的参数cnt计算次数，次数一样时候，把最久的键值对删除
type cacheLFU struct {
	mu        sync.Mutex
	maxBytes  int64
	nBytes    int64
	storage   map[string]*entryLFU
	minHeap   *MinHeap
	onEvicted func(key string, value Value)
}

type entryLFU struct {
	entry entry
	cnt   int
}

func NewLFU(maxBytes int64, onEvicted func(key string, value Value)) *cacheLFU {
	c := &cacheLFU{
		maxBytes:  maxBytes,
		nBytes:    0,
		onEvicted: onEvicted,
		storage:   make(map[string]*entryLFU),
		minHeap:   &MinHeap{},
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

func (c *cacheLFU) Get(key string) (value Value, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.storage[key]; ok {
		e.cnt++
		e.entry.TimeUpdate()
		heap.Fix(c.minHeap, c.getKeyIndex(e))
		return e.entry.value, ok
	}
	return
}
func (c *cacheLFU) Put(key string, value Value) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.storage[key]; ok {
		e.cnt++
		e.entry.TimeUpdate()
		c.nBytes = c.nBytes - int64(e.entry.value.Len()) + int64(value.Len())
		e.entry.value = value
		heap.Fix(c.minHeap, c.getKeyIndex(e))
	} else {
		kv := &entryLFU{
			cnt: 1,
			entry: entry{
				key:   key,
				value: value,
			},
		}
		kv.entry.TimeUpdate()
		c.storage[key] = kv
		c.nBytes = c.nBytes + int64(value.Len()) + int64(len(key))
		heap.Push(c.minHeap, c.storage[key])
	}
	for c.maxBytes != 0 && c.nBytes > c.maxBytes {
		c.Remove()
	}
}
func (c *cacheLFU) Remove() {
	item := heap.Pop(c.minHeap)
	kv := item.(*entryLFU)
	if kv != nil {
		delete(c.storage, kv.entry.key)
		c.nBytes = c.nBytes - int64(kv.entry.value.Len()) - int64(len(kv.entry.key))
		if c.onEvicted != nil {
			c.onEvicted(kv.entry.key, kv.entry.value)
		}
	}
}

func (c *cacheLFU) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.minHeap.Len() == 0 {
		return
	}
	var newHeap MinHeap
	for _, item := range *c.minHeap {
		if !item.entry.Expired() {
			newHeap = append(newHeap, item)
		} else {
			delete(c.storage, item.entry.key)
			c.nBytes = c.nBytes - int64(item.entry.value.Len()) - int64(len(item.entry.key))
			if c.onEvicted != nil {
				c.onEvicted(item.entry.key, item.entry.value)
			}
		}
	}
	*c.minHeap = newHeap
	heap.Init(c.minHeap)
}

func (c *cacheLFU) getKeyIndex(item *entryLFU) int {
	for i, cacheItem := range *c.minHeap {
		if cacheItem.entry.key == item.entry.key {
			return i
		}
	}
	return -1
}

func (c *cacheLFU) Len() int {
	return c.minHeap.Len()
}

type MinHeap []*entryLFU

func (h MinHeap) Len() int { return len(h) }
func (h MinHeap) Less(a, b int) bool {
	if h[a].cnt == h[b].cnt {
		//更久理当先被淘汰 a 在 b 之前
		return h[a].entry.timeout.Before(h[b].entry.timeout) 
	}
	return h[a].cnt < h[b].cnt
}
func (h MinHeap) Swap(a, b int) { h[a], h[b] = h[b], h[a] }
func (h *MinHeap) Push(x interface{}) {
	*h = append(*h, x.(*entryLFU))
}
func (h *MinHeap) Pop() interface{} {
	old := *h
	x := old[len(old)-1]
	old = old[:len(old)-1]
	*h = old
	return x
}
