package consistenthash

import (
	"log"
	"strconv"
	"testing"
)

func TestHash(t *testing.T) {
	m := New(3, func(data []byte) uint32 {
		x, _ := strconv.Atoi(string(data))
		return uint32(x)
	})
	//2 4 6 12 14 16 22 24 26
	m.Add("2", "4", "6")
	log.Printf("m.keys %v", m.keys)
	record := map[string]string{
		"27": "2",
		"11": "2",
		"13": "4",
		"25": "6",
	}

	for k, v := range record {
		if m.Get(k) != v {
			t.Fatalf("Asking for %s, should have yielded %s", k, v)
		}
	}
	m.Add("8")
	record["27"] = "8"
	for k, v := range record {
		if m.Get(k) != v {
			t.Fatalf("Asking for %s, should have yielded %s", k, v)
		}
	}
}
