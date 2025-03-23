package main

/*
基于自定义链表实现的LFU（Least Frequently Used）缓存

使用自定义双向链表而非标准库实现，直接实现所有链表操作，不依赖任何数据结构库。
采用哈希表+多个自定义双向链表的组合结构：
- 哈希表：快速查找键是否存在
- 频率映射：每个频率对应一个自定义双向链表
- 自定义双向链表：存储相同访问频率的节点，按访问时间排序

LFU缓存淘汰策略：当缓存满时，首先淘汰访问频率最低的数据；
若存在多个相同最低频率的数据，则淘汰其中最久未访问的数据。
*/

import (
	"fmt"
)

// CustomLFUNode 自定义LFU缓存节点结构
type CustomLFUNode struct {
	Key   string      // 键
	Value interface{} // 值
	Freq  int         // 访问频率
}

// CustomLFUCache 自定义LFU缓存结构
type CustomLFUCache struct {
	capacity int                  // 最大容量
	cache    map[string]*ListNode // 键 -> 链表节点
	freqMap  map[int]*List        // 频率 -> 对应频率的链表
	minFreq  int                  // 当前最小频率
}

// NewCustomLFUCache 创建指定容量的自定义LFU缓存
func NewCustomLFUCache(capacity int) *CustomLFUCache {
	return &CustomLFUCache{
		capacity: capacity,
		cache:    make(map[string]*ListNode),
		freqMap:  make(map[int]*List),
		minFreq:  0,
	}
}

// 增加节点频率并更新位置
func (c *CustomLFUCache) incrementFreq(node *ListNode) {
	lfuNode := node.Value.(*CustomLFUNode)

	// 从当前频率链表中删除
	c.freqMap[lfuNode.Freq].Remove(node)

	// 如果当前频率链表为空，且是最小频率，更新最小频率
	if c.freqMap[lfuNode.Freq].Len() == 0 && c.minFreq == lfuNode.Freq {
		c.minFreq++
	}

	// 增加节点频率
	lfuNode.Freq++

	// 确保新频率的链表存在
	if _, ok := c.freqMap[lfuNode.Freq]; !ok {
		c.freqMap[lfuNode.Freq] = NewList()
	}

	// 添加到新频率链表的头部
	newNode := c.freqMap[lfuNode.Freq].PushFront(lfuNode)

	// 更新缓存映射
	c.cache[lfuNode.Key] = newNode
}

// Get 获取键对应的值，不存在返回nil和false
func (c *CustomLFUCache) Get(key string) (interface{}, bool) {
	node, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	// 获取节点
	lfuNode := node.Value.(*CustomLFUNode)

	// 增加访问频率
	c.incrementFreq(node)

	return lfuNode.Value, true
}

// Put 插入或更新键值对
func (c *CustomLFUCache) Put(key string, value interface{}) {
	// 如果容量为0，不做任何操作
	if c.capacity == 0 {
		return
	}

	// 如果键已存在，更新值并增加频率
	if node, exists := c.cache[key]; exists {
		lfuNode := node.Value.(*CustomLFUNode)
		lfuNode.Value = value
		c.incrementFreq(node)
		return
	}

	// 如果达到容量上限，删除访问频率最低的元素
	if len(c.cache) >= c.capacity {
		// 获取最小频率链表
		minFreqList := c.freqMap[c.minFreq]
		// 删除链表尾部元素（最早加入的）
		leastFreqNode := minFreqList.Back()
		if leastFreqNode != nil {
			lfuNode := leastFreqNode.Value.(*CustomLFUNode)
			// 从链表中删除
			minFreqList.Remove(leastFreqNode)
			// 从缓存中删除
			delete(c.cache, lfuNode.Key)
		}
	}

	// 对新元素，频率从1开始
	c.minFreq = 1

	// 确保频率为1的链表存在
	if _, ok := c.freqMap[1]; !ok {
		c.freqMap[1] = NewList()
	}

	// 创建新节点
	lfuNode := &CustomLFUNode{
		Key:   key,
		Value: value,
		Freq:  1,
	}

	// 添加到频率为1的链表头部
	node := c.freqMap[1].PushFront(lfuNode)

	// 更新缓存映射
	c.cache[key] = node
}

// 场景示例：视频播放器缓存
func CustomLFUCacheDemo() {
	// 创建容量为4的LFU缓存，用于存储视频片段
	cache := NewCustomLFUCache(4)

	fmt.Println("视频播放器缓存场景 (自定义LFU缓存容量=4):")

	// 用户观看多个视频片段
	cache.Put("video:intro", "介绍片段数据")
	cache.Put("video:part1", "第一部分数据")
	cache.Put("video:part2", "第二部分数据")
	cache.Put("video:part3", "第三部分数据")

	// 打印初始缓存状态
	fmt.Println("\n=== 初始加载四个视频片段后 ===")
	printCustomLFUStatus(cache)

	// 用户反复观看intro片段
	cache.Get("video:intro") // 第2次
	cache.Get("video:intro") // 第3次

	// 用户看了两次part1
	cache.Get("video:part1") // 第2次

	fmt.Println("\n=== 多次访问后的缓存状态 ===")
	printCustomLFUStatus(cache)

	// 用户访问新片段，此时part2或part3（频率均为1）中的一个应被淘汰
	cache.Put("video:ending", "结尾片段数据")

	fmt.Println("\n=== 添加新片段后的缓存状态 ===")
	printCustomLFUStatus(cache)

	// 再添加一个新片段，此时频率为1的最早片段应被淘汰
	cache.Put("video:credits", "演职员表数据")

	fmt.Println("\n=== 再次添加新片段后的缓存状态 ===")
	printCustomLFUStatus(cache)

	// 更新已有片段
	cache.Put("video:intro", "更新后的介绍片段数据")

	fmt.Println("\n=== 更新片段后的缓存状态 ===")
	printCustomLFUStatus(cache)
}

// 辅助函数：打印自定义LFU缓存状态
func printCustomLFUStatus(cache *CustomLFUCache) {
	// 按频率分组打印
	for freq := 1; freq <= 10; freq++ {
		if list, exists := cache.freqMap[freq]; exists && list.Len() > 0 {
			fmt.Printf("频率 %d:\n", freq)
			for node := list.Front(); node != nil; node = node.Next() {
				lfuNode := node.Value.(*CustomLFUNode)
				fmt.Printf("  键: %s, 值: %v\n", lfuNode.Key, lfuNode.Value)
			}
		}
	}
	fmt.Printf("当前最小频率: %d\n", cache.minFreq)
}
