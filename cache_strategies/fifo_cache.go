package cache_strategies

/*
FIFO（First In First Out）缓存替换算法

原理：
FIFO是最简单的缓存替换算法，基于"先进先出"原则淘汰数据。
最先进入缓存的数据在缓存满时会被优先淘汰，不考虑数据的访问频率和时间。

关键特点：
1. 维护一个队列，新数据从队尾加入，淘汰时从队头移除
2. 淘汰策略仅基于数据入队顺序，与访问模式无关
3. 实现简单，开销小

实现方式：
- 使用队列 + 哈希表的组合结构
- 队列维护数据的先后顺序
- 哈希表提供O(1)的快速查找

应用场景：
- 简单的Web页面缓存
- 资源受限环境下的缓存实现
- 访问模式接近随机的场景

优缺点：
- 优点：实现简单，内存开销小
- 缺点：不考虑数据热度，可能淘汰常用数据，命中率较低

以下实现了一个基本的FIFO缓存，支持Get、Put和Remove操作。
*/

import (
	"container/list"
	"fmt"
)

// FIFONode FIFO缓存节点结构
type FIFONode struct {
	Key   string
	Value interface{}
}

// FIFOCache FIFO缓存结构
type FIFOCache struct {
	capacity int                      // 最大容量
	queue    *list.List               // 队列：维护先进先出顺序
	cache    map[string]*list.Element // 哈希表：键 -> 队列节点
}

// NewFIFOCache 创建指定容量的FIFO缓存
func NewFIFOCache(capacity int) *FIFOCache {
	return &FIFOCache{
		capacity: capacity,
		queue:    list.New(),
		cache:    make(map[string]*list.Element),
	}
}

// Get 获取缓存中的值，不存在返回nil和false
func (c *FIFOCache) Get(key string) (interface{}, bool) {
	// 查找哈希表
	if element, exists := c.cache[key]; exists {
		// 返回节点值，但不改变位置（与LRU不同）
		return element.Value.(*FIFONode).Value, true
	}
	// 未找到
	return nil, false
}

// Put 插入或更新缓存中的键值对
func (c *FIFOCache) Put(key string, value interface{}) {
	// 如果键已存在，只更新值，不改变位置（与LRU不同）
	if element, exists := c.cache[key]; exists {
		element.Value.(*FIFONode).Value = value
		return
	}

	// 如果达到容量上限，从队列头部删除最早的元素
	if c.queue.Len() >= c.capacity {
		oldest := c.queue.Front()
		if oldest != nil {
			c.queue.Remove(oldest)
			// 从哈希表中删除
			delete(c.cache, oldest.Value.(*FIFONode).Key)
		}
	}

	// 创建新节点并添加到队列尾部
	node := &FIFONode{Key: key, Value: value}
	element := c.queue.PushBack(node)

	// 在哈希表中记录节点位置
	c.cache[key] = element
}

// Remove 从缓存中删除指定键
func (c *FIFOCache) Remove(key string) bool {
	if element, exists := c.cache[key]; exists {
		c.queue.Remove(element)
		delete(c.cache, key)
		return true
	}
	return false
}

// Size 返回当前缓存中的元素数量
func (c *FIFOCache) Size() int {
	return c.queue.Len()
}

// Clear 清空缓存
func (c *FIFOCache) Clear() {
	c.queue = list.New()
	c.cache = make(map[string]*list.Element)
}

// Keys 返回缓存中所有键的列表（按FIFO顺序）
func (c *FIFOCache) Keys() []string {
	keys := make([]string, 0, c.queue.Len())
	for e := c.queue.Front(); e != nil; e = e.Next() {
		keys = append(keys, e.Value.(*FIFONode).Key)
	}
	return keys
}

// 场景示例：网络请求缓存
func FIFOCacheDemo() {
	// 创建容量为3的FIFO缓存
	cache := NewFIFOCache(3)

	fmt.Println("网络请求缓存示例 (FIFO缓存容量=3):")

	// 模拟API请求响应缓存
	cache.Put("api/users", "用户列表数据")
	cache.Put("api/products", "产品列表数据")
	cache.Put("api/orders", "订单列表数据")

	// 查看缓存状态
	fmt.Println("\n=== 初始缓存状态 ===")
	printFIFOStatus(cache)

	// 重复获取已存在的数据（不影响其在FIFO中的位置）
	if data, found := cache.Get("api/users"); found {
		fmt.Printf("获取数据: api/users = %v\n", data)
	}

	// 添加新数据，此时会淘汰最早进入的数据（api/users）
	cache.Put("api/settings", "系统设置数据")

	fmt.Println("\n=== 添加新数据后 ===")
	printFIFOStatus(cache)

	// 尝试获取已淘汰的数据
	if data, found := cache.Get("api/users"); found {
		fmt.Printf("获取数据: api/users = %v\n", data)
	} else {
		fmt.Println("数据不存在: api/users (已被淘汰)")
	}

	// 测试手动删除
	cache.Remove("api/products")

	fmt.Println("\n=== 删除数据后 ===")
	printFIFOStatus(cache)
}

// 辅助函数：打印FIFO缓存状态
func printFIFOStatus(cache *FIFOCache) {
	fmt.Println("FIFO队列顺序（从先到后）:")
	for i, key := range cache.Keys() {
		value, _ := cache.Get(key)
		fmt.Printf("%d. 键: %s, 值: %v\n", i+1, key, value)
	}
}
