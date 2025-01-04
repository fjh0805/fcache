package strategies

import (
	"reflect"
	"testing"
)

func TestLFUGet(t *testing.T) {
	lfu := NewLFU(int64(0), nil)
	lfu.Put("k1", String("v1"))
	if v, ok := lfu.Get("k1"); !ok || string(v.(String)) != "v1" {
		t.Fatal("cache hit k1 = v1 failed")
	}
	if _, ok := lfu.Get("k2"); ok {
		t.Fatal("cache miss key2 failed")
	}
}

func TestLFURemoveoldest(t *testing.T) {
	k1, k2, k3 := "k1", "k23", "k3"
	v1, v2, v3 := "v1", "v23", "v3"
	maxBytes := len(k1 + k2 + v1 + v2) // 10
	lfu := NewLFU(int64(maxBytes), nil)
	lfu.Put(k1, String(v1))
	lfu.Put(k2, String(v2))
	lfu.Get(k1)
	lfu.Put(k3, String(v3))
	//不能取到k2
	if _, ok := lfu.Get(k2); ok || lfu.Len() != 2 {
		t.Fatalf("removeOldest k2 failed")
	}
}

func TestLFUOnEvicted(t *testing.T) {
	keys := []string{}
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	k1, k2, k3 := "k1", "k23", "k345"
	v1, v2, v3 := "v1", "v23", "v345"
	maxBytes := len(k1 + k2 + v1 + v2) // 10
	lfu := NewLFU(int64(maxBytes), callback)
	lfu.Put(k1, String(v1))
	lfu.Put(k2, String(v2))
	lfu.Get(k1)
	lfu.Put(k3, String(v3))

	expect := []string{"k23", "k345"} 
	if !reflect.DeepEqual(keys, expect) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s, but %s", expect, keys)
	}
}
