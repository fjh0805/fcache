package main

import (
	"context"
	"log"
	"time"

	cachepb "github.com/limerence-yu/fcache/cachepb"
	"github.com/limerence-yu/fcache/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	//DB.Init()
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := registry.EtcdDial(cli, "GroupCache")
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	log.Println("gRPC client connected to server")
	grpcClient := cachepb.NewGroupCacheClient(conn)
	response, err := grpcClient.Get(ctx, &cachepb.Request{
		Group: "scores",
		Key:   "Tom",
	})
	if err != nil {
		log.Fatalf("%v", err.Error())
		return
	}
	log.Printf("成功从 RPC 返回调用结果：%s\n", string(response.GetValue()))
}
