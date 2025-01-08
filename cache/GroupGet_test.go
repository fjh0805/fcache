package cache

import (
	"fmt"
	"log"
	"testing"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGet(t *testing.T) {
	loadCnt := make(map[string]int)
	group := NewGroup("score", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Printf("search key %s in DB", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCnt[key]; !ok {
					loadCnt[key] = 0
				}
				loadCnt[key]++
				return []byte(v), nil
			}
			return []byte{}, fmt.Errorf("key %s not exist", key)
		}))
	for k, v := range db {
		if b, err := group.Get(k); err != nil || b.String() != v {
			t.Fatal("failed to get value of Tom")
		}
		if _, err := group.Get(k); err != nil || loadCnt[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
		if _, err := group.Get(k); err != nil || loadCnt[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}
	if view, err := group.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}
