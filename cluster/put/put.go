package main

import (
	"context"
	"fmt"
	"time"


	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	//初始化
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		fmt.Println("new clientv3 failed,err:", err)
		return
	}

	fmt.Println("connect to etcd success!")
	defer cli.Close()

	//put
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// cluster is prefix
	_, err = cli.Put(ctx, fmt.Sprintf("clusters/%s", "localhost:9999"), "localhost:9999")
	if err != nil {
		return
	}
	_, err = cli.Put(ctx, fmt.Sprintf("clusters/%s", "localhost:10000"), "localhost:10000")
	if err != nil {
		return
	}
	_, err = cli.Put(ctx, fmt.Sprintf("clusters/%s", "localhost:10001"), "localhost:10001")
	if err != nil {
		return
	}

	fmt.Println("put groupcache service to etcd success!")
}
