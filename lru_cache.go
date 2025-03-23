package main

/*
LRU（Least Recently Used）缓存替换算法

原理：
LRU算法基于"最近最少使用"原则淘汰数据，核心思想是"如果数据最近被访问过，那么将来被访问的几率也更高"。

关键特点：
1. 当缓存满时，优先淘汰最长时间未被访问的数据
2. 每次数据被访问时，需要将其移动到"最近使用"的位置
3. 新数据插入时放在"最近使用"的位置

实现方式：
- 采用哈希表+双向链表的组合结构
- 哈希表提供O(1)时间复杂度的查找能力
- 双向链表维护数据的访问顺序，支持O(1)删除和添加

应用场景：
- Web页面缓存
- 数据库缓存
- 操作系统页面置换算法
- 浏览器最近浏览历史

优缺点：
- 优点：实现简单，命中率较高
- 缺点：无法识别热点数据，只关注访问时间，不关注访问频率

以下实现了一个基本的LRU缓存，支持Get和Put操作，容量有限。
*/

import (
	"container/list"
	"fmt"
)

// LRUNode 双向链表节点结构
type LRUNode struct {
	Key   string
	Value interface{}
}

// LRUCache LRU缓存结构
type LRUCache struct {
	capacity int                      // 最大容量
	cache    map[string]*list.Element // 哈希表: 键 -> 链表节点指针
	list     *list.List               // 双向链表: 维护访问顺序
}

// NewLRUCache 创建指定容量的LRU缓存
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		list:     list.New(),
	}
}

// Get 获取缓存中的值，不存在返回nil和false
func (c *LRUCache) Get(key string) (interface{}, bool) {
	// 查找哈希表
	if element, exists := c.cache[key]; exists {
		// 找到节点，将其移动到链表头部（表示最近使用）
		c.list.MoveToFront(element)
		// 返回节点值
		return element.Value.(*LRUNode).Value, true
	}
	// 未找到
	return nil, false
}

// Put 插入或更新缓存中的键值对
func (c *LRUCache) Put(key string, value interface{}) {
	// 如果键已存在，更新值并移动到链表头部
	if element, exists := c.cache[key]; exists {
		// 更新值
		element.Value.(*LRUNode).Value = value
		// 移动到链表头部
		c.list.MoveToFront(element)
		return
	}

	// 如果达到容量上限，删除最近最少使用的元素（链表尾部）
	if c.list.Len() >= c.capacity {
		// 获取链表尾部节点
		leastUsed := c.list.Back()
		if leastUsed != nil {
			// 从链表中删除
			c.list.Remove(leastUsed)
			// 从哈希表中删除
			delete(c.cache, leastUsed.Value.(*LRUNode).Key)
		}
	}

	// 创建新节点
	node := &LRUNode{Key: key, Value: value}
	// 插入链表头部
	element := c.list.PushFront(node)
	// 在哈希表中记录节点位置
	c.cache[key] = element
}

// 场景示例：网页浏览历史缓存
func LRUCacheDemo() {
	// 创建容量为3的LRU缓存
	cache := NewLRUCache(3)

	// 模拟用户访问网页
	fmt.Println("用户浏览网站场景 (LRU缓存容量=3):")

	// 用户访问三个不同网页
	cache.Put("https://example.com/page1", "首页")
	cache.Put("https://example.com/page2", "产品页")
	cache.Put("https://example.com/page3", "关于我们")

	// 查看当前缓存状态
	printCacheStatus(cache, "初始访问三个页面后")

	// 用户重新访问page1，将page1提升为最近使用
	if page, found := cache.Get("https://example.com/page1"); found {
		fmt.Printf("重新访问页面: %v\n", page)
	}

	printCacheStatus(cache, "访问page1后")

	// 用户访问新页面page4，此时最久未使用的page2应被淘汰
	cache.Put("https://example.com/page4", "联系我们")

	printCacheStatus(cache, "访问新页面page4后")

	// 用户尝试访问已被淘汰的page2
	if page, found := cache.Get("https://example.com/page2"); found {
		fmt.Printf("访问页面: %v\n", page)
	} else {
		fmt.Println("页面不在缓存中: https://example.com/page2 (已被淘汰)")
	}
}

// 辅助函数：打印缓存状态
func printCacheStatus(cache *LRUCache, title string) {
	fmt.Printf("\n=== %s ===\n", title)
	// 从最近到最久遍历所有缓存项
	for element := cache.list.Front(); element != nil; element = element.Next() {
		node := element.Value.(*LRUNode)
		fmt.Printf("键: %s, 值: %v\n", node.Key, node.Value)
	}
}
