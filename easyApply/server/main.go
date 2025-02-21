// grpc-etcd/grpc-server/mian.go

package main

import (
	"context"
	"google.golang.org/grpc"
	"easyApply/pb"
	"log"
	"net"
)

// 服务1
type server1 struct {
	pb.UnimplementedGreeter1Server
}

func (server1) SayHello1(context.Context, *pb.HelloRequest) (*pb.HelloResponse, error) {
	resp := new(pb.HelloResponse)
	resp.Reply = "server1:hello"
	return resp, nil
}

// 服务2
type server2 struct {
	pb.UnimplementedGreeter2Server
}

func (server2) SayHello2(context.Context, *pb.HelloRequest) (*pb.HelloResponse, error) {
	resp := new(pb.HelloResponse)
	resp.Reply = "server2:hello"
	return resp, nil
}

const (
	ServerAddr1 = "127.0.0.1:8080"
	ServerAddr2 = "127.0.0.1:8081"
	ServerName1 = "ayang/server1"
	ServerName2 = "ayang/server2"
)

func main() {
	var err error
	// 1. 创建两个 tcp 连接
	conn1, err := net.Listen("tcp", ServerAddr1)
	conn2, err := net.Listen("tcp", ServerAddr2)

	if err != nil {
		log.Fatal(err.Error())
		return
	}

	// 2. 创建两个 grpc 服务器
	s1 := grpc.NewServer()
	s2 := grpc.NewServer()

	// 3. 注册到 grpc 服务器中
	pb.RegisterGreeter1Server(s1, &server1{})
	pb.RegisterGreeter2Server(s2, &server2{})

	// 4. 注册到 etcd 中
	go registerEndPointToEtcd(context.TODO(), ServerAddr1, ServerName1)
	go registerEndPointToEtcd(context.TODO(), ServerAddr2, ServerName2)

	go func() {
		err = s1.Serve(conn1)
		if err != nil {
			log.Fatal(err.Error())
			return
		}
	}()

	go func() {
		err = s2.Serve(conn2)
		if err != nil {
			log.Fatal(err.Error())
			return
		}
	}()

	<-make(chan struct{})

}
