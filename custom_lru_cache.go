package main

/*
基于自定义链表实现的LRU（Least Recently Used）缓存

使用自定义双向链表而非标准库实现，直接实现所有链表操作，不依赖任何数据结构库。
采用哈希表+自定义双向链表的组合结构：
- 哈希表：快速查找键是否存在
- 自定义双向链表：维护数据的访问顺序

LRU缓存淘汰策略：当缓存满时，淘汰最长时间未被访问的数据项
*/

import (
	"fmt"
)

// CustomLRUNode 自定义LRU缓存节点结构
type CustomLRUNode struct {
	Key   string      // 键
	Value interface{} // 值
}

// CustomLRUCache 自定义LRU缓存结构
type CustomLRUCache struct {
	capacity int                  // 最大容量
	cache    map[string]*ListNode // 哈希表: 键 -> 链表节点
	list     *List                // 自定义双向链表: 维护访问顺序
}

// NewCustomLRUCache 创建指定容量的自定义LRU缓存
func NewCustomLRUCache(capacity int) *CustomLRUCache {
	return &CustomLRUCache{
		capacity: capacity,
		cache:    make(map[string]*ListNode),
		list:     NewList(),
	}
}

// Get 获取缓存中的值，不存在返回nil和false
func (c *CustomLRUCache) Get(key string) (interface{}, bool) {
	// 查找哈希表
	if node, exists := c.cache[key]; exists {
		// 找到节点，将其移动到链表头部（表示最近使用）
		c.list.MoveToFront(node)
		// 返回节点值
		return node.Value.(*CustomLRUNode).Value, true
	}
	// 未找到
	return nil, false
}

// Put 插入或更新缓存中的键值对
func (c *CustomLRUCache) Put(key string, value interface{}) {
	// 如果键已存在，更新值并移动到链表头部
	if node, exists := c.cache[key]; exists {
		// 更新值
		node.Value.(*CustomLRUNode).Value = value
		// 移动到链表头部
		c.list.MoveToFront(node)
		return
	}

	// 如果达到容量上限，删除最近最少使用的元素（链表尾部）
	if c.list.Len() >= c.capacity {
		// 获取链表尾部节点
		leastUsed := c.list.Back()
		if leastUsed != nil {
			// 从哈希表中删除
			delete(c.cache, leastUsed.Value.(*CustomLRUNode).Key)
			// 从链表中删除
			c.list.Remove(leastUsed)
		}
	}

	// 创建新节点
	lruNode := &CustomLRUNode{Key: key, Value: value}
	// 插入链表头部
	newNode := c.list.PushFront(lruNode)
	// 在哈希表中记录节点位置
	c.cache[key] = newNode
}

// 场景示例：文件系统缓存
func CustomLRUCacheDemo() {
	// 创建容量为4的LRU缓存
	cache := NewCustomLRUCache(4)

	fmt.Println("文件系统缓存场景 (自定义LRU缓存容量=4):")

	// 用户访问多个文件
	cache.Put("file1.txt", "这是文件1的内容")
	cache.Put("file2.txt", "这是文件2的内容")
	cache.Put("file3.txt", "这是文件3的内容")
	cache.Put("file4.txt", "这是文件4的内容")

	// 查看当前缓存状态
	printCustomLRUStatus(cache, "初始访问四个文件后")

	// 用户再次访问file1，将其提升为最近使用
	if content, found := cache.Get("file1.txt"); found {
		fmt.Printf("访问文件: file1.txt, 内容: %v\n", content)
	}

	printCustomLRUStatus(cache, "访问file1.txt后")

	// 用户访问新文件file5，此时最久未使用的file2应被淘汰
	cache.Put("file5.txt", "这是文件5的内容")

	printCustomLRUStatus(cache, "访问新文件file5.txt后")

	// 用户尝试访问已被淘汰的file2
	if content, found := cache.Get("file2.txt"); found {
		fmt.Printf("访问文件: file2.txt, 内容: %v\n", content)
	} else {
		fmt.Println("文件不在缓存中: file2.txt (已被淘汰)")
	}

	// 测试更新操作
	cache.Put("file3.txt", "这是文件3的更新内容")
	printCustomLRUStatus(cache, "更新file3.txt后")
}

// 辅助函数：打印自定义LRU缓存状态
func printCustomLRUStatus(cache *CustomLRUCache, title string) {
	fmt.Printf("\n=== %s ===\n", title)
	// 从最近到最久遍历所有缓存项
	for node := cache.list.Front(); node != nil; node = node.Next() {
		item := node.Value.(*CustomLRUNode)
		fmt.Printf("键: %s, 值: %v\n", item.Key, item.Value)
	}
}
