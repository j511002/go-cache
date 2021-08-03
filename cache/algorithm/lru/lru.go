package lru

import (
	"container/list"
)

type Value interface {
	Len() int
}

// 双向链表节点的数据类型
type entry struct {
	key   string
	value Value
}

type Cache struct {
	// 允许使用的最大内存
	maxBytes int64
	// 当前已使用的内存
	nBytes int64
	// go官方定义的双向链表
	ll    *list.List
	cache map[string]*list.Element
	// 可选，被移除时调用的回调函数
	OnRemove func(key string, value Value)
}

// New 构造函数
func New(maxBytes int64, onRemove func(string, Value)) *Cache {
	return &Cache{
		maxBytes: maxBytes,
		ll:       list.New(),
		cache:    make(map[string]*list.Element),
		OnRemove: onRemove,
	}
}

// Get 从链表中获取值。
// 从字典中找到对应的双向链表的节点
// 将该节点移动到队尾
func (c *Cache) Get(key string) (value Value, ok bool) {
	if element, ok := c.cache[key]; ok {
		// 如果有值，那么就将元素放到队尾
		c.ll.MoveToFront(element)
		// 拿出对应的值
		kv := element.Value.(*entry)
		return kv.value, true
	}
	return
}

// Remove 从列表中移除值
func (c *Cache) Remove() {
	// 取到队首节点，从链表中删除
	element := c.ll.Back()
	if element != nil {
		c.ll.Remove(element)
		kv := element.Value.(*entry)
		// 从字典中 c.cache 删除该节点的映射关系
		delete(c.cache, kv.key)
		// 更新当前所用的内存
		c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		// 调用回调函数（如果不为nil
		if c.OnRemove != nil {
			c.OnRemove(kv.key, kv.value)
		}
	}
}

// Add 添加或者修改
func (c *Cache) Add(key string, value Value) {
	if element, ok := c.cache[key]; ok {
		// 移动到队列头部
		c.ll.MoveToFront(element)
		// 获取节点对象
		kv := element.Value.(*entry)
		// 修改内存占用
		c.nBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		element := c.ll.PushFront(&entry{key, value})
		c.cache[key] = element
		c.nBytes += int64(len(key)) + int64(value.Len())
	}

	// 清理掉超量的值（如果需要的话）
	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.Remove()
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
