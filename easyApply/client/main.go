// grpc-etcd/grpc-client/etcd.go

package main

import (
	"context"
	"easyApply/pb"
	"fmt"
	"log"

	eresolver "go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	// etcd
	eclient "go.etcd.io/etcd/client/v3"
)

const (
	serverNamePreResolve = "etcd:///ayang/server1"
	EtcdAddr             = "http://localhost:2379"
)

func main() {
	var err error
	// 创建 etcd 客户端
	etcdClient, err := eclient.NewFromURL(EtcdAddr)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}

	// 创建 etcd 实现的 grpc 服务注册发现模块 resolver
	etcdResolverBuilder, err := eresolver.NewBuilder(etcdClient)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}

	// 创建 grpc 连接代理
	conn, err := grpc.Dial(
		// 服务名称
		serverNamePreResolve,
		// 注入 etcd resolver
		grpc.WithResolvers(etcdResolverBuilder),
		// 声明使用的负载均衡策略为 roundrobin，轮询。（测试 target 时去除该注释）
		// grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, roundrobin.Name)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}

	for i := 0; i < 4; i++ {
		greeter1 := pb.NewGreeter1Client(conn)
		resp, err := greeter1.SayHello1(context.Background(), &pb.HelloRequest{
			Name: "ayang",
		})

		if err != nil {
			log.Fatalln(err.Error())
			return
		}

		fmt.Printf("%d  %s\n", i, resp.Reply)
	}

	defer conn.Close()
}
