package cache

//抽象节点
//对应的HTTPPool 服务端
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

//通过peer 根据group和key 找到缓存值
//对应的httpGetter 客户端
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}