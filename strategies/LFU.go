package strategies

import (
	"container/heap"
	"time"
)

// 最少使用
type cacheLFU struct {
	maxBytes  int64
	nBytes    int64
	storage   map[string]*entryLFU
	minHeap   *MinHeap
	onEvicted func(key string, value Value)
}

type entryLFU struct {
	key            string
	value          Value
	cnt            int
	lastAccessTime int64
}

func NewLFU(maxBytes int64, onEvicted func(key string, value Value)) *cacheLFU {
	return &cacheLFU{
		maxBytes:  maxBytes,
		nBytes:    0,
		onEvicted: onEvicted,
		storage:   make(map[string]*entryLFU),
		minHeap:   &MinHeap{},
	}
}

func (c *cacheLFU) Get(key string) (value Value, ok bool) {
	if e, ok := c.storage[key]; ok {
		e.cnt++
		e.lastAccessTime = time.Now().UnixNano()
		heap.Fix(c.minHeap, c.getKeyIndex(e))
		return e.value, ok
	}
	return
}
func (c *cacheLFU) Put(key string, value Value) {
	if e, ok := c.storage[key]; ok {
		e.cnt++
		e.lastAccessTime = time.Now().UnixNano()
		c.nBytes = c.nBytes - int64(e.value.Len()) + int64(value.Len())
		e.value = value
		heap.Fix(c.minHeap, c.getKeyIndex(e))
	} else {
		c.storage[key] = &entryLFU{
			key:            key,
			cnt:            1,
			value:          value,
			lastAccessTime: time.Now().UnixNano(),
		}
		c.nBytes = c.nBytes + int64(value.Len()) + int64(len(key))
		heap.Push(c.minHeap, c.storage[key])
	}
	for c.maxBytes != 0 && c.nBytes > c.maxBytes {
		c.Remove()
	}
}
func (c *cacheLFU) Remove() {
	item := heap.Pop(c.minHeap)
	entry := item.(*entryLFU)
	if entry != nil {
		delete(c.storage, entry.key)
		c.nBytes = c.nBytes - int64(entry.value.Len()) - int64(len(entry.key))
		if c.onEvicted != nil {
			c.onEvicted(entry.key, entry.value)
		}
	}
}

func (c *cacheLFU) getKeyIndex(item *entryLFU) int {
	for i, cacheItem := range *c.minHeap {
		if cacheItem.key == item.key {
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
		return h[a].lastAccessTime < h[b].lastAccessTime
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
