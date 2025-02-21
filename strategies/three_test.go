package strategies

import (
	"fmt"
	"testing"
	"time"
)

type mockValue struct{ size int }
func (m *mockValue) Len() int { return m.size }

func TestCacheStrategies(t *testing.T) {
	onEvicted := func(key string, value Value) {
		fmt.Printf("Evicted: %s\n", key)
	}

	// 测试 FIFO
	t.Run("FIFO", func(t *testing.T) {
		c := NewFIFO(25, onEvicted)
		c.Put("k1", &mockValue{10})
		c.Put("k2", &mockValue{20}) // k1 移除
		if _, ok := c.Get("k1"); ok {
			t.Errorf("k1 should be evicted")
		}
		c.storage["k2"].Value.(*entry).timeout = time.Now().Add(-1 * time.Second)
		c.Cleanup()
		if c.Len() != 0 {
			t.Errorf("Len expected 0, got %d", c.Len())
		}
	})

	// 测试 LRU
	t.Run("LRU", func(t *testing.T) {
		c := NewLRU(25, onEvicted)
		c.Put("k1", &mockValue{10})
		c.Put("k2", &mockValue{20}) // k1 移除
		if _, ok := c.Get("k1"); ok {
			t.Errorf("k1 should be evicted")
		}
		c.Get("k2") // 更新 k2
		c.storage["k2"].Value.(*entry).timeout = time.Now().Add(-1 * time.Second)
		c.Cleanup()
		if c.Len() != 0 {
			t.Errorf("Len expected 0, got %d", c.Len())
		}
	})

	// 测试 LFU
	t.Run("LFU", func(t *testing.T) {
		c := NewLFU(25, onEvicted)
		c.Put("k1", &mockValue{10})
		c.Get("k1") // cnt=2
		c.Put("k2", &mockValue{20}) // k2 cnt=1 移除
		if _, ok := c.Get("k2"); ok {
			t.Errorf("k2 should be evicted")
		}
		c.storage["k1"].entry.timeout = time.Now().Add(-1 * time.Second)
		c.Cleanup()
		if c.Len() != 0 {
			t.Errorf("Len expected 0, got %d", c.Len())
		}
	})
}

func TestMain(m *testing.M) {
	ttl = 2 * time.Second
	m.Run()
}