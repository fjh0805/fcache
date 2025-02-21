package main

// example.go file
// 运行前，你需要在本地启动Etcd实例，作为服务中心。
import (
	"flag"
	"fmt"
	"log"

	"github.com/limerence-yu/fcache/DB"
	"github.com/limerence-yu/fcache/cache"
	"github.com/limerence-yu/fcache/grpc"
	"github.com/limerence-yu/fcache/registry"
	"gorm.io/gorm"
)

// todo sql
// var mysql = map[string]string{
// 	"Tom":  "630",
// 	"Jack": "589",
// 	"Sam":  "567",
// }
// group := grpc.NewGroup("scores", 2<<10, cache.GetterFunc(
// 	func(key string) ([]byte, error) {
// 		log.Println("[Mysql] search key", key)
// 		if v, ok := mysql[key]; ok {
// 			log.Printf("find key %s, value %s", key, v)
// 			return []byte(v), nil
// 		}
// 		return nil, fmt.Errorf("%s not exist", key)
// 	}))

func mysqlGetterFunc(db *gorm.DB) cache.GetterFunc {
	return func(key string) ([]byte, error) {
		var s DB.Student
		err := db.Where("name = ?", key).First(&s).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				log.Printf("name %s not found in mysql", key)
				return nil, fmt.Errorf("%s not exist", key)
			}
			log.Printf("error querying MySQL for name %s: %v", key, err)
			return nil, fmt.Errorf("failed to get name %s from MySQL: %v", key, err)
		}
		log.Printf("mysql found name %s, value %s", key, s.Score)
		return []byte(s.Score), nil
	}
}

var (
	port        = flag.Int("port", 9999, "port")
	serviceName = "GroupCache"
)

func main() {
	flag.Parse()
	db, _ := DB.Init()
	// 新建cache实例,如果改为mysql回调函数要修改
	group := grpc.NewGroup("scores", 2<<10, mysqlGetterFunc(db))
	// New一个服务实例
	addr := fmt.Sprintf("localhost:%d", *port)
	updatechan := make(chan struct{}) //用于监控节点变化
	svr, err := grpc.NewServer(updatechan, addr)
	if err != nil {
		log.Fatal(err)
	}
	go registry.WatchUpdate(updatechan, serviceName)
	// 设置同伴节点IP(包括自己)
	// todo: 这里的peer地址从etcd获取(服务发现)
	addrs, err := registry.DiscoverPeer(serviceName)
	log.Printf("get addrs %v from etcd", addrs)
	if err != nil { //查询失败使用默认地址
		addrs = []string{"localhost:9999"}
	}
	svr.SetPeers(addrs)
	// 将服务与cache绑定 因为cache和server是解耦合的
	group.RegisterSvr(svr)
	log.Println("fcache is running at", addr)

	svr.Start()
}
