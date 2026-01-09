package registry

import (
	"context"
	"log"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

/*
ClientConn 表示与概念端点的虚拟连接，用于执行 RPC，ClientConn 可根据配置、负载等情况，与端点自由建立零个或多个实际连接。
*/
func EtcdDial(c *clientv3.Client, service string) (*grpc.ClientConn, error) {
	// NewBuilder creates a parser builder. It is used to parse the request path sent by the client to identify the object to connect to
	etcdResolver, err := resolver.NewBuilder(c)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.NewClient("etcd:///"+service, grpc.WithResolvers(etcdResolver),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	log.Printf("target: %s", conn.Target())	
	return conn, err
}

// WatchUpdate 监听 etcd 中服务节点的变化
// 当有节点增加或删除时，通过 updatechan 通知上层进行哈希环重建
//
// 监听流程：
// 1. 连接 etcd 客户端
// 2. 使用 Watch API 监听指定服务名称前缀的所有键
// 3. 当检测到 PUT 事件（节点注册/更新）时，发送更新信号
// 4. 当检测到 DELETE 事件（节点下线）时，发送更新信号
// 5. 上层收到信号后，会重新从 etcd 获取所有存活节点并重建哈希环
//
// 节点变化检测机制：
// - 使用 etcd 的 Watch API 实现实时监听
// - 不区分增加和删除，统一触发重建
// - 重建时会从 etcd 获取最新的完整节点列表
// - 这样确保哈希环状态与 etcd 中的注册信息一致
//
// 注意：此函数会阻塞，应在单独的 goroutine 中调用
func WatchUpdate(updatechan chan struct{}, serviceName string) {
	cli, err := clientv3.New(defaultEtcdConfig)
	if err != nil {
		log.Printf("connect etcd failed, err %s", err)
	}
	defer cli.Close()

	watchChan := cli.Watch(context.Background(), serviceName, clientv3.WithPrefix())
	//for range 当通道关闭（close(ch)）时，循环自动退出。 阻塞等待watchchan数据
	//适合单向channel
	for watchresp := range watchChan {
		for _, event := range watchresp.Events {
			switch event.Type {
			case clientv3.EventTypePut:
				updatechan <- struct{}{}
				log.Printf("Service endpoint added or updated: %s", string(event.Kv.Key))
			case clientv3.EventTypeDelete:
				updatechan <- struct{}{}
				log.Printf("Service endpoint delete: %s", string(event.Kv.Key))
			}
		}
	}
}
//发现存活的节点
//创建客户端 -> 初始化端点管理器 -> 获取键值对 -> 提取地址 -> 返回。
func DiscoverPeer(serviceName string) ([]string, error) {
	cli, err := clientv3.New(defaultEtcdConfig)
	if err != nil {
		log.Printf("connect etcd failed, err %s", err)
	}
	defer cli.Close()
	manager, err := endpoints.NewManager(cli, serviceName)
	if err != nil {
		log.Printf("create endpoints manager failed %v", err)
		return nil, err
	}
	key2endpoint, err := manager.List(context.Background())
	if err != nil {
		log.Printf("list endpoint node for target service failed, err : %v", err)
	}
	peerAddr := []string{}
	for _, endpoint := range key2endpoint {
		peerAddr = append(peerAddr, endpoint.Addr)
		//log.Printf("found endpoint addr: %s (%s):(%v)", key, endpoint.Addr, endpoint.Metadata)
	}
	log.Printf("当前存活节点 %v", peerAddr)
	return peerAddr, nil
}