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

func TestCacheLRU(t *testing.T) {
	// 测试回调函数
	onEvicted := func(key string, value Value) {
		fmt.Printf("Evicted: key=%s, value=%d\n", key, value.(*mockValue).size)
	}

	// 创建 LRU 缓存
	c := NewLRU(100, onEvicted)

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
		c := NewLRU(100, onEvicted)
		c.Put("k1", &mockValue{10})
		c.Put("k2", &mockValue{20})
		// 手动设置 k1 过期
		c.storage["k1"].Value.(*entry).timeout = time.Now().Add(-1 * time.Second)
		time.Sleep(1 * time.Second) // 等待过期
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
		c := NewLRU(25, onEvicted)
		c.Put("k1", &mockValue{10})
		c.Put("k2", &mockValue{20}) // 超出 25，k1 应被移除
		if _, ok := c.Get("k1"); ok {
			t.Errorf("k1 should be evicted due to maxBytes")
		}
		if c.Len() != 1 {
			t.Errorf("Len expected 1, got %d", c.Len())
		}
	})

	// 测试定时清理（简化版）
	t.Run("PeriodicCleanup", func(t *testing.T) {
		c := NewLRU(100, onEvicted)
		c.Put("k1", &mockValue{10})
		c.storage["k1"].Value.(*entry).timeout = time.Now().Add(1 * time.Second)
		time.Sleep(3 * time.Second) // 等待定时器触发
		if _, ok := c.Get("k1"); ok {
			t.Errorf("k1 should be evicted by periodic cleanup")
		}
	})
}

func TestMain(m *testing.M) {
	// 设置固定的 ttl 以便测试
	ttl = 2 * time.Second
	m.Run()
}