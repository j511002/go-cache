package cache

import (
	"fmt"
	"log"
	"sync"
)

// Getter 用key获取缓存.
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc Getter的接口型函数
type GetterFunc func(key string) ([]byte, error)

// Get Getter中Get方法的实现
func (receiver GetterFunc) Get(key string) ([]byte, error) {
	return receiver(key)
}

type Group struct {
	name      string // group的name
	getter    Getter
	mainCache cache
	peers     PeerPicker
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// RegisterPeers 将实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegisterPeers(picker PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}

	g.peers = picker
}

// getFromPeer 使用实现了 PeerGetter 接口的 HttpGetter 从访问远程节点，获取缓存值
func (g *Group) getFromPeer(getter PeerGetter, key string) (ByteView, error) {
	bytes, err := getter.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}

	return ByteView{b: bytes}, nil
}

func NewGroup(name string, cacheByte int64, getter Getter) *Group {
	if getter == nil {
		panic("nil getter")
	}

	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheByte: cacheByte},
	}

	groups[name] = g

	return g
}

// GetGroup 从groups中取出对应name的group
func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()

	g := groups[name]

	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	// 从 mainCache 中查找缓存，如果存在则返回缓存值
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[Cache] hit")
		return v, nil
	}

	// 如果没有，	则调用 load 方法，从外部源获取数据
	return g.load(key)
}

func (g *Group) load(key string) (ByteView, error) {
	if g.peers != nil {
		// 使用PickPeer方法选择节点，如果是外部节点，就连接外部节点获取数据
		if peer, ok := g.peers.PickPeer(key); ok {
			if value, err := g.getFromPeer(peer, key); err == nil {
				return value, nil
			} else {
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
	}
	// 如果是内部节点，那么就从本地获取数据
	return g.getLocally(key)
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	value := ByteView{b: cloneByte(bytes)}

	g.populateCache(key, value)

	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
