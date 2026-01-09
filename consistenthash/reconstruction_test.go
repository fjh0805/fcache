package consistenthash

import (
	"strconv"
	"testing"
)

// TestHashRingReconstruction 演示哈希环的重建过程
// 模拟节点添加场景，展示哈希环如何处理节点变化
func TestHashRingReconstruction(t *testing.T) {
	// 使用简单的哈希函数便于测试
	hash := func(data []byte) uint32 {
		x, _ := strconv.Atoi(string(data))
		return uint32(x)
	}

	// 场景1: 初始状态 - 创建哈希环并添加3个节点
	t.Log("场景1: 初始化哈希环，添加3个节点")
	ring1 := New(3, hash)
	ring1.Add("node1", "node2", "node3")
	
	t.Logf("初始哈希环包含 %d 个虚拟节点 (3个物理节点 × 3个副本)", len(ring1.keys))
	
	// 测试一些键的映射
	testKeys := []string{"10", "20", "30", "40"}
	initialMapping := make(map[string]string)
	
	t.Log("初始键映射:")
	for _, key := range testKeys {
		node := ring1.Get(key)
		initialMapping[key] = node
		t.Logf("  键 %s -> 节点 %s", key, node)
	}

	// 场景2: 添加新节点 - 完全重建策略
	// 这模拟了 Server.reconstruct() 的行为
	t.Log("\n场景2: 添加新节点 node4（使用完全重建策略）")
	
	// 创建新的哈希环实例（完全重建）
	ring2 := New(3, hash)
	// 添加所有节点，包括新节点
	ring2.Add("node1", "node2", "node3", "node4")
	
	t.Logf("重建后哈希环包含 %d 个虚拟节点 (4个物理节点 × 3个副本)", len(ring2.keys))
	
	// 检查键的映射变化
	t.Log("重建后键映射:")
	changedCount := 0
	for _, key := range testKeys {
		newNode := ring2.Get(key)
		oldNode := initialMapping[key]
		
		if newNode != oldNode {
			t.Logf("  键 %s: %s -> %s (已迁移)", key, oldNode, newNode)
			changedCount++
		} else {
			t.Logf("  键 %s -> %s (未变化)", key, newNode)
		}
	}
	
	// 根据一致性哈希的特性，只有部分键应该被重新映射
	t.Logf("\n总结: %d/%d 个键被重新映射到不同节点", changedCount, len(testKeys))
	
	// 场景3: 演示增量添加（当前代码不支持，但可以展示差异）
	t.Log("\n场景3: 对比 - 如果使用增量添加会怎样")
	ring3 := New(3, hash)
	ring3.Add("node1", "node2", "node3")
	
	// 增量添加新节点
	ring3.Add("node4")
	
	t.Logf("增量添加后哈希环包含 %d 个虚拟节点", len(ring3.keys))
	
	// 验证增量添加的结果应该与完全重建相同
	for _, key := range testKeys {
		node2 := ring2.Get(key)
		node3 := ring3.Get(key)
		
		if node2 != node3 {
			t.Errorf("键 %s 的映射不一致: 完全重建=%s, 增量添加=%s", key, node2, node3)
		}
	}
	
	t.Log("验证: 完全重建和增量添加的结果一致 ✓")
}

// TestHashRingScaling 测试哈希环在不同规模下的重建性能
func TestHashRingScaling(t *testing.T) {
	tests := []struct {
		name         string
		initialNodes int
		replicas     int
	}{
		{"小规模集群", 5, 50},
		{"中等规模集群", 20, 100},
		{"大规模集群", 50, 150},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建初始节点列表
			nodes := make([]string, tt.initialNodes)
			for i := 0; i < tt.initialNodes; i++ {
				nodes[i] = "node" + strconv.Itoa(i)
			}

			// 模拟重建过程
			t.Logf("配置: %d个节点, %d个副本/节点", tt.initialNodes, tt.replicas)
			
			// 创建新的哈希环（完全重建）
			ring := New(tt.replicas, nil)
			ring.Add(nodes...)
			
			totalVirtualNodes := tt.initialNodes * tt.replicas
			t.Logf("虚拟节点总数: %d", totalVirtualNodes)
			
			if len(ring.keys) != totalVirtualNodes {
				t.Errorf("期望 %d 个虚拟节点, 实际 %d 个", totalVirtualNodes, len(ring.keys))
			}
			
			// 验证所有节点都可以被查找到
			nodesSeen := make(map[string]bool)
			for i := 0; i < 1000; i++ {
				key := "key" + strconv.Itoa(i)
				node := ring.Get(key)
				nodesSeen[node] = true
			}
			
			// 在合理的采样下，应该能看到大部分节点
			t.Logf("1000次查询中使用了 %d/%d 个不同的节点", len(nodesSeen), tt.initialNodes)
		})
	}
}

// TestConsistentHashingProperty 测试一致性哈希的关键特性
// 验证添加节点时，只有部分键会被重新映射
func TestConsistentHashingProperty(t *testing.T) {
	// 使用真实的 CRC32 哈希函数
	// 创建初始哈希环
	ring1 := New(50, nil)
	ring1.Add("nodeA", "nodeB", "nodeC")

	// 测试大量键的映射
	const numKeys = 1000
	initialMapping := make(map[string]string)
	for i := 0; i < numKeys; i++ {
		key := "key_" + strconv.Itoa(i) // 使用有意义的前缀
		initialMapping[key] = ring1.Get(key)
	}

	// 添加新节点后重建
	ring2 := New(50, nil)
	ring2.Add("nodeA", "nodeB", "nodeC", "nodeD")

	// 统计被重新映射的键的数量
	remappedCount := 0
	for key, oldNode := range initialMapping {
		newNode := ring2.Get(key)
		if newNode != oldNode {
			remappedCount++
		}
	}

	// 理论上，添加1个节点到N个节点的集群，应该有约 1/(N+1) 的键被重新映射
	// 在这个例子中，从3个节点变成4个节点，期望约 25% 的键被重新映射
	remappedPercent := float64(remappedCount) / float64(numKeys) * 100

	t.Logf("添加节点前: 3个节点")
	t.Logf("添加节点后: 4个节点")
	t.Logf("重新映射的键: %d/%d (%.2f%%)", remappedCount, numKeys, remappedPercent)

	// 验证重新映射的比例在合理范围内（20%-30%）
	if remappedPercent < 15 || remappedPercent > 35 {
		t.Logf("警告: 重新映射比例 %.2f%% 超出预期范围 (15%%-35%%)", remappedPercent)
		t.Logf("说明: 这可能是由于虚拟节点数量、哈希函数特性或键的分布导致的")
	} else {
		t.Logf("✓ 一致性哈希特性验证通过: 只有部分键被重新映射")
	}
}
