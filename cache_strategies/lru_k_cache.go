package cache_strategies

/*
LRU-K（Least Recently Used K）缓存替换算法

原理：
LRU-K是LRU算法的一个变种，它考虑数据的前K次访问，而不仅仅是最近一次访问。
通过记录每个数据的前K次访问历史，计算数据的"K距离"(第K次最近访问的时间)来决定淘汰顺序。

关键特点：
1. 维护每个数据的K次最近访问历史
2. 对于访问次数少于K次的数据，使用特殊处理（通常在初始阶段更容易被淘汰）
3. 当缓存满时，淘汰K距离最大的数据
4. K值越大，算法对访问频率的敏感度越高，越有利于识别热点数据

实现方式：
- 使用哈希表存储数据及其访问历史
- 使用两个队列：一个用于存储访问次数小于K次的数据，另一个用于存储访问次数达到K次的数据
- 对于两个队列的淘汰策略略有不同

应用场景：
- 数据库缓存
- 需要识别长期热点数据的场景
- 访问模式具有明显局部性的场景

优缺点：
- 优点：更好地识别长期热点数据，对突发访问不敏感
- 缺点：实现复杂，需要维护更多的历史信息，内存开销较大

以下实现一个LRU-2缓存（K=2），记录每个数据的前两次访问时间。
*/

import (
	"container/list"
	"fmt"
	"time"
)

// LRUK参数常量
const (
	DefaultK             = 2              // 默认K值
	InfiniteDistance     = int64(1 << 60) // 无限大的距离值（用于未满K次访问的数据）
	CorrelationThreshold = 100            // 历史关联阈值（毫秒）
)

// LRUKNode LRU-K缓存节点结构
type LRUKNode struct {
	Key          string      // 键
	Value        interface{} // 值
	HistoryTimes []int64     // 历史访问时间戳（最多K个）
	AccessCount  int         // 历史访问次数
}

// LRUKCache LRU-K缓存结构
type LRUKCache struct {
	capacity int                      // 最大容量
	k        int                      // K值
	cache    map[string]*list.Element // 哈希表: 键 -> 链表节点
	history  *list.List               // 历史队列: 访问次数 < K 的节点
	cache2q  *list.List               // 缓存队列: 访问次数 >= K 的节点
	clock    func() int64             // 时钟函数，用于模拟或获取时间
}

// NewLRUKCache 创建指定容量和K值的LRU-K缓存
func NewLRUKCache(capacity int, k int) *LRUKCache {
	if k <= 0 {
		k = DefaultK
	}
	return &LRUKCache{
		capacity: capacity,
		k:        k,
		cache:    make(map[string]*list.Element),
		history:  list.New(),
		cache2q:  list.New(),
		clock:    func() int64 { return time.Now().UnixNano() / int64(time.Millisecond) },
	}
}

// 获取节点的K距离（第K次最近访问的时间）
func (c *LRUKCache) kDistance(node *LRUKNode) int64 {
	if node.AccessCount < c.k {
		return InfiniteDistance // 未满K次访问，返回无限大的距离
	}
	// K距离等于当前时间减去第K次最近的访问时间
	// 在这里，历史记录是按时间从新到旧排序的，所以取倒数第一个元素
	return c.clock() - node.HistoryTimes[c.k-1]
}

// Get 获取缓存中的值，不存在返回nil和false
func (c *LRUKCache) Get(key string) (interface{}, bool) {
	if element, exists := c.cache[key]; exists {
		node := element.Value.(*LRUKNode)
		c.recordAccess(node, element)
		return node.Value, true
	}
	return nil, false
}

// recordAccess 记录节点的访问
func (c *LRUKCache) recordAccess(node *LRUKNode, element *list.Element) {
	// 记录新的访问时间
	now := c.clock()

	// 更新访问历史
	if node.AccessCount < c.k {
		// 未满K次，添加新的访问记录
		node.HistoryTimes = append([]int64{now}, node.HistoryTimes...)
		node.AccessCount++
	} else {
		// 已满K次，移除最旧的，添加新的
		node.HistoryTimes = append([]int64{now}, node.HistoryTimes[:c.k-1]...)
	}

	// 如果节点在历史队列中且已达到K次访问，将其移至缓存队列
	if element.Value == node && node.AccessCount == c.k && element.Value == node {
		c.history.Remove(element)
		newElement := c.cache2q.PushFront(node)
		c.cache[node.Key] = newElement
	} else if element.Value == node && node.AccessCount >= c.k {
		// 已经在缓存队列中，移至队列前端
		c.cache2q.MoveToFront(element)
	}
}

// Put 插入或更新缓存中的键值对
func (c *LRUKCache) Put(key string, value interface{}) {
	// 如果键已存在，更新值并记录访问
	if element, exists := c.cache[key]; exists {
		node := element.Value.(*LRUKNode)
		node.Value = value
		c.recordAccess(node, element)
		return
	}

	// 如果达到容量上限，需要淘汰节点
	if len(c.cache) >= c.capacity {
		c.evict()
	}

	// 创建新节点
	node := &LRUKNode{
		Key:          key,
		Value:        value,
		HistoryTimes: []int64{c.clock()},
		AccessCount:  1,
	}

	// 新节点放入历史队列
	element := c.history.PushFront(node)
	c.cache[key] = element
}

// 淘汰策略
func (c *LRUKCache) evict() {
	// 优先从历史队列中淘汰
	if c.history.Len() > 0 {
		oldest := c.history.Back()
		c.history.Remove(oldest)
		delete(c.cache, oldest.Value.(*LRUKNode).Key)
		return
	}

	// 如果历史队列为空，从缓存队列淘汰K距离最大的
	if c.cache2q.Len() > 0 {
		var toRemove *list.Element
		maxDistance := int64(-1)

		// 遍历查找K距离最大的节点
		for e := c.cache2q.Back(); e != nil; e = e.Prev() {
			node := e.Value.(*LRUKNode)
			distance := c.kDistance(node)
			if distance > maxDistance {
				maxDistance = distance
				toRemove = e
			}
		}

		if toRemove != nil {
			c.cache2q.Remove(toRemove)
			delete(c.cache, toRemove.Value.(*LRUKNode).Key)
		}
	}
}

// Remove 从缓存中删除指定键
func (c *LRUKCache) Remove(key string) bool {
	if element, exists := c.cache[key]; exists {
		node := element.Value.(*LRUKNode)
		if node.AccessCount < c.k {
			c.history.Remove(element)
		} else {
			c.cache2q.Remove(element)
		}
		delete(c.cache, key)
		return true
	}
	return false
}

// Size 返回当前缓存中的元素数量
func (c *LRUKCache) Size() int {
	return len(c.cache)
}

// 场景示例：数据库查询缓存
func LRUKCacheDemo() {
	// 创建容量为4的LRU-2缓存
	cache := NewLRUKCache(4, 2)

	// 自定义时钟，方便演示
	currentTime := int64(0)
	cache.clock = func() int64 {
		currentTime += 100 // 每次访问时间增加100ms
		return currentTime
	}

	fmt.Println("数据库查询缓存示例 (LRU-2缓存容量=4):")

	// 模拟数据库查询缓存
	cache.Put("SELECT * FROM users", "用户表查询结果")    // 访问1次
	cache.Put("SELECT * FROM products", "产品表查询结果") // 访问1次
	cache.Put("SELECT * FROM orders", "订单表查询结果")   // 访问1次

	// 再次查询users和products，使其访问次数达到2次
	cache.Get("SELECT * FROM users")    // 访问2次
	cache.Get("SELECT * FROM products") // 访问2次

	// 查看当前缓存状态
	fmt.Println("\n=== 初始缓存状态(部分查询已访问2次) ===")
	printLRUKStatus(cache)

	// 新增一个查询，不会淘汰已达到2次访问的查询
	cache.Put("SELECT * FROM categories", "分类表查询结果") // 访问1次

	fmt.Println("\n=== 添加新查询后 ===")
	printLRUKStatus(cache)

	// 再新增一个查询，此时应淘汰访问次数为1次的旧查询(orders)
	cache.Put("SELECT * FROM inventory", "库存表查询结果") // 访问1次

	fmt.Println("\n=== 再次添加新查询后 ===")
	printLRUKStatus(cache)

	// 使新查询也达到2次访问
	cache.Get("SELECT * FROM categories") // 访问2次
	cache.Get("SELECT * FROM inventory")  // 访问2次

	fmt.Println("\n=== 所有查询都达到2次访问后 ===")
	printLRUKStatus(cache)

	// 再新增查询，此时应根据K距离淘汰
	cache.Put("SELECT * FROM suppliers", "供应商表查询结果")

	fmt.Println("\n=== 添加新查询后(根据K距离淘汰) ===")
	printLRUKStatus(cache)
}

// 辅助函数：打印LRU-K缓存状态
func printLRUKStatus(cache *LRUKCache) {
	fmt.Println("历史队列(访问次数<K):")
	for e := cache.history.Front(); e != nil; e = e.Next() {
		node := e.Value.(*LRUKNode)
		fmt.Printf("  键: %s, 值: %v, 访问次数: %d, 访问历史: %v\n",
			node.Key, node.Value, node.AccessCount, node.HistoryTimes)
	}

	fmt.Println("缓存队列(访问次数>=K):")
	for e := cache.cache2q.Front(); e != nil; e = e.Next() {
		node := e.Value.(*LRUKNode)
		fmt.Printf("  键: %s, 值: %v, 访问次数: %d, 访问历史: %v, K距离: %d\n",
			node.Key, node.Value, node.AccessCount, node.HistoryTimes,
			cache.kDistance(node))
	}
}
