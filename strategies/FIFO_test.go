package strategies

import (
	"reflect"
	"testing"
)

func TestFIFOGet(t *testing.T) {
	fifo := NewFIFO(int64(0), nil)
	fifo.Put("k1", String("v1"))
	if v, ok := fifo.Get("k1"); !ok || string(v.(String)) != "v1" {
		t.Fatal("cache hit k1 = v1 failed")
	}
	if _, ok := fifo.Get("k2"); ok {
		t.Fatal("cache miss key2 failed")
	}
}

func TestFIFORemoveoldest(t *testing.T) {
	k1, k2, k3 := "k1", "k23", "k3"
	v1, v2, v3 := "v1", "v23", "v3"
	maxBytes := len(k1 + k2 + v1 + v2) // 10
	fifo := NewFIFO(int64(maxBytes), nil)
	fifo.Put(k1, String(v1))
	fifo.Put(k2, String(v2))
	fifo.Get(k1)
	fifo.Put(k3, String(v3))
	// 不能取到k1
	if _, ok := fifo.Get(k1); ok || fifo.Len() != 2 {
		t.Fatalf("removeOldest k1 failed")
	}
}

func TestFIFOOnEvicted(t *testing.T) {
	keys := []string{}
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	k1, k2, k3 := "k1", "k23", "k345"
	v1, v2, v3 := "v1", "v23", "v345"
	maxBytes := len(k1 + k2 + v1 + v2) // 10
	fifo := NewFIFO(int64(maxBytes), callback)
	fifo.Put(k1, String(v1))
	fifo.Put(k2, String(v2))
	fifo.Put(k3, String(v3))

	expect := []string{"k1", "k23"}
	if !reflect.DeepEqual(keys, expect) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s", expect)
	}
}
