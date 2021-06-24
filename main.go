package main

import (
	"cache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *cache.Group {
	return cache.NewGroup("scores", 2<<10, cache.GetterFunc(func(key string) ([]byte, error) {
		log.Println("[SlowDB] search key", key)
		if v, ok := db[key]; ok {
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s not exist", key)
	}))
}

// startCacheServer 创建 HTTPPool，添加节点信息，注册到 cache 中，启动 HTTP 服务
func startCacheServer(addr string, addrs []string, c *cache.Group) {
	peers := cache.NewHTTPPool(addr)
	peers.Set(addrs...)
	c.RegisterPeers(peers)
	log.Println("cache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// startAPIServer 启动一个 API 服务
func startAPIServer(apiAddr string, c *cache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := c.Get(key)
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

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "cache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string

	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	c := createGroup()

	if api {
		go startAPIServer(apiAddr, c)
	}

	startCacheServer(addrMap[port], addrs, c)
}
