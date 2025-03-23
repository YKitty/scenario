package practical_applications

/*
限流器 - 令牌桶/漏桶算法实现

原理：
限流器用于控制API或服务的访问速率，防止过载。实现方式主要有两种：
1. 令牌桶算法：以固定速率向桶中添加令牌，请求需要消耗令牌才能被处理
2. 漏桶算法：请求以任意速率进入桶中，但以固定速率流出并被处理

关键特点：
1. 令牌桶允许一定程度的突发流量，但总体速率受控
2. 漏桶对流量起平滑作用，以固定速率处理请求
3. 支持分布式环境下的限流
4. 可配置的速率和容量参数

实现方式：
- 令牌桶：使用计时器定期添加令牌，使用原子操作进行令牌计数
- 漏桶：使用队列和定时器实现固定处理速率

应用场景：
- API访问频率控制
- 防止DoS攻击
- 服务降级和保护
- 资源使用限制（如数据库连接数）

优缺点：
- 优点：有效控制访问速率，防止资源耗尽
- 缺点：可能增加请求延迟，需要合理配置参数

以下实现了令牌桶和漏桶两种限流算法。
*/

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	// Allow 判断当前请求是否允许通过
	Allow() bool
	// AllowN 判断N个请求是否允许通过
	AllowN(n int64) bool
	// Wait 等待直到有足够的令牌可用或上下文取消
	Wait(ctx context.Context) error
	// WaitN 等待直到有N个令牌可用或上下文取消
	WaitN(ctx context.Context, n int64) error
	// GetStats 获取限流器统计信息
	GetStats() map[string]interface{}
}

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	rate           int64      // 令牌生成速率（每秒）
	capacity       int64      // 桶容量
	tokens         int64      // 当前令牌数
	lastRefillTime int64      // 上次令牌补充时间（Unix纳秒）
	mutex          sync.Mutex // 互斥锁
	accessCount    int64      // 请求总数
	limitedCount   int64      // 被限制的请求数
	passedCount    int64      // 通过的请求数
}

// NewTokenBucket 创建新的令牌桶限流器
func NewTokenBucket(rate, capacity int64) *TokenBucket {
	if rate <= 0 {
		rate = 1
	}
	if capacity <= 0 {
		capacity = rate
	}

	return &TokenBucket{
		rate:           rate,
		capacity:       capacity,
		tokens:         capacity, // 初始状态桶是满的
		lastRefillTime: time.Now().UnixNano(),
	}
}

// refillTokens 补充令牌
func (tb *TokenBucket) refillTokens() {
	now := time.Now().UnixNano()
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// 计算自上次补充以来经过的时间（秒）
	elapsed := float64(now-tb.lastRefillTime) / float64(time.Second.Nanoseconds())

	// 计算应该添加的令牌数
	newTokens := int64(elapsed * float64(tb.rate))
	if newTokens > 0 {
		tb.tokens = min(tb.capacity, tb.tokens+newTokens)
		tb.lastRefillTime = now
	}
}

// Allow 判断当前请求是否允许通过
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN 判断N个请求是否允许通过
func (tb *TokenBucket) AllowN(n int64) bool {
	if n <= 0 {
		return true
	}

	atomic.AddInt64(&tb.accessCount, 1)
	tb.refillTokens()

	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	if tb.tokens >= n {
		tb.tokens -= n
		atomic.AddInt64(&tb.passedCount, 1)
		return true
	}

	atomic.AddInt64(&tb.limitedCount, 1)
	return false
}

// Wait 等待直到有足够的令牌可用或上下文取消
func (tb *TokenBucket) Wait(ctx context.Context) error {
	return tb.WaitN(ctx, 1)
}

// WaitN 等待直到有N个令牌可用或上下文取消
func (tb *TokenBucket) WaitN(ctx context.Context, n int64) error {
	if n <= 0 {
		return nil
	}

	atomic.AddInt64(&tb.accessCount, 1)

	for {
		select {
		case <-ctx.Done():
			atomic.AddInt64(&tb.limitedCount, 1)
			return ctx.Err()
		default:
			tb.refillTokens()
			tb.mutex.Lock()
			if tb.tokens >= n {
				tb.tokens -= n
				atomic.AddInt64(&tb.passedCount, 1)
				tb.mutex.Unlock()
				return nil
			}
			tb.mutex.Unlock()

			// 计算等待时间
			waitTime := time.Duration(float64(n-tb.tokens) / float64(tb.rate) * float64(time.Second))
			if waitTime < time.Millisecond {
				waitTime = time.Millisecond
			}

			// 设置定时器等待
			timer := time.NewTimer(waitTime)
			select {
			case <-ctx.Done():
				timer.Stop()
				atomic.AddInt64(&tb.limitedCount, 1)
				return ctx.Err()
			case <-timer.C:
				// 继续尝试获取令牌
			}
		}
	}
}

// GetStats 获取令牌桶统计信息
func (tb *TokenBucket) GetStats() map[string]interface{} {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	return map[string]interface{}{
		"type":         "令牌桶",
		"rate":         tb.rate,
		"capacity":     tb.capacity,
		"current":      tb.tokens,
		"accessCount":  atomic.LoadInt64(&tb.accessCount),
		"passedCount":  atomic.LoadInt64(&tb.passedCount),
		"limitedCount": atomic.LoadInt64(&tb.limitedCount),
	}
}

// LeakyBucket 漏桶限流器
type LeakyBucket struct {
	rate         int64          // 漏出速率（每秒）
	capacity     int64          // 桶容量
	water        int64          // 当前水量
	lastLeakTime int64          // 上次漏水时间（Unix纳秒）
	mutex        sync.Mutex     // 互斥锁
	waiters      *PriorityQueue // 等待队列
	accessCount  int64          // 请求总数
	limitedCount int64          // 被限制的请求数
	passedCount  int64          // 通过的请求数
}

// Waiter 等待请求
type Waiter struct {
	n       int64         // 需要的资源数
	readyCh chan struct{} // 准备好的通知通道
}

// PriorityQueue 优先队列
type PriorityQueue struct {
	items []*Waiter
	mutex sync.Mutex
}

// NewPriorityQueue 创建新的优先队列
func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		items: make([]*Waiter, 0),
	}
}

// Push 添加等待者到队列
func (pq *PriorityQueue) Push(w *Waiter) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	pq.items = append(pq.items, w)
}

// Pop 从队列中取出等待者
func (pq *PriorityQueue) Pop() *Waiter {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	if len(pq.items) == 0 {
		return nil
	}
	w := pq.items[0]
	pq.items = pq.items[1:]
	return w
}

// Len 返回队列长度
func (pq *PriorityQueue) Len() int {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	return len(pq.items)
}

// NewLeakyBucket 创建新的漏桶限流器
func NewLeakyBucket(rate, capacity int64) *LeakyBucket {
	if rate <= 0 {
		rate = 1
	}
	if capacity <= 0 {
		capacity = rate
	}

	lb := &LeakyBucket{
		rate:         rate,
		capacity:     capacity,
		water:        0, // 初始状态桶是空的
		lastLeakTime: time.Now().UnixNano(),
		waiters:      NewPriorityQueue(),
	}

	// 启动漏水协程
	go lb.leakingProcess()

	return lb
}

// leakingProcess 漏水过程
func (lb *LeakyBucket) leakingProcess() {
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

	for range ticker.C {
		lb.leak()
		lb.checkWaiters()
	}
}

// leak 漏水
func (lb *LeakyBucket) leak() {
	now := time.Now().UnixNano()

	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// 计算自上次漏水以来经过的时间（秒）
	elapsed := float64(now-lb.lastLeakTime) / float64(time.Second.Nanoseconds())

	// 计算应该漏出的水量
	leakedWater := int64(elapsed * float64(lb.rate))
	if leakedWater > 0 {
		lb.water = max(0, lb.water-leakedWater)
		lb.lastLeakTime = now
	}
}

// checkWaiters 检查等待队列
func (lb *LeakyBucket) checkWaiters() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	for lb.waiters.Len() > 0 {
		waiter := lb.waiters.Pop()
		if lb.water+waiter.n <= lb.capacity {
			lb.water += waiter.n
			close(waiter.readyCh)
		} else {
			// 放回队列，等待下次检查
			lb.waiters.Push(waiter)
			break
		}
	}
}

// Allow 判断当前请求是否允许通过
func (lb *LeakyBucket) Allow() bool {
	return lb.AllowN(1)
}

// AllowN 判断N个请求是否允许通过
func (lb *LeakyBucket) AllowN(n int64) bool {
	if n <= 0 {
		return true
	}

	atomic.AddInt64(&lb.accessCount, 1)
	lb.leak()

	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	if lb.water+n <= lb.capacity {
		lb.water += n
		atomic.AddInt64(&lb.passedCount, 1)
		return true
	}

	atomic.AddInt64(&lb.limitedCount, 1)
	return false
}

// Wait 等待直到有足够的空间或上下文取消
func (lb *LeakyBucket) Wait(ctx context.Context) error {
	return lb.WaitN(ctx, 1)
}

// WaitN 等待直到有N个空间或上下文取消
func (lb *LeakyBucket) WaitN(ctx context.Context, n int64) error {
	if n <= 0 {
		return nil
	}

	atomic.AddInt64(&lb.accessCount, 1)
	lb.leak()

	lb.mutex.Lock()
	if lb.water+n <= lb.capacity {
		lb.water += n
		atomic.AddInt64(&lb.passedCount, 1)
		lb.mutex.Unlock()
		return nil
	}
	lb.mutex.Unlock()

	// 创建等待者
	readyCh := make(chan struct{})
	waiter := &Waiter{
		n:       n,
		readyCh: readyCh,
	}

	// 添加到等待队列
	lb.waiters.Push(waiter)

	// 等待信号或上下文取消
	select {
	case <-readyCh:
		atomic.AddInt64(&lb.passedCount, 1)
		return nil
	case <-ctx.Done():
		atomic.AddInt64(&lb.limitedCount, 1)
		return ctx.Err()
	}
}

// GetStats 获取漏桶统计信息
func (lb *LeakyBucket) GetStats() map[string]interface{} {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	return map[string]interface{}{
		"type":         "漏桶",
		"rate":         lb.rate,
		"capacity":     lb.capacity,
		"current":      lb.water,
		"waiting":      lb.waiters.Len(),
		"accessCount":  atomic.LoadInt64(&lb.accessCount),
		"passedCount":  atomic.LoadInt64(&lb.passedCount),
		"limitedCount": atomic.LoadInt64(&lb.limitedCount),
	}
}

// 辅助函数
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// 场景示例：API访问限流
func RateLimiterDemo() {
	fmt.Println("API访问限流示例:")

	// 创建令牌桶限流器，每秒5个请求，最多允许10个突发请求
	tokenBucket := NewTokenBucket(5, 10)

	// 创建漏桶限流器，每秒5个请求，最多积压10个请求
	leakyBucket := NewLeakyBucket(5, 10)

	// 使用两种限流器执行同样的测试
	testRateLimiter := func(name string, limiter RateLimiter) {
		fmt.Printf("\n测试%s限流器:\n", name)

		// 1. 测试突发请求
		fmt.Println("模拟突发请求(15个):")
		passed := 0
		for i := 0; i < 15; i++ {
			if limiter.Allow() {
				passed++
				fmt.Printf("请求 %d: 通过\n", i+1)
			} else {
				fmt.Printf("请求 %d: 限流\n", i+1)
			}
		}
		fmt.Printf("突发请求通过率: %d/%d\n", passed, 15)

		// 2. 等待一段时间后再次测试
		fmt.Println("\n等待2秒后继续请求...")
		time.Sleep(2 * time.Second)

		// 3. 测试等待模式
		fmt.Println("模拟10个带等待的请求:")
		ctx := context.Background()
		for i := 0; i < 10; i++ {
			start := time.Now()
			err := limiter.Wait(ctx)
			elapsed := time.Since(start)
			if err != nil {
				fmt.Printf("请求 %d: 等待失败 - %v\n", i+1, err)
			} else {
				fmt.Printf("请求 %d: 等待 %v 后通过\n", i+1, elapsed.Round(time.Millisecond))
			}
			// 短暂睡眠，避免所有请求同时发出
			time.Sleep(50 * time.Millisecond)
		}

		// 4. 显示限流器状态
		stats := limiter.GetStats()
		fmt.Println("\n限流器统计:")
		for k, v := range stats {
			fmt.Printf("%s: %v\n", k, v)
		}
	}

	// 测试令牌桶
	testRateLimiter("令牌桶", tokenBucket)

	// 测试漏桶
	testRateLimiter("漏桶", leakyBucket)

	// 5. 对比两种限流器的结果
	fmt.Println("\n两种限流器对比:")
	fmt.Println("- 令牌桶允许突发流量，初始状态可以处理更多请求")
	fmt.Println("- 漏桶对请求进行排队，平滑处理速率更稳定")
	fmt.Println("- 两者都能有效控制长期的请求速率")
}
