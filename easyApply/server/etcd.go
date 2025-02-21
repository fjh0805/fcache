// grpc-etcd/grpc-server/etcd.go

package main

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/etcd/client/v3/naming/endpoints"
	eclient "go.etcd.io/etcd/client/v3"
)

const (
	// etcd 服务器的地址
	EtcdAddr = "http://localhost:2379"
)

func registerEndPointToEtcd(ctx context.Context, serverAddr, serverName string) {
	// 创建 etcd 客户端
	etcdClient, _ := eclient.NewFromURL(EtcdAddr)
	etcdManager, _ := endpoints.NewManager(etcdClient, serverName)

	// 创建一个租约，每隔 10s 需要向 etcd 汇报一次心跳，证明当前节点仍然存活
	var ttl int64 = 10
	lease, _ := etcdClient.Grant(ctx, ttl)

	// 添加注册节点到 etcd 中，并且携带上租约 id
	// 以 serverName/serverAddr 为 key，serverAddr 为 value
	// serverName/serverAddr 中的 serverAddr 可以自定义，只要能够区分同一个 grpc 服务器功能的不同机器即可
	_ = etcdManager.AddEndpoint(ctx, fmt.Sprintf("%s/%s", serverName, serverAddr), endpoints.Endpoint{Addr: serverAddr}, eclient.WithLease(lease.ID))

	// 每隔 5 s进行一次延续租约的动作
	for {
		select {
		case <-time.After(5 * time.Second):
			// 续约操作
			resp, _ := etcdClient.KeepAliveOnce(ctx, lease.ID)
			fmt.Printf("keep alive resp: %+v\n", resp)
		case <-ctx.Done():
			return
		}
	}
}
