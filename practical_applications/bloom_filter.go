package practical_applications

/*
布隆过滤器 - 大数据集合的成员检测

原理：
布隆过滤器是一种空间效率很高的概率型数据结构，用于判断一个元素是否在一个集合中。
它可能返回错误的肯定结果（即有"假阳性"），但不会返回错误的否定结果（"假阴性"）。
简单来说，如果布隆过滤器说元素不在集合中，那它一定不在；如果说元素在集合中，那它可能在。

关键特点：
1. 空间效率高，使用固定大小的位数组
2. 插入和查询操作都是O(k)复杂度，k是哈希函数的数量
3. 不能删除元素（有变种可以支持删除）
4. 错误率随着存储的元素数量增加而增加
5. 可以控制准确率和内存使用的平衡

实现方式：
- 使用位数组（bit array）存储元素信息
- 使用多个哈希函数计算元素在位数组中的位置
- 通过参数调整，可以平衡错误率和内存使用

应用场景：
- 网页爬虫URL去重
- 垃圾邮件过滤
- 缓存穿透防护
- 大规模数据集合的快速查询
- 单词拼写检查

优缺点：
- 优点：内存占用少，查询速度快
- 缺点：有一定的错误率，无法直接删除元素

以下实现了一个基本的布隆过滤器，支持插入和查询操作，并可以估计错误率。
*/

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"sync"
	"time"
)

// BloomFilter 布隆过滤器结构
type BloomFilter struct {
	bitArray    []bool // 位数组
	size        uint   // 位数组大小
	hashFuncs   uint   // 哈希函数数量
	count       uint   // 已插入元素数量
	mutex       sync.RWMutex
	hashFuncGen func(index uint) func(data []byte) uint // 哈希函数生成器
}

// NewBloomFilter 创建指定大小和哈希函数数量的布隆过滤器
func NewBloomFilter(size uint, hashFuncs uint) *BloomFilter {
	return &BloomFilter{
		bitArray:    make([]bool, size),
		size:        size,
		hashFuncs:   hashFuncs,
		count:       0,
		hashFuncGen: defaultHashFuncGenerator,
	}
}

// NewBloomFilterWithParams 根据预期元素数量和期望错误率创建布隆过滤器
func NewBloomFilterWithParams(expectedItems uint, falsePositiveRate float64) *BloomFilter {
	// 计算最佳大小
	size := uint(math.Ceil(-float64(expectedItems) * math.Log(falsePositiveRate) / math.Pow(math.Log(2), 2)))

	// 计算最佳哈希函数数量
	hashFuncs := uint(math.Ceil(float64(size) / float64(expectedItems) * math.Log(2)))

	// 确保至少有1个哈希函数
	if hashFuncs < 1 {
		hashFuncs = 1
	}

	return NewBloomFilter(size, hashFuncs)
}

// defaultHashFuncGenerator 默认哈希函数生成器
func defaultHashFuncGenerator(index uint) func(data []byte) uint {
	return func(data []byte) uint {
		var h hash.Hash

		// 根据index选择不同的哈希算法
		switch index % 3 {
		case 0:
			h = fnv.New64a()
		case 1:
			h = md5.New()
		case 2:
			h = sha1.New()
		}

		h.Write(data)
		sum := h.Sum(nil)

		// 将哈希值转为uint64
		var val uint64
		if len(sum) >= 8 {
			val = binary.BigEndian.Uint64(sum[:8])
		} else {
			val = uint64(binary.BigEndian.Uint32(sum[:4]))
		}

		// 使用位运算确保不同的index得到不同的哈希值
		val ^= (val >> 13) * uint64(index+1)
		val ^= (val << 7) * uint64(index+1)

		// 对数组大小取模得到位置
		return uint(val%uint64(math.MaxUint32)) % uint(len(sum))
	}
}

// Add 向布隆过滤器中添加元素
func (bf *BloomFilter) Add(data []byte) {
	if data == nil || len(data) == 0 {
		return
	}

	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	// 使用多个哈希函数计算位置并设置对应位
	for i := uint(0); i < bf.hashFuncs; i++ {
		hashFunc := bf.hashFuncGen(i)
		position := hashFunc(data) % bf.size
		bf.bitArray[position] = true
	}

	bf.count++
}

// AddString 添加字符串元素
func (bf *BloomFilter) AddString(s string) {
	bf.Add([]byte(s))
}

// Contains 检查元素是否可能在布隆过滤器中
func (bf *BloomFilter) Contains(data []byte) bool {
	if data == nil || len(data) == 0 {
		return false
	}

	bf.mutex.RLock()
	defer bf.mutex.RUnlock()

	// 检查所有哈希位置的位是否都被设置
	for i := uint(0); i < bf.hashFuncs; i++ {
		hashFunc := bf.hashFuncGen(i)
		position := hashFunc(data) % bf.size
		if !bf.bitArray[position] {
			return false // 如果有一个位未设置，元素肯定不在集合中
		}
	}

	// 所有位都被设置，元素可能在集合中
	return true
}

// ContainsString 检查字符串元素是否可能在布隆过滤器中
func (bf *BloomFilter) ContainsString(s string) bool {
	return bf.Contains([]byte(s))
}

// EstimatedFalsePositiveRate 估计当前的假阳性率
func (bf *BloomFilter) EstimatedFalsePositiveRate() float64 {
	bf.mutex.RLock()
	defer bf.mutex.RUnlock()

	// 根据布隆过滤器理论公式计算假阳性率
	// 公式: (1 - e^(-k*n/m))^k
	// k: 哈希函数数量, n: 元素数量, m: 位数组大小
	k := float64(bf.hashFuncs)
	n := float64(bf.count)
	m := float64(bf.size)

	return math.Pow(1.0-math.Exp(-k*n/m), k)
}

// Reset 重置布隆过滤器
func (bf *BloomFilter) Reset() {
	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	bf.bitArray = make([]bool, bf.size)
	bf.count = 0
}

// Count 返回已添加的元素数量
func (bf *BloomFilter) Count() uint {
	bf.mutex.RLock()
	defer bf.mutex.RUnlock()
	return bf.count
}

// Info 返回布隆过滤器的基本信息
func (bf *BloomFilter) Info() map[string]interface{} {
	bf.mutex.RLock()
	defer bf.mutex.RUnlock()

	// 计算设置的位数
	setBits := uint(0)
	for _, bit := range bf.bitArray {
		if bit {
			setBits++
		}
	}

	return map[string]interface{}{
		"size":                 bf.size,
		"hashFunctions":        bf.hashFuncs,
		"itemCount":            bf.count,
		"setBitsCount":         setBits,
		"setBitsPercentage":    float64(setBits) / float64(bf.size) * 100,
		"estimatedErrorRate":   bf.EstimatedFalsePositiveRate(),
		"theoreticalErrorRate": math.Pow(1-math.Exp(-float64(bf.hashFuncs)*float64(bf.count)/float64(bf.size)), float64(bf.hashFuncs)),
	}
}

// 场景示例：网页爬虫URL去重
func BloomFilterDemo() {
	fmt.Println("布隆过滤器示例 - 网页爬虫URL去重:")

	// 创建布隆过滤器，预期处理100万个URL，错误率为0.1%
	filter := NewBloomFilterWithParams(1000000, 0.001)

	// 模拟已爬取的URL
	crawledURLs := []string{
		"https://example.com/",
		"https://example.com/about",
		"https://example.com/products",
		"https://example.com/contact",
		"https://example.com/blog/post1",
		"https://example.com/blog/post2",
		"https://example.com/users/profile",
	}

	// 添加已爬取的URL到布隆过滤器
	for _, url := range crawledURLs {
		filter.AddString(url)
	}

	// 显示布隆过滤器信息
	fmt.Println("布隆过滤器初始化完成:")
	info := filter.Info()
	fmt.Printf("  位数组大小: %d 位\n", info["size"])
	fmt.Printf("  哈希函数数量: %d\n", info["hashFunctions"])
	fmt.Printf("  已添加URL数量: %d\n", info["itemCount"])
	fmt.Printf("  已设置位数量: %d (%.2f%%)\n", info["setBitsCount"], info["setBitsPercentage"])
	fmt.Printf("  估计错误率: %.6f%%\n", info["estimatedErrorRate"].(float64)*100)

	// 模拟新的爬虫任务，检查URL是否已被爬取
	fmt.Println("\n模拟新的爬虫任务:")
	newURLs := []string{
		"https://example.com/",               // 已爬取
		"https://example.com/new-page",       // 未爬取
		"https://example.com/blog/post2",     // 已爬取
		"https://example.com/products/item1", // 未爬取
		"https://example.com/users/profile",  // 已爬取
		"https://example.com/category",       // 未爬取
	}

	for _, url := range newURLs {
		if filter.ContainsString(url) {
			fmt.Printf("URL已爬取 (跳过): %s\n", url)
		} else {
			fmt.Printf("URL未爬取 (添加到队列): %s\n", url)
			// 模拟爬取操作
			filter.AddString(url)
		}
	}

	// 测试假阳性率
	fmt.Println("\n测试布隆过滤器的假阳性率:")

	// 生成一批随机URL
	randomURLs := generateRandomURLs(10000)
	falsePositives := 0

	start := time.Now()
	for _, url := range randomURLs {
		if filter.ContainsString(url) {
			falsePositives++
		}
	}
	duration := time.Since(start)

	fmt.Printf("测试了 %d 个随机URL (%.2f ms)\n", len(randomURLs), float64(duration.Milliseconds()))
	fmt.Printf("假阳性数量: %d (%.4f%%)\n", falsePositives, float64(falsePositives)/float64(len(randomURLs))*100)
	fmt.Printf("理论错误率: %.4f%%\n", filter.EstimatedFalsePositiveRate()*100)

	// 内存占用对比
	fmt.Println("\n内存占用对比:")
	bloomSize := float64(filter.size) / 8 / 1024 // 位数组大小转换为KB
	mapSize := float64(filter.Count()*40) / 1024 // 假设使用map, 每个URL平均40字节

	fmt.Printf("布隆过滤器占用内存: %.2f KB\n", bloomSize)
	fmt.Printf("使用map存储占用预估: %.2f KB\n", mapSize)
	fmt.Printf("内存节省: %.1f 倍\n", mapSize/bloomSize)
}

// 生成随机URL，用于测试假阳性率
func generateRandomURLs(count int) []string {
	urls := make([]string, count)
	for i := 0; i < count; i++ {
		paths := []string{
			fmt.Sprintf("page%d", i),
			fmt.Sprintf("article%d", i),
			fmt.Sprintf("product%d", i),
			fmt.Sprintf("user%d", i),
			fmt.Sprintf("category%d/item%d", i%50, i),
		}
		urls[i] = fmt.Sprintf("https://randomsite%d.com/%s", i%100, paths[i%5])
	}
	return urls
}
