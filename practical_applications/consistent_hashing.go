package practical_applications

/*
一致性哈希 - 分布式系统负载均衡

原理：
一致性哈希是一种特殊的哈希算法，用于解决分布式缓存系统中的数据分布问题。
传统哈希在节点增减时会导致大量键的重新映射，而一致性哈希只影响部分键，
大大减少了数据迁移量，实现了分布式系统的可扩展性和高可用性。

关键特点：
1. 平衡性：数据尽量均匀分布到所有节点
2. 单调性：添加新节点时，只需重新分配部分键
3. 分散性：同一个客户端的不同请求可以分散到不同节点
4. 负载：尽量均匀分配负载（可通过虚拟节点实现）
5. 平滑性：节点变化时数据迁移量最小化

实现方式：
- 将所有节点映射到一个环上（0-2^32-1的范围）
- 键也映射到环上，并顺时针找到第一个节点
- 通过引入虚拟节点提高均衡性

应用场景：
- 分布式缓存系统（如Memcached）
- 分布式存储系统
- 负载均衡器
- 分布式数据库分片
- 内容分发网络(CDN)的路由

优缺点：
- 优点：节点变动时最小化数据迁移，良好的扩展性
- 缺点：实现复杂度较高，可能需要调整虚拟节点数量

以下实现了一个基本的一致性哈希算法，支持添加和删除节点，以及查找键对应的节点。
*/

import (
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

// 常量定义
const (
	DefaultVirtualNodes = 150 // 默认虚拟节点数量
)

// ConsistentHash 一致性哈希结构
type ConsistentHash struct {
	circle         map[uint32]string // 哈希环
	sortedHashes   []uint32          // 已排序的哈希值列表
	virtualNodes   int               // 每个真实节点对应的虚拟节点数
	nodes          map[string]bool   // 真实节点集合
	mutex          sync.RWMutex      // 读写锁
	customHashFunc HashFunc          // 自定义哈希函数
}

// HashFunc 哈希函数类型
type HashFunc func(data []byte) uint32

// NewConsistentHash 创建新的一致性哈希实例
func NewConsistentHash(virtualNodes int) *ConsistentHash {
	if virtualNodes <= 0 {
		virtualNodes = DefaultVirtualNodes
	}

	return &ConsistentHash{
		circle:         make(map[uint32]string),
		sortedHashes:   make([]uint32, 0),
		virtualNodes:   virtualNodes,
		nodes:          make(map[string]bool),
		customHashFunc: crc32.ChecksumIEEE,
	}
}

// SetHashFunc 设置自定义哈希函数
func (ch *ConsistentHash) SetHashFunc(fn HashFunc) {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	ch.customHashFunc = fn
	// 重建哈希环
	ch.rebuild()
}

// AddNode 添加新节点
func (ch *ConsistentHash) AddNode(node string) bool {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	// 检查节点是否已存在
	if _, exists := ch.nodes[node]; exists {
		return false
	}

	// 添加到节点集合
	ch.nodes[node] = true

	// 为该节点创建虚拟节点
	for i := 0; i < ch.virtualNodes; i++ {
		virtualNodeName := ch.getVirtualNodeName(node, i)
		hash := ch.hashKey(virtualNodeName)
		ch.circle[hash] = node
		ch.sortedHashes = append(ch.sortedHashes, hash)
	}

	// 重新排序
	sort.Slice(ch.sortedHashes, func(i, j int) bool {
		return ch.sortedHashes[i] < ch.sortedHashes[j]
	})

	return true
}

// RemoveNode 移除节点
func (ch *ConsistentHash) RemoveNode(node string) bool {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	// 检查节点是否存在
	if _, exists := ch.nodes[node]; !exists {
		return false
	}

	// 从节点集合中移除
	delete(ch.nodes, node)

	// 移除该节点的所有虚拟节点
	newHashes := make([]uint32, 0, len(ch.sortedHashes)-ch.virtualNodes)
	for i := 0; i < ch.virtualNodes; i++ {
		virtualNodeName := ch.getVirtualNodeName(node, i)
		hash := ch.hashKey(virtualNodeName)
		delete(ch.circle, hash)
	}

	// 重建排序的哈希值列表
	for hash := range ch.circle {
		newHashes = append(newHashes, hash)
	}

	ch.sortedHashes = newHashes
	sort.Slice(ch.sortedHashes, func(i, j int) bool {
		return ch.sortedHashes[i] < ch.sortedHashes[j]
	})

	return true
}

// GetNode 获取键对应的节点
func (ch *ConsistentHash) GetNode(key string) (string, bool) {
	if len(ch.nodes) == 0 {
		return "", false
	}

	ch.mutex.RLock()
	defer ch.mutex.RUnlock()

	hash := ch.hashKey(key)

	// 二分查找最接近的节点
	idx := ch.findNearestNodeIndex(hash)
	if idx == len(ch.sortedHashes) {
		idx = 0 // 如果超过了最大哈希值，回到环的起点
	}

	return ch.circle[ch.sortedHashes[idx]], true
}

// 查找最接近的节点索引（二分查找）
func (ch *ConsistentHash) findNearestNodeIndex(hash uint32) int {
	idx := sort.Search(len(ch.sortedHashes), func(i int) bool {
		return ch.sortedHashes[i] >= hash
	})

	if idx >= len(ch.sortedHashes) {
		idx = 0
	}

	return idx
}

// 获取虚拟节点名称
func (ch *ConsistentHash) getVirtualNodeName(node string, index int) string {
	return node + ":" + strconv.Itoa(index)
}

// 计算键的哈希值
func (ch *ConsistentHash) hashKey(key string) uint32 {
	return ch.customHashFunc([]byte(key))
}

// 重建哈希环
func (ch *ConsistentHash) rebuild() {
	// 清空哈希环
	ch.circle = make(map[uint32]string)
	ch.sortedHashes = make([]uint32, 0)

	// 重新添加所有节点
	for node := range ch.nodes {
		for i := 0; i < ch.virtualNodes; i++ {
			virtualNodeName := ch.getVirtualNodeName(node, i)
			hash := ch.hashKey(virtualNodeName)
			ch.circle[hash] = node
			ch.sortedHashes = append(ch.sortedHashes, hash)
		}
	}

	// 重新排序
	sort.Slice(ch.sortedHashes, func(i, j int) bool {
		return ch.sortedHashes[i] < ch.sortedHashes[j]
	})
}

// GetNodeCount 获取当前节点数量
func (ch *ConsistentHash) GetNodeCount() int {
	ch.mutex.RLock()
	defer ch.mutex.RUnlock()
	return len(ch.nodes)
}

// GetDistribution 获取键在节点上的分布情况
func (ch *ConsistentHash) GetDistribution(keys []string) map[string]int {
	ch.mutex.RLock()
	defer ch.mutex.RUnlock()

	distribution := make(map[string]int)

	// 初始化分布计数
	for node := range ch.nodes {
		distribution[node] = 0
	}

	// 计算每个键对应的节点
	for _, key := range keys {
		if node, ok := ch.GetNode(key); ok {
			distribution[node]++
		}
	}

	return distribution
}

// 场景示例：分布式缓存系统
func ConsistentHashingDemo() {
	fmt.Println("一致性哈希示例 - 分布式缓存系统:")

	// 创建一致性哈希实例
	ch := NewConsistentHash(100) // 每个节点100个虚拟节点

	// 添加初始缓存服务器
	initialServers := []string{
		"cache-01.example.com",
		"cache-02.example.com",
		"cache-03.example.com",
	}

	for _, server := range initialServers {
		ch.AddNode(server)
		fmt.Printf("添加服务器: %s\n", server)
	}

	// 生成测试键
	testKeys := make([]string, 1000)
	for i := 0; i < len(testKeys); i++ {
		testKeys[i] = fmt.Sprintf("user:%d:profile", i)
	}

	// 获取初始分布
	fmt.Println("\n初始键分布:")
	initialDistribution := ch.GetDistribution(testKeys)
	displayDistribution(initialDistribution, len(testKeys))

	// 添加新服务器
	fmt.Println("\n添加新服务器 cache-04.example.com:")
	ch.AddNode("cache-04.example.com")

	// 获取新的分布
	fmt.Println("\n添加服务器后的键分布:")
	newDistribution := ch.GetDistribution(testKeys)
	displayDistribution(newDistribution, len(testKeys))

	// 计算重新分配的键数量
	relocated := calculateRelocated(initialDistribution, newDistribution)
	fmt.Printf("\n添加服务器后，需要重新分配的键数量: %d (%.2f%%)\n",
		relocated, float64(relocated)/float64(len(testKeys))*100)

	// 移除服务器
	fmt.Println("\n移除服务器 cache-02.example.com:")
	ch.RemoveNode("cache-02.example.com")

	// 获取移除后的分布
	fmt.Println("\n移除服务器后的键分布:")
	removeDistribution := ch.GetDistribution(testKeys)
	displayDistribution(removeDistribution, len(testKeys))

	// 计算重新分配的键数量
	relocated = calculateRelocated(newDistribution, removeDistribution)
	fmt.Printf("\n移除服务器后，需要重新分配的键数量: %d (%.2f%%)\n",
		relocated, float64(relocated)/float64(len(testKeys))*100)

	// 演示查找特定键的服务器
	fmt.Println("\n查找特定键对应的服务器:")
	sampleKeys := []string{
		"user:42:profile",
		"user:123:profile",
		"user:789:profile",
		"product:abc123",
		"category:electronics",
	}

	for _, key := range sampleKeys {
		if server, ok := ch.GetNode(key); ok {
			fmt.Printf("键 '%s' 映射到服务器: %s\n", key, server)
		} else {
			fmt.Printf("键 '%s' 没有找到对应的服务器\n", key)
		}
	}

	// 对比传统哈希方法
	fmt.Println("\n传统哈希 vs. 一致性哈希 (在添加/删除节点时):")
	fmt.Println("传统哈希: 节点变化时，几乎所有键需要重新分配")
	fmt.Println("一致性哈希: 只有一小部分键需要重新分配")
	fmt.Println("  - 添加节点: 大约 1/n 的键需要重新分配")
	fmt.Println("  - 删除节点: 只有属于被删除节点的键需要重新分配")
}

// 显示分布情况
func displayDistribution(distribution map[string]int, total int) {
	for server, count := range distribution {
		percentage := float64(count) / float64(total) * 100
		fmt.Printf("  %s: %d 键 (%.2f%%)\n", server, count, percentage)
	}
}

// 计算重新分配的键数量
func calculateRelocated(oldDist, newDist map[string]int) int {
	relocated := 0

	// 获取所有服务器
	allServers := make(map[string]bool)
	for server := range oldDist {
		allServers[server] = true
	}
	for server := range newDist {
		allServers[server] = true
	}

	// 对于每个服务器，计算键的变化量
	for server := range allServers {
		oldCount := oldDist[server]
		newCount := newDist[server]

		// 如果是新增服务器，全部键都是重新分配的
		if oldCount == 0 {
			relocated += newCount
			continue
		}

		// 如果是移除的服务器，全部键都是重新分配的
		if newCount == 0 {
			relocated += oldCount
			continue
		}

		// 如果服务器存在于新旧分布中，取变化量的绝对值的一半
		// 这是因为一个键从A迁移到B，会导致A减1，B加1，所以变化量应该/2
		diff := abs(newCount - oldCount)
		relocated += diff
	}

	// 由于每个重新分配的键会计算两次（源和目标），所以除以2
	return relocated / 2
}

// 绝对值函数
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
