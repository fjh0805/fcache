package cache

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"

	"github.com/limerence-yu/fcache/consistenthash"
	cachepb "github.com/limerence-yu/fcache/cachepb"
)

const defaultBasePath = "/_fcache/"
const defaultReplicas = 50

type HTTPPool struct {
	self        string //peer http://localhost:9000
	basePath    string // /_fcache/
	mu          sync.Mutex
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter
}

type httpGetter struct {
	baseURL string //peer + p.basePath
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{peer + p.basePath}
	}
}

func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		return p.httpGetters[peer], true
	}
	return nil, false
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//log.Printf("Full URL: %s", r)
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("basePath error")
	}
	//p.Log("r.URL.Path: %s, r.Host: %s", r.URL.Path, r.Host)
	p.Log("%s %s", r.Method, r.URL.Path)
	part := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	groupName := part[0]
	key := part[1]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group"+groupName, http.StatusNotFound)
		return
	}
	v, err := group.Get(key)
	body, err := proto.Marshal(&cachepb.Response{Value: v.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//application/octet-stream 是一种通用的二进制数据类型，用于传输任意类型的二进制数据，没有特定的结构或者格式，
	//可以用于传输图片、音频、视频、压缩文件等任意二进制数据。
	// HTTP库掌握的不行
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (h *httpGetter) Get(in *cachepb.Request, out *cachepb.Response) error {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := io.ReadAll(res.Body)
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}
