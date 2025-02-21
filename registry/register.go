package registry

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
)

var (
	defaultEtcdConfig = clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}
)

// 在租凭模式添加一对kv至etcd
func etcdAdd(c *clientv3.Client, lid clientv3.LeaseID, service string, addr string) error {
	em, err := endpoints.NewManager(c, service)
	if err != nil {
		return err
	}
	return em.AddEndpoint(c.Ctx(), service+"/"+addr, endpoints.Endpoint{Addr: addr, Metadata: "fcache services"}, clientv3.WithLease(lid))
}

// 注册一个服务至etcd
func Register(service string, addr string, stop chan error) error {
	cli, err := clientv3.New(defaultEtcdConfig)
	if err != nil {
		return fmt.Errorf("create etcd client failed: %v", err)
	}
	defer cli.Close()
	//创建一个租约，五秒过期
	resp, err := cli.Grant(context.Background(), 5)
	if err != nil {
		return fmt.Errorf("create lease failed: %v", err)
	}
	leaseID := resp.ID
	//注册服务
	err = etcdAdd(cli, leaseID, service, addr)
	if err != nil {
		return fmt.Errorf("add etcd failed: %v", err)
	}
	//设置服务心跳检测
	ch, err := cli.KeepAlive(context.Background(), leaseID)
	if err != nil {
		return fmt.Errorf("set keepalive failed: %v", err)
	}

	log.Printf("[%s] register service ok\n", addr)
	for {
		select {
		case err := <-stop:
			if err != nil {
				log.Println(err)
			}
			// 主动撤销租约
			_, revokeErr := cli.Revoke(context.Background(), leaseID)
			if revokeErr != nil {
				log.Printf("Failed to revoke lease: %v", revokeErr)
			}
			return err
		case <-cli.Ctx().Done():
			log.Println("service closed")
			return nil
		case _, ok := <-ch:
			if !ok {
				log.Println("keep alive channel closed")
				_, err := cli.Revoke(context.Background(), leaseID)
				return err
			}
		}
	}
}
