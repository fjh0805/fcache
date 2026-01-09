package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	pb "github.com/limerence-yu/fcache/cachepb"
	"github.com/limerence-yu/fcache/consistenthash"
	"github.com/limerence-yu/fcache/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

const (
	defaultAddr     = "127.0.0.1:6324"
	defaultReplicas = 50
	serviceName     = "GroupCache"
)

var (
	defaultEtcdConfig = clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}
)

type Server struct {
	pb.UnimplementedGroupCacheServer

	Addr       string
	Status     bool
	StopSignal chan error
	mu         sync.Mutex
	Hash       *consistenthash.Map
	clients    map[string]*Client
	updatechan chan struct{}
}

func NewServer(updatachan chan struct{}, addr string) (*Server, error) {
	if addr == "" {
		addr = defaultAddr
	}
	if !validPeerAddr(addr) {
		return nil, fmt.Errorf("invalid addr %s, it should be x.x.x.x:port", addr)
	}
	return &Server{Addr: addr, updatechan: updatachan}, nil
}

func (s *Server) Get(ctx context.Context, in *pb.Request) (*pb.Response, error) {
	group, key := in.GetGroup(), in.GetKey()
	resp := &pb.Response{}

	log.Printf("[cache_server %s] Recv RPC Request - (%s)/(%s)", s.Addr, group, key)
	if key == "" {
		return resp, fmt.Errorf("key required")
	}
	g := GetGroup(group)
	if g == nil {
		return resp, fmt.Errorf("group not found")
	}
	view, err := g.Get(key)
	if err != nil {
		return resp, err
	}
	resp.Value = view.ByteSlice()
	return resp, nil
}

// SetPeers 将各个远端主机IP配置到Server里
// 这样Server就可以Pick他们了
// 注意: 此操作是*覆写*操作！
// 注意: peersIP必须满足 x.x.x.x:port的格式
//
// 初始化流程：
// 1. 创建一致性哈希环（使用50个虚拟节点副本）
// 2. 将所有节点地址添加到哈希环中
// 3. 为每个节点创建 gRPC 客户端
// 4. 启动监听协程，等待节点变化通知
//
// 节点变化监听机制：
// - updatechan: 当 etcd 检测到节点增加/删除时，会发送信号到此通道
// - 收到信号后，调用 reconstruct() 方法重建哈希环
// - StopSignal: 用于停止服务时的通知
func (s *Server) SetPeers(peersAddr []string) {
	s.mu.Lock()

	s.Hash = consistenthash.New(defaultReplicas, nil)
	s.Hash.Add(peersAddr...)
	s.clients = make(map[string]*Client)
	for _, peerAddr := range peersAddr {
		if !validPeerAddr(peerAddr) {
			panic(fmt.Sprintf("[peer %s] invalid address format, it should be x.x.x.x:port", peerAddr))
		}
		s.clients[peerAddr] = NewClient("GroupCache", peerAddr)
	}
	s.mu.Unlock()
	go func() {
		for {
			select {
			case <-s.updatechan:
				s.reconstruct()
			case <-s.StopSignal:
				s.Stop()
			default:
				time.Sleep(2 * time.Second)
			}
		}
	}()
}

// 重新构建哈希环
// reconstruct 方法在检测到服务节点变化时被调用，用于重建一致性哈希环
// 
// 重建流程：
// 1. 从 etcd 服务注册中心获取所有存活的节点地址
// 2. 创建新的哈希环实例（使用默认的50个虚拟节点副本）
// 3. 将所有节点地址添加到哈希环中，每个节点会生成 replicas 个虚拟节点
// 4. 为每个节点创建 gRPC 客户端连接
// 5. 更新服务器的哈希环和客户端映射
//
// 注意：此方法采用完全重建策略，即每次变化都会：
//   - 丢弃旧的哈希环
//   - 创建新的哈希环实例
//   - 重新添加所有节点（包括未变化的节点）
//
// 这种方式的优点：
//   - 实现简单，逻辑清晰
//   - 确保哈希环状态与 etcd 中的服务注册信息完全一致
//   - 避免增量更新可能带来的状态不一致问题
//
// 这种方式的缺点：
//   - 每次重建都需要重新计算所有节点的哈希值
//   - 在节点数量较多时，重建开销较大
//   - 但由于哈希计算本身很快，实际影响较小
func (s *Server) reconstruct() {
	//检测到服务实例发生改变，重新构建试图
	serviceAddr, err := registry.DiscoverPeer(serviceName)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.Hash = consistenthash.New(defaultReplicas, nil)
	s.Hash.Add(serviceAddr...)
	for _, peerAddr := range serviceAddr {
		if !validPeerAddr(peerAddr) {
			panic(fmt.Sprintf("[peer %s] invalid address format, it should be x.x.x.x:port", peerAddr))
		}
		s.clients[peerAddr] = NewClient(serviceName, peerAddr)
	}
	s.mu.Unlock()
	log.Printf("hash ring reconstruct")
}

func (s *Server) Pick(key string) (Fetcher, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	peerAddr := s.Hash.Get(key)
	if peerAddr == s.Addr {
		log.Printf("ooh! pick myself, I am %s\n", s.Addr)
		return nil, false
	}
	log.Printf("[cache %s] pick remote peer: %s\n", s.Addr, peerAddr)
	return s.clients[peerAddr], true
}

func (s *Server) Start() error {
	s.mu.Lock()
	if s.Status == true {
		s.mu.Unlock()
		return fmt.Errorf("server already started")
	}
	// -----------------启动服务----------------------
	// 1. 设置status为true 表示服务器已在运行
	// 2. 初始化stop channal,这用于通知registry stop  keep alive
	// 3. 初始化tcp socket并开始监听
	// 4. 注册rpc服务至grpc 这样grpc收到request可以分发给server处理
	// 5. 将自己的服务名/Host地址注册至etcd 这样client可以通过etcd
	//    获取服务Host地址 从而进行通信。这样的好处是client只需知道服务名
	//    以及etcd的Host即可获取对应服务IP 无需写死至client代码中
	// ----------------------------------------------
	s.Status = true
	s.StopSignal = make(chan error)

	port := strings.Split(s.Addr, ":")[1]
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterGroupCacheServer(grpcServer, s)

	//注册至etcd
	go func() {
		err := registry.Register("GroupCache", s.Addr, s.StopSignal)
		//在这里，被Register阻塞了
		if err != nil {
			log.Fatal(err)
		}
		// close channel
		close(s.StopSignal)
		// close tcp listen
		err = lis.Close()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("[%s] revoke service and close tcp socket ok.", s.Addr)
	}()

	s.mu.Unlock()

	//在lis上监听客户端发来的连接请求
	//grpc请求到达时，Serve会将这些请求分发给相应的处理程序，并执行对应的rpc方法
	//阻塞方法，这意味着它会持续运行，直到服务器停止或发生致命错误。
	if err := grpcServer.Serve(lis); err != nil && s.Status {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}

// Stop 停止server运行 如果server没有运行 这将是一个no-op（表示没有效果）
func (s *Server) Stop() {
	s.mu.Lock()
	if s.Status == false {
		s.mu.Unlock()
		return
	}
	s.StopSignal <- nil // 发送停止keepalive信号
	s.Status = false    // 设置server运行状态为stop
	s.clients = nil     // 清空一致性哈希信息 有助于垃圾回收
	s.Hash = nil
	s.mu.Unlock()
}

var _ Picker = (*Server)(nil)
