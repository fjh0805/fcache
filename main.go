package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/limerence-yu/fcache/cache"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *cache.Group {
	return cache.NewGroup("scores", 2<<10, cache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func startAPIServer(apiAddr string, f *cache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := f.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func startCacheServer(addr string, addrs []string, f *cache.Group) {
	peers := cache.NewHTTPPool(addr)
	peers.Set(addrs...)
	f.RegisterPeers(peers)
	log.Println("cache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "cache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	f := createGroup()
	//脚本会启动三个缓存服务器，其中一个既作为api服务器也作为缓存服务器
	if api {
		//开启一个服务，监听在 http://localhost:9999 上，并处理 /api 路径的请求。
		//在这里f.Get(key)，实际上只有是api服务器的有的f才可以调用这里的方法
		//对应的 ./server -port=8003 -api=1 &
		//也就是说当你有一个这样的请求 curl "http://localhost:9999/api?key=Tom" 
		//（服务器地址是http://localhost:9999，路径是/api）
		//首先匹配到了，因此调用f.Get(key)，对应的group是属于8003，因此调用group里的get
		//首先是检查自身cache是否有这个key
		//没有就通过哈希环选节点得到peer,形式[http://localhost:8001/_fcache/],
		//然后调用peer.Get，对应的也就是httpGetter.Get() u = http://localhost:8001/_fcache/group/key
		//调用http.Get(u) -> ServeHTTP -> group.Get
		//这里的group已经是8001的了，到了这里同样也是查询本地cache，但是选是不会选到自己了
		//自己缓存没有就用回调函数，找到了就返回，最终返回到8003，8003再到9999
		go startAPIServer(apiAddr, f)
	}
	startCacheServer(addrMap[port], []string(addrs), f)
}
