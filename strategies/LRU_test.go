package strategies

import (
	"reflect"
	"testing"
)

type String string

func (s String) Len() int {
	return len(s)
}

func TestGet(t *testing.T) {
	lru := NewLRU(int64(0), nil)
	lru.Put("k1", String("v1"))
	if v, ok := lru.Get("k1"); !ok || string(v.(String)) != "v1" {
		t.Fatal("cache hit k1 = v1 failed")
	}
	if _, ok := lru.Get("k2"); ok {
		t.Fatal("cache miss key2 failed")
	}
}

func TestRemoveoldest(t *testing.T) {
	k1, k2, k3 := "k1", "k23", "k3"
	v1, v2, v3 := "v1", "v23", "v3"
	maxBytes := len(k1 + k2 + v1 + v2) // 10
	lru := NewLRU(int64(maxBytes), nil)
	lru.Put(k1, String(v1))
	lru.Put(k2, String(v2))
	lru.Get(k1)
	lru.Put(k3, String(v3))
	//不能取到k2
	if _, ok := lru.Get(k2); ok || lru.Len() != 2 {
		t.Fatalf("removeOldest k2 failed")
	}
}

func TestOnEvicted(t *testing.T) {
	keys := []string{}
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	k1, k2, k3 := "k1", "k23", "k345"
	v1, v2, v3 := "v1", "v23", "v345"
	maxBytes := len(k1 + k2 + v1 + v2) // 10
	lru := NewLRU(int64(maxBytes), callback)
	lru.Put(k1, String(v1))
	lru.Put(k2, String(v2))
	lru.Put(k3, String(v3))

	expect := []string{"k1", "k23"} 
	if !reflect.DeepEqual(keys, expect) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s", expect)
	}
}
