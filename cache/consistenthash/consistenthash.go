package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash hash函数，将值转化位uint32
type Hash func(data []byte) uint32

type Map struct {
	hash     Hash           // 哈希算法
	replicas int            // 虚拟节点倍数
	keys     []int          // 哈希环
	hashMap  map[int]string // 虚拟节点映射，键是虚拟节点的哈希值，值是真实节点的名称
}

// New 构造函数，允许自定义虚拟节点倍数和哈希函数
func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}

	// 如果没有注入哈希函数，那么默认采用ChecksumIEEE算法
	if fn == nil {
		m.hash = crc32.ChecksumIEEE
	}

	return m
}

// Add 添加真实机器节点，可以传入多个真实节点的名称
func (m Map) Add(keys ...string) {
	// 对每一个真实节点 key，对应创建 m.replicas 个虚拟节点
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			// 虚拟节点的名称为strconv.Itoa(i) + key
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 将计算虚拟节点的哈希值，使用 append(m.keys, hash) 添加到环上
			m.keys = append(m.keys, hash)
			// 增加虚拟节点和真实节点的映射关系
			m.hashMap[hash] = key
		}
	}
	// 环上的哈希值排序
	sort.Ints(m.keys)
}

// Get 选择节点
func (m Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	// 计算 key 的哈希值
	hash := int(m.hash([]byte(key)))

	// 顺时针找到第一个匹配的虚拟节点的下标index
	index := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// 从m.keys中获取到对应的哈希值，因为 m.keys 是一个环状结构，所以用取余数的方式来处理这种情况。
	return m.hashMap[m.keys[index%len(m.keys)]]
}
