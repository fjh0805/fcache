package cache

import (
	cachepb "github.com/limerence-yu/fcache/cachepb"
)

// 抽象节点
// 对应的HTTPPool 服务端
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// 通过peer 根据group和key 找到缓存值
// 对应的httpGetter 客户端
type PeerGetter interface {
	//Get(group string, key string) ([]byte, error)
	Get(in *cachepb.Request, out *cachepb.Response) error
}
