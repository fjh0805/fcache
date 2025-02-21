package strategies

import (
	"fmt"
	"testing"
	"time"
)

// mockValue 实现 Value 接口
type mockValue struct {
	size int
}

func (m *mockValue) Len() int { return m.size }

func TestCacheFIFO(t *testing.T) {
	// 测试回调函数
	onEvicted := func(key string, value Value) {
		fmt.Printf("Evicted: key=%s, value=%d\n", key, value.(*mockValue).size)
	}

	// 创建 FIFO 缓存
	c := NewFIFO(100, onEvicted)

	// 测试 Put 和 Get
	t.Run("PutAndGet", func(t *testing.T) {
		c.Put("k1", &mockValue{10})
		c.Put("k2", &mockValue{20})
		if v, ok := c.Get("k1"); !ok || v.(*mockValue).size != 10 {
			t.Errorf("Get k1 failed, got %v, want 10", v)
		}
		if c.Len() != 2 {
			t.Errorf("Len expected 2, got %d", c.Len())
		}
	})

	// 测试过期清理
	t.Run("Cleanup", func(t *testing.T) {
		c := NewFIFO(100, onEvicted)
		c.Put("k1", &mockValue{10})
		c.Put("k2", &mockValue{20})
		c.storage["k1"].Value.(*entry).timeout = time.Now().Add(-1 * time.Second) // k1 过期
		time.Sleep(1 * time.Second)
		c.Cleanup()
		if _, ok := c.Get("k1"); ok {
			t.Errorf("k1 should be evicted")
		}
		if c.Len() != 1 {
			t.Errorf("Len expected 1, got %d", c.Len())
		}
	})

	// 测试容量限制
	t.Run("MaxBytes", func(t *testing.T) {
		c := NewFIFO(25, onEvicted)
		c.Put("k1", &mockValue{10})
		c.Put("k2", &mockValue{20}) // 超出 25，k1 移除
		if _, ok := c.Get("k1"); ok {
			t.Errorf("k1 should be evicted due to maxBytes")
		}
		if c.Len() != 1 {
			t.Errorf("Len expected 1, got %d", c.Len())
		}
	})

	// 测试定时清理
	t.Run("PeriodicCleanup", func(t *testing.T) {
		c := NewFIFO(100, onEvicted)
		c.Put("k1", &mockValue{10})
		c.storage["k1"].Value.(*entry).timeout = time.Now().Add(1 * time.Second)
		time.Sleep(3 * time.Second) // 等待定时器
		if _, ok := c.Get("k1"); ok {
			t.Errorf("k1 should be evicted by periodic cleanup")
		}
	})
}

func TestMain(m *testing.M) {
	ttl = 2 * time.Second // 固定 ttl 测试
	m.Run()
}