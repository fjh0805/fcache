package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type Map struct {
	replicas int
	hash     Hash
	keys     []int
	hashMap  map[int]string
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		keys:     []int{},
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			temp := strconv.Itoa(i) + key
			hash := int(m.hash([]byte(temp)))
			m.hashMap[hash] = key
			m.keys = append(m.keys, hash)
		}
	}
	sort.Ints(m.keys)
}

func (m *Map) Get(key string) string {
	hash := int(m.hash([]byte(key)))
	//没找到符合条件的就是插入位置
	idx := sort.Search(len(m.keys), func(i int) bool {
		return hash <= m.keys[i]
	})
	idx %= len(m.keys)
	// idx := m.find(hash)
	return m.hashMap[m.keys[idx]]
}

func (m *Map) find(hash int) int {
	keys := m.keys
	i, j := 0, len(keys) - 1
	for i <= j {
		mid := i + (j - i) / 2
		if keys[mid] == hash {
			return mid
		} else if keys[mid] < hash {
			i = mid + 1
		} else {
			j = mid - 1
		}
	} 
	return i % len(keys)
}
