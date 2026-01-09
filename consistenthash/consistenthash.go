package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash 定义哈希函数类型，用于将字节数组映射为32位无符号整数
type Hash func(data []byte) uint32

// Map 一致性哈希环的实现
// 
// 字段说明：
// - replicas: 每个物理节点对应的虚拟节点数量，通常设置为50-150
//   虚拟节点越多，负载分布越均匀，但占用内存也越多
// 
// - hash: 哈希函数，用于计算节点和键的哈希值
//   默认使用 CRC32，也可以自定义（如 MD5、SHA1 等）
// 
// - keys: 已排序的虚拟节点哈希值数组
//   排序后可以使用二分查找，提高查找效率到 O(log N)
// 
// - hashMap: 虚拟节点哈希值到物理节点的映射
//   通过哈希值快速定位到实际的物理节点地址
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

// Add 向哈希环中添加节点
// 参数 keys 是节点的字符串标识（通常是 IP:Port）
//
// 添加流程：
// 1. 对于每个节点，创建 replicas 个虚拟节点
// 2. 每个虚拟节点的键是 "{副本编号}{节点标识}"，例如 "0localhost:9999"
// 3. 计算虚拟节点的哈希值，作为其在哈希环上的位置
// 4. 在 hashMap 中记录 哈希值->节点 的映射
// 5. 将所有哈希值添加到 keys 切片中
// 6. 对 keys 进行排序，以便后续二分查找
//
// 虚拟节点的作用：
// - 将每个物理节点映射为多个虚拟节点
// - 使节点在哈希环上分布更均匀
// - 提高负载均衡效果
// - 减少节点增删时的数据迁移量
//
// 注意：此方法会追加节点，不会清空已有节点
// 如果要完全重建哈希环，需要先创建新的 Map 实例
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

// Get 根据给定的键，返回应该存储该键的节点
// 这是一致性哈希的核心查找方法
//
// 查找流程：
// 1. 计算键的哈希值
// 2. 使用二分查找，在已排序的哈希环上找到第一个 >= 该哈希值的虚拟节点
// 3. 如果所有虚拟节点的哈希值都小于该键的哈希值，则取模运算回到环的起点
// 4. 通过 hashMap 找到该虚拟节点对应的物理节点并返回
//
// 一致性哈希的特性：
// - 环形结构：当键的哈希值大于所有节点时，通过取模回到起点
// - 顺时针查找：总是找顺时针方向第一个节点
// - 节点增删影响小：只影响相邻节点间的键分布
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
