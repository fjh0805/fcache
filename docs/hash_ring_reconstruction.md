# 哈希环重建机制详解

## 概述

fcache 使用一致性哈希算法来实现分布式缓存的负载均衡。当集群中的节点数量发生变化（增加或删除节点）时，系统需要重建哈希环以确保缓存键能够正确路由到相应的节点。

## 架构组件

### 1. 一致性哈希环 (consistenthash.Map)

一致性哈希环是核心数据结构，负责将缓存键映射到物理节点。

**关键特性：**
- 使用虚拟节点技术，每个物理节点对应多个虚拟节点（默认50个）
- 虚拟节点在哈希环上均匀分布，提高负载均衡效果
- 使用二分查找快速定位节点，时间复杂度 O(log N)

**数据结构：**
```go
type Map struct {
    replicas int            // 每个物理节点对应的虚拟节点数量
    hash     Hash            // 哈希函数
    keys     []int           // 已排序的虚拟节点哈希值
    hashMap  map[int]string  // 哈希值 -> 物理节点映射
}
```

### 2. 服务注册中心 (registry)

使用 etcd 作为服务注册中心，实现服务发现和节点变化监听。

**主要功能：**
- `Register()`: 将节点注册到 etcd，并维持心跳
- `DiscoverPeer()`: 从 etcd 获取所有存活节点
- `WatchUpdate()`: 监听节点变化事件

### 3. gRPC 服务器 (grpc.Server)

管理哈希环和节点间通信。

**核心方法：**
- `SetPeers()`: 初始化哈希环和节点客户端
- `reconstruct()`: 重建哈希环
- `Pick()`: 根据键选择目标节点

## 节点添加时的哈希环重建流程

### 完整流程图

```
新节点启动
    ↓
1. 调用 Register() 将节点信息注册到 etcd
    ↓
2. etcd 触发 PUT 事件
    ↓
3. WatchUpdate() 检测到变化，发送信号到 updatechan
    ↓
4. Server.SetPeers() 中的监听协程收到信号
    ↓
5. 调用 reconstruct() 方法
    ↓
6. 从 etcd 获取所有存活节点列表 (DiscoverPeer)
    ↓
7. 创建新的哈希环实例
    ↓
8. 将所有节点（包括新节点）添加到哈希环
    ↓
9. 为每个节点创建 gRPC 客户端
    ↓
10. 更新 Server 的哈希环和客户端映射
    ↓
哈希环重建完成
```

### 详细步骤说明

#### 步骤 1-3: 节点注册与变化检测

当新节点启动时：

```go
// main.go
svr.Start() // 在内部调用 registry.Register()
```

Register 函数会：
1. 创建 etcd 租约（默认5秒过期）
2. 将节点信息（服务名+地址）写入 etcd
3. 启动 KeepAlive 维持租约

```go
// registry/register.go
func Register(service string, addr string, stop chan error) error {
    // 创建租约
    resp, err := cli.Grant(context.Background(), 5)
    if err != nil {
        return fmt.Errorf("create lease failed: %v", err)
    }
    
    // 注册服务
    err = etcdAdd(cli, resp.ID, service, addr)
    if err != nil {
        return fmt.Errorf("add etcd failed: %v", err)
    }
    
    // 心跳保活
    ch, err := cli.KeepAlive(context.Background(), resp.ID)
    if err != nil {
        return fmt.Errorf("set keepalive failed: %v", err)
    }
    // ...
}
```

同时，WatchUpdate 协程一直在监听：

```go
// registry/discover.go
func WatchUpdate(updatechan chan struct{}, serviceName string) {
    watchChan := cli.Watch(context.Background(), serviceName, clientv3.WithPrefix())
    
    for watchresp := range watchChan {
        for _, event := range watchresp.Events {
            if event.Type == clientv3.EventTypePut {
                updatechan <- struct{}{} // 发送更新信号
            }
        }
    }
}
```

#### 步骤 4-5: 触发重建

Server.SetPeers() 中的监听协程：

```go
// grpc/server.go
func (s *Server) SetPeers(peersAddr []string) {
    // ...初始化...
    
    go func() {
        for {
            select {
            case <-s.updatechan:  // 收到更新信号
                s.reconstruct()    // 触发重建
            // ...
            }
        }
    }()
}
```

#### 步骤 6-10: 哈希环重建

reconstruct 方法执行完全重建：

```go
// grpc/server.go
func (s *Server) reconstruct() {
    // 1. 从 etcd 获取所有存活节点
    serviceAddr, err := registry.DiscoverPeer(serviceName)
    
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // 2. 创建新的哈希环（丢弃旧的）
    s.Hash = consistenthash.New(defaultReplicas, nil)
    
    // 3. 添加所有节点到哈希环
    s.Hash.Add(serviceAddr...)
    
    // 4. 为每个节点创建客户端
    for _, peerAddr := range serviceAddr {
        s.clients[peerAddr] = NewClient(serviceName, peerAddr)
    }
    
    log.Printf("hash ring reconstruct")
}
```

Add 方法的内部实现：

```go
// consistenthash/consistenthash.go
func (m *Map) Add(keys ...string) {
    for _, key := range keys {
        // 为每个节点创建多个虚拟节点
        for i := 0; i < m.replicas; i++ {
            // 生成虚拟节点标识：编号+物理节点地址
            temp := strconv.Itoa(i) + key
            hash := int(m.hash([]byte(temp)))
            
            // 记录映射关系
            m.hashMap[hash] = key
            m.keys = append(m.keys, hash)
        }
    }
    // 排序以支持二分查找
    sort.Ints(m.keys)
}
```

## 重建策略分析

### 当前实现：完全重建策略

**特点：**
- 每次节点变化都创建新的哈希环实例
- 重新添加所有节点（包括未变化的节点）
- 丢弃旧的哈希环和所有映射关系

**优点：**
1. **实现简单**：逻辑清晰，易于理解和维护
2. **状态一致**：确保哈希环与 etcd 中的注册信息完全一致
3. **避免复杂性**：不需要处理增量更新的各种边界情况
4. **性能足够**：哈希计算很快，即使有大量节点也能快速完成

**缺点：**
1. **重复计算**：未变化的节点也需要重新计算哈希
2. **内存分配**：每次都创建新的数据结构
3. **扩展性限制**：节点数量极大时可能有性能影响

### 性能分析

假设：
- 节点数量：N
- 虚拟节点数：R（默认50）
- 总虚拟节点数：N × R

**时间复杂度：**
- 创建虚拟节点：O(N × R)
- 哈希计算：O(N × R)
- 排序：O(N × R × log(N × R))
- 总计：O(N × R × log(N × R))

**实际性能：**
- 10个节点：500个虚拟节点，排序约4500次比较
- 100个节点：5000个虚拟节点，排序约45000次比较
- 对于大多数场景，这个开销完全可以接受

## 数据迁移

当哈希环重建后，部分缓存键会被重新映射到不同的节点。

### 受影响的键

根据一致性哈希的特性：
- **新增节点**：只有顺时针相邻的下一个节点的部分键会迁移到新节点
- **删除节点**：该节点的所有键会迁移到顺时针下一个节点
- **未受影响的键**：大部分键的映射关系保持不变

### 迁移处理

fcache 采用**惰性迁移**策略：
1. 哈希环重建后，不主动迁移数据
2. 当访问某个键时，根据新的哈希环路由到正确的节点
3. 如果目标节点没有该键，则从源数据库加载（回源）
4. 旧节点上的过期数据会通过 TTL 机制自动清理

**优点：**
- 无需停机维护
- 不占用额外的迁移带宽
- 数据访问驱动，按需加载

**缺点：**
- 迁移过程中可能有缓存未命中
- 短期内可能有重复数据

## 使用示例

### 启动节点

```bash
# 启动第一个节点（端口9999）
go run main.go -port=9999

# 启动第二个节点（端口9998）
# 第二个节点启动后，所有节点会自动重建哈希环
go run main.go -port=9998

# 启动第三个节点（端口9997）
go run main.go -port=9997
```

### 日志输出示例

节点启动时：
```
[localhost:9999] register service ok
当前存活节点 [localhost:9999]
fcache is running at localhost:9999
```

新节点加入时，其他节点会输出：
```
Service endpoint added or updated: GroupCache/localhost:9998
当前存活节点 [localhost:9999 localhost:9998]
hash ring reconstruct
```

## 最佳实践

### 1. 节点命名

使用 IP:Port 格式，确保唯一性：
```go
addr := fmt.Sprintf("192.168.1.10:%d", port)
```

### 2. 虚拟节点数量

根据集群规模调整：
- 小集群（<10节点）：50-100个虚拟节点
- 中等集群（10-50节点）：100-150个虚拟节点
- 大集群（>50节点）：150-200个虚拟节点

### 3. 监控

关注以下指标：
- 哈希环重建频率
- 节点心跳状态
- 缓存命中率变化

### 4. 故障处理

节点故障时的自动处理：
1. etcd 租约过期（5秒）
2. 触发 DELETE 事件
3. 其他节点自动重建哈希环
4. 流量自动转移到其他节点

## 总结

fcache 的哈希环重建机制具有以下特点：

1. **自动化**：通过 etcd 实现节点的自动发现和变化检测
2. **实时性**：节点变化后立即触发重建，通常在秒级完成
3. **简单可靠**：采用完全重建策略，确保状态一致性
4. **性能足够**：对于大多数场景，重建开销可以忽略不计
5. **数据安全**：使用惰性迁移，避免数据丢失

这种设计在简单性和性能之间取得了良好的平衡，适合大多数分布式缓存场景。
