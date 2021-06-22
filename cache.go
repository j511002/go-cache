package cache

import (
	"cache/algorithm/lru"
	"sync"
)

type cache struct {
	lru       *lru.Cache
	mu        sync.Mutex
	cacheByte int64
}

func (c *cache) add(key string, value ByteView) {

}

func (c *cache) get(key string) (value ByteView, ok bool) {

}
