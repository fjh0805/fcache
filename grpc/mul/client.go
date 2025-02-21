package main

import (
	"context"
	"fmt"
	"log"
	"time"

	cachepb "github.com/limerence-yu/fcache/cachepb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials/insecure"
)

const etcdUrl = "http://localhost:2379"
const serviceName = "GroupCache"

func main() {
	cli, err := clientv3.NewFromURL(etcdUrl)
	if err != nil {
		panic(err)
	}
	etcdResolver, err := resolver.NewBuilder(cli)
	if err != nil {
		panic(err)
	}
	conn, err := grpc.Dial(fmt.Sprintf("etcd:///%s", serviceName), grpc.WithResolvers(etcdResolver), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, roundrobin.Name)))
	if err != nil {
		fmt.Printf("err: %v", err)
		return
	}
	grpcClient := cachepb.NewGroupCacheClient(conn)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		response, err := grpcClient.Get(ctx, &cachepb.Request{
			Group: "scores",
			Key:   "Tom",
		})
		if err != nil {
			log.Fatalf("%v", err.Error())
			return
		}
		log.Printf("成功从查询到 %s 的结果：%s\n", "Tom", string(response.GetValue()))
		response, err = grpcClient.Get(ctx, &cachepb.Request{
			Group: "scores",
			Key:   "Jack",
		})
		if err != nil {
			log.Fatalf("%v", err.Error())
			return
		}
		log.Printf("成功从查询到 %s 的结果：%s\n", "Jack", string(response.GetValue()))
		time.Sleep(500 * time.Millisecond)
	}
}
