package grpc

import (
	"context"
	"fmt"
	"log"
	"time"

	cachepb "github.com/limerence-yu/fcache/cachepb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	name string
	addr string
}

// func (c *client) Fetch(groupName string, key string) ([]byte, error) {
// 	// 直接使用目标节点地址建立连接
// 	conn, err := grpc.Dial(
// 		c.addr, // 使用创建 client 时传入的地址
// 		grpc.WithInsecure(),
// 		grpc.WithBlock(),
// 		grpc.WithTimeout(3*time.Second),
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("connect to peer %s failed: %v", c.addr, err)
// 	}
// 	defer conn.Close()

// 	grpcClient := cachepb.NewGroupCacheClient(conn)
// 	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
// 	defer cancel()

// 	log.Printf("[client] sending request to peer %s for group %s, key %s", c.addr, groupName, key)
// 	resp, err := grpcClient.Get(ctx, &cachepb.Request{
// 		Group: groupName,
// 		Key:   key,
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get %s/%s from peer %s: %v", groupName, key, c.addr, err)
// 	}

// 	return resp.GetValue(), nil
// }

func (c *Client) Fetch(groupName string, key string) ([]byte, error) {
	cli, err := clientv3.NewFromURL("http://localhost:2379")
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	//conn, err := registry.EtcdDial(cli, c.name)
	//直接与一致性哈希选出来的节点连接
	conn, err := grpc.NewClient(
		c.addr, // 目标地址，如 "127.0.0.1:9999"
		grpc.WithTransportCredentials(insecure.NewCredentials()), // 传输安全配置
	)
	//
	//log.Printf("[conn] %v, err %v", conn, err)
	log.Printf("here!, c.name : %s, c.addr : %s", c.name, c.addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	grpcClient := cachepb.NewGroupCacheClient(conn)
	log.Printf("[grpcClient] %v", grpcClient)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	resp, err := grpcClient.Get(ctx, &cachepb.Request{
		Group: groupName,
		Key:   key,
	})
	if err != nil {
		log.Printf("err: %s", err)
		return nil, fmt.Errorf("could not get %s/%s from peer %s", groupName, key, c.name)
	}
	return resp.GetValue(), nil
}

func NewClient(service string, addr string) *Client {
	return &Client{
		name: service,
		addr: addr,
	}
}

var _ Fetcher = (*Client)(nil)
