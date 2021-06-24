package cache

import (
	"cache/consistenthash"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/cache/"
	defaultReplicas = 32
)

type HTTPPool struct {
	self        string
	basePath    string
	mu          sync.Mutex
	peers       *consistenthash.Map    // 根据具体的 key 选择节点
	httpGetters map[string]*HTTPGetter // 映射远程节点与对应的httpGetter(httpGetter与远程节点的地址baseURL有关)
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log info with server name
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP HTTP服务器的实现
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 如果不是访问对应的basePath，返回错误
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	// 记录日志
	p.Log("%s %s", r.Method, r.URL.Path)

	// 分割处理URL
	// /<basePath>/<groupName>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// HTTPGetter HTTP客户端
type HTTPGetter struct {
	baseURL string // e.g: http://example.com/cache/
}

// Get HTTP客户端的GET实现
func (h *HTTPGetter) Get(group string, key string) ([]byte, error) {
	// 拼接url
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)

	response, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", response.Status)
	}

	// 将返回值转化为[]byte类型
	bytes, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

// Set 实例化一致性哈希算法
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.peers = consistenthash.New(defaultReplicas, nil)

	// 添加传入的节点
	p.peers.Add(peers...)

	p.httpGetters = make(map[string]*HTTPGetter, len(peers))

	for _, peer := range peers {
		p.httpGetters[peer] = &HTTPGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer 根据具体的 key，选择节点，返回节点对应的 HTTP 客户端
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)

		return p.httpGetters[peer], true
	}

	return nil, false
}
