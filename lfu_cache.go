package main

/*
LFU（Least Frequently Used）缓存替换算法

原理：
LFU算法基于"访问频率"淘汰数据，核心思想是"如果数据过去被访问次数少，那么将来被访问的几率也更低"。

关键特点：
1. 当缓存满时，优先淘汰访问次数最少的数据
2. 当多个数据访问次数相同时，淘汰最早进入缓存的数据(最少最近使用)
3. 每次访问数据时，需要增加其访问计数
4. 新插入数据的访问计数从1开始

实现方式：
- 需要同时维护数据访问频率和访问时间顺序
- 通常使用哈希表+多个双向链表的组合结构
- 每个频率对应一个双向链表，存储相同频率的节点
- 另一个哈希表记录每个节点的频率

应用场景：
- 内容分发网络(CDN)缓存
- 数据库查询缓存
- 需要识别热点数据的应用场景
- 文件系统缓存

优缺点：
- 优点：能够更好地识别热点数据，提高命中率
- 缺点：实现复杂，需要额外维护频率计数，可能存在"缓存污染"问题（长时间未使用但历史频率高的数据难以被淘汰）

以下实现了一个基本的LFU缓存，支持Get和Put操作，容量有限。
*/

import (
	"container/list"
	"fmt"
)

// LFUNode LFU缓存节点结构
type LFUNode struct {
	Key   string
	Value interface{}
	Freq  int // 访问频率
}

// LFUCache LFU缓存结构
type LFUCache struct {
	capacity int                      // 最大容量
	cache    map[string]*list.Element // 键 -> 链表节点
	freqMap  map[int]*list.List       // 频率 -> 包含该频率节点的链表
	minFreq  int                      // 当前最小频率
}

// NewLFUCache 创建指定容量的LFU缓存
func NewLFUCache(capacity int) *LFUCache {
	return &LFUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		freqMap:  make(map[int]*list.List),
		minFreq:  0,
	}
}

// 增加节点频率并更新位置
func (c *LFUCache) incrementFreq(element *list.Element) {
	node := element.Value.(*LFUNode)

	// 从当前频率链表中删除
	c.freqMap[node.Freq].Remove(element)

	// 如果当前频率链表为空，且是最小频率，更新最小频率
	if c.freqMap[node.Freq].Len() == 0 && c.minFreq == node.Freq {
		c.minFreq++
	}

	// 增加节点频率
	node.Freq++

	// 确保新频率的链表存在
	if _, ok := c.freqMap[node.Freq]; !ok {
		c.freqMap[node.Freq] = list.New()
	}

	// 添加到新频率链表的头部
	newElement := c.freqMap[node.Freq].PushFront(node)

	// 更新缓存映射
	c.cache[node.Key] = newElement
}

// Get 获取键对应的值，不存在返回nil和false
func (c *LFUCache) Get(key string) (interface{}, bool) {
	element, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	// 获取节点
	node := element.Value.(*LFUNode)

	// 增加访问频率
	c.incrementFreq(element)

	return node.Value, true
}

// Put 插入或更新键值对
func (c *LFUCache) Put(key string, value interface{}) {
	// 如果容量为0，不做任何操作
	if c.capacity == 0 {
		return
	}

	// 如果键已存在，更新值并增加频率
	if element, exists := c.cache[key]; exists {
		node := element.Value.(*LFUNode)
		node.Value = value
		c.incrementFreq(element)
		return
	}

	// 如果达到容量上限，删除访问频率最低的元素
	if len(c.cache) >= c.capacity {
		// 获取最小频率链表
		minFreqList := c.freqMap[c.minFreq]
		// 删除链表尾部元素（最早加入的）
		leastFreqNode := minFreqList.Back()
		if leastFreqNode != nil {
			// 从链表中删除
			minFreqList.Remove(leastFreqNode)
			// 从缓存中删除
			delete(c.cache, leastFreqNode.Value.(*LFUNode).Key)
		}
	}

	// 对新元素，频率从1开始
	c.minFreq = 1

	// 确保频率为1的链表存在
	if _, ok := c.freqMap[1]; !ok {
		c.freqMap[1] = list.New()
	}

	// 创建新节点
	node := &LFUNode{
		Key:   key,
		Value: value,
		Freq:  1,
	}

	// 添加到频率为1的链表头部
	element := c.freqMap[1].PushFront(node)

	// 更新缓存映射
	c.cache[key] = element
}

// 场景示例：在线商城商品缓存
func LFUCacheDemo() {
	// 创建容量为3的LFU缓存，用于存储热门商品信息
	cache := NewLFUCache(3)

	fmt.Println("电商平台热门商品缓存场景 (LFU缓存容量=3):")

	// 模拟商品浏览
	// 用户浏览三种商品
	cache.Put("product:1001", "iPhone 手机")
	cache.Put("product:1002", "MacBook 笔记本")
	cache.Put("product:1003", "iPad 平板")

	// 打印初始缓存状态
	fmt.Println("\n=== 初始缓存状态 ===")
	printLFUStatus(cache)

	// 用户多次查看iPhone（增加访问频率）
	cache.Get("product:1001") // 第2次
	cache.Get("product:1001") // 第3次
	cache.Get("product:1001") // 第4次

	// 用户查看一次MacBook
	cache.Get("product:1002") // 第2次

	fmt.Println("\n=== 多次访问后的缓存状态 ===")
	printLFUStatus(cache)

	// 添加新商品AirPods，此时应淘汰访问频率最低的iPad
	cache.Put("product:1004", "AirPods 耳机")

	fmt.Println("\n=== 添加新商品后的缓存状态 ===")
	printLFUStatus(cache)

	// 用户尝试查看已被淘汰的iPad
	if product, found := cache.Get("product:1003"); found {
		fmt.Printf("查看商品: %v\n", product)
	} else {
		fmt.Println("商品不在缓存中: product:1003 (已被淘汰)")
	}

	// 展示访问频率决定淘汰的特性
	// 再添加一个新商品，此时应淘汰AirPods（频率为1）而非MacBook（频率为2）
	cache.Put("product:1005", "Apple Watch 手表")

	fmt.Println("\n=== 再次添加新商品后的缓存状态 ===")
	printLFUStatus(cache)
}

// 辅助函数：打印LFU缓存状态
func printLFUStatus(cache *LFUCache) {
	// 按频率分组打印
	for freq := 1; freq <= 10; freq++ {
		if list, exists := cache.freqMap[freq]; exists && list.Len() > 0 {
			fmt.Printf("频率 %d:\n", freq)
			for e := list.Front(); e != nil; e = e.Next() {
				node := e.Value.(*LFUNode)
				fmt.Printf("  键: %s, 值: %v\n", node.Key, node.Value)
			}
		}
	}
	fmt.Printf("当前最小频率: %d\n", cache.minFreq)
}
