package cache_strategies

/*
TTL（Time To Live）缓存

原理：
TTL缓存为每个缓存项设置一个过期时间，当缓存项被访问时，如果已经过期则自动删除。
这种机制确保缓存中的数据始终是"新鲜"的，过期的数据会被自动清理。

关键特点：
1. 每个缓存项都有自己的过期时间
2. 支持两种过期策略：
   a. 懒惰过期：仅在访问时检查过期
   b. 主动过期：后台任务定期清理过期项
3. 可以设置默认过期时间，也可以为单个项设置特定过期时间
4. 支持设置永不过期的项

实现方式：
- 哈希表存储缓存项及其元数据(过期时间等)
- 可选的定时器进行周期性清理
- 访问时进行过期检查

应用场景：
- 会话管理（Session缓存）
- API响应缓存
- DNS缓存
- 临时凭证存储
- 任何需要自动失效的数据缓存场景

优缺点：
- 优点：自动管理数据新鲜度，不需要手动清理
- 缺点：需要额外存储过期时间信息，检查过期会有小的性能开销

以下实现了一个带TTL功能的缓存，支持懒惰过期和周期性清理。
*/

import (
	"fmt"
	"sync"
	"time"
)

// TTLCacheItem TTL缓存项结构
type TTLCacheItem struct {
	Key        string
	Value      interface{}
	ExpireTime time.Time // 过期时间点
}

// IsExpired 检查缓存项是否已过期
func (item *TTLCacheItem) IsExpired() bool {
	return !item.ExpireTime.IsZero() && time.Now().After(item.ExpireTime)
}

// TTLCache TTL缓存结构
type TTLCache struct {
	items           map[string]*TTLCacheItem // 缓存项
	mutex           sync.RWMutex             // 读写锁
	defaultTTL      time.Duration            // 默认过期时间
	cleanupInterval time.Duration            // 清理间隔
	stopCleanup     chan bool                // 停止清理的信号
}

// TTLCacheOptions TTL缓存配置选项
type TTLCacheOptions struct {
	DefaultTTL      time.Duration // 默认过期时间
	CleanupInterval time.Duration // 清理间隔
}

// DefaultTTLCacheOptions 默认的TTL缓存配置
var DefaultTTLCacheOptions = TTLCacheOptions{
	DefaultTTL:      time.Minute * 5, // 默认5分钟过期
	CleanupInterval: time.Minute * 1, // 每分钟清理一次
}

// NewTTLCache 创建新的TTL缓存
func NewTTLCache(options ...TTLCacheOptions) *TTLCache {
	opts := DefaultTTLCacheOptions
	if len(options) > 0 {
		opts = options[0]
	}

	cache := &TTLCache{
		items:           make(map[string]*TTLCacheItem),
		defaultTTL:      opts.DefaultTTL,
		cleanupInterval: opts.CleanupInterval,
		stopCleanup:     make(chan bool),
	}

	// 启动后台清理任务
	if opts.CleanupInterval > 0 {
		go cache.startCleanupTimer()
	}

	return cache
}

// startCleanupTimer 启动清理定时器
func (c *TTLCache) startCleanupTimer() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// StopCleanup 停止清理定时器
func (c *TTLCache) StopCleanup() {
	c.stopCleanup <- true
}

// Cleanup 执行过期项清理
func (c *TTLCache) Cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if !item.ExpireTime.IsZero() && now.After(item.ExpireTime) {
			delete(c.items, key)
		}
	}
}

// Set 设置缓存，使用默认过期时间
func (c *TTLCache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL 设置缓存，指定过期时间
func (c *TTLCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var expireTime time.Time
	if ttl > 0 {
		expireTime = time.Now().Add(ttl)
	}

	c.items[key] = &TTLCacheItem{
		Key:        key,
		Value:      value,
		ExpireTime: expireTime,
	}
}

// SetForever 设置永不过期的缓存项
func (c *TTLCache) SetForever(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[key] = &TTLCacheItem{
		Key:        key,
		Value:      value,
		ExpireTime: time.Time{}, // 零值表示永不过期
	}
}

// Get 获取缓存值，如果不存在或已过期则返回nil和false
func (c *TTLCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	item, found := c.items[key]
	c.mutex.RUnlock()

	if !found {
		return nil, false
	}

	// 懒惰过期检查
	if item.IsExpired() {
		c.mutex.Lock()
		delete(c.items, key)
		c.mutex.Unlock()
		return nil, false
	}

	return item.Value, true
}

// Remove 删除缓存项
func (c *TTLCache) Remove(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, found := c.items[key]; found {
		delete(c.items, key)
		return true
	}
	return false
}

// Size 返回当前缓存中的元素数量（包括已过期但未清理的）
func (c *TTLCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.items)
}

// Clear 清空缓存
func (c *TTLCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.items = make(map[string]*TTLCacheItem)
}

// Keys 返回缓存中所有未过期键的列表
func (c *TTLCache) Keys() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	keys := make([]string, 0, len(c.items))
	now := time.Now()

	for key, item := range c.items {
		if item.ExpireTime.IsZero() || now.Before(item.ExpireTime) {
			keys = append(keys, key)
		}
	}

	return keys
}

// 场景示例：会话管理系统
func TTLCacheDemo() {
	// 创建TTL缓存，设置较短的过期时间用于演示
	options := TTLCacheOptions{
		DefaultTTL:      time.Second * 3, // 默认3秒过期
		CleanupInterval: time.Second * 1, // 每秒清理一次
	}
	cache := NewTTLCache(options)
	defer cache.StopCleanup() // 确保在函数结束时停止清理任务

	fmt.Println("会话管理系统示例 (TTL缓存):")

	// 模拟用户登录，创建会话
	cache.Set("session:user1", map[string]string{
		"userId": "user1",
		"name":   "张三",
		"role":   "admin",
	})

	cache.SetWithTTL("session:user2", map[string]string{
		"userId": "user2",
		"name":   "李四",
		"role":   "user",
	}, time.Second*5) // 5秒过期

	cache.SetForever("config:system", map[string]string{
		"version":     "1.0",
		"environment": "production",
	}) // 永不过期

	// 初始状态
	fmt.Println("\n=== 初始会话状态 ===")
	printTTLCacheStatus(cache)

	// 访问会话
	if session, found := cache.Get("session:user1"); found {
		fmt.Printf("用户已登录: %v\n", session)
	}

	// 等待2秒，此时user1会话仍有效
	fmt.Println("\n正在等待2秒...")
	time.Sleep(time.Second * 2)

	fmt.Println("\n=== 2秒后状态 ===")
	printTTLCacheStatus(cache)

	// 再等待2秒，此时user1会话应已过期
	fmt.Println("\n再等待2秒...")
	time.Sleep(time.Second * 2)

	fmt.Println("\n=== 4秒后状态（user1会话已过期） ===")
	printTTLCacheStatus(cache)

	// 验证user1会话已过期
	if _, found := cache.Get("session:user1"); found {
		fmt.Println("用户1会话仍有效")
	} else {
		fmt.Println("用户1会话已过期")
	}

	// 但user2会话和系统配置仍有效
	if _, found := cache.Get("session:user2"); found {
		fmt.Println("用户2会话仍有效")
	}

	if _, found := cache.Get("config:system"); found {
		fmt.Println("系统配置永不过期")
	}

	// 再等待2秒，此时user2会话也应过期
	fmt.Println("\n再等待2秒...")
	time.Sleep(time.Second * 2)

	fmt.Println("\n=== 6秒后状态（两个用户会话均已过期） ===")
	printTTLCacheStatus(cache)
}

// 辅助函数：打印TTL缓存状态
func printTTLCacheStatus(cache *TTLCache) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	now := time.Now()
	fmt.Printf("当前缓存项数量: %d\n", len(cache.items))

	for key, item := range cache.items {
		var expireInfo string
		if item.ExpireTime.IsZero() {
			expireInfo = "永不过期"
		} else {
			remaining := item.ExpireTime.Sub(now)
			if remaining > 0 {
				expireInfo = fmt.Sprintf("剩余 %.1f 秒", remaining.Seconds())
			} else {
				expireInfo = "已过期"
			}
		}

		fmt.Printf("键: %s, 过期状态: %s\n", key, expireInfo)
	}
}
