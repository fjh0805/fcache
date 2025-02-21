package grpc

import (
	"context"
	"log"
	"testing"
	"time"

	pb "github.com/limerence-yu/fcache/cachepb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

func TestGrpc(t *testing.T) {
	// 1. 连接ETCD
	etcdCli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"}, // 根据你的etcd地址修改
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("连接etcd失败: %v", err)
	}
	defer etcdCli.Close()

	// 2. 获取集群节点
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	resp, err := etcdCli.Get(ctx, "clusters/", clientv3.WithPrefix())
	cancel()
	if err != nil {
		log.Fatalf("查询etcd失败: %v", err)
	}

	// 提取节点地址
	var addrs []string
	for _, kv := range resp.Kvs {
		addrs = append(addrs, string(kv.Value))
	}

	if len(addrs) < 2 {
		log.Fatal("至少需要2个节点进行测试")
	}
	testNodes := addrs[:1] // 取前两个节点测试
	log.Println(testNodes)
	// 3. 测试gRPC连接
	for _, addr := range testNodes {
		// 建立gRPC连接
		conn, err := grpc.Dial(
			"etcd:///GroupCache",
			grpc.WithInsecure(),
			grpc.WithBlock(),
			grpc.WithTimeout(3*time.Second),
		)
		log.Printf("conn %v, err %s", conn, err)
		if err != nil {
			log.Printf("连接失败 [%s]: %v", addr, err)
			continue
		}
		defer conn.Close()

		// 调用Put方法测试
		client := pb.NewGroupCacheClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// 假设你的PutRequest结构如下：
		// message PutRequest { string key = 1; bytes value = 2; }
		_, err = client.Get(ctx, &pb.Request{
			Group: "scores",
			Key:   "Tom",
		})
		if err != nil {
			log.Printf("调用Put失败 [%s]: %v", addr, err)
		} else {
			log.Printf("节点 [%s] 测试成功", addr)
		}
	}
}
