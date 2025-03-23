package concurrency

/*
信号量（Semaphore）

原理：
信号量是一种同步原语，用于控制对共享资源的访问。它维护一个计数器，表示可用资源的数量。
当线程/协程需要访问资源时，它会尝试减少计数器；当释放资源时，增加计数器。

关键特点：
1. 计数器维护可用资源数量
2. 支持阻塞操作（当计数器为0时，请求资源的线程会被阻塞）
3. 支持超时获取资源
4. 支持资源的公平分配（可选）

实现方式：
- 使用互斥锁和条件变量实现基本的同步机制
- 使用通道（channel）实现信号量行为
- 提供带超时的资源获取方法

应用场景：
- 限制对数据库连接的并发访问
- 控制对物理资源（如打印机）的访问
- 实现并发限制和速率限制
- 保护共享内存区域

优缺点：
- 优点：简单有效的并发控制机制，可防止资源耗尽
- 缺点：可能导致死锁，使用不当会影响性能

以下实现了一个计数信号量，支持阻塞和超时获取资源。
*/

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Semaphore 计数信号量
type Semaphore struct {
	capacity int           // 信号量容量（最大可用资源数）
	tokens   chan struct{} // 表示可用资源的令牌通道
	mu       sync.Mutex    // 用于保护内部状态的互斥锁
	waiting  int           // 当前等待获取资源的协程数
	acquired int           // 当前已获取资源的协程数
}

// NewSemaphore 创建新的信号量
func NewSemaphore(capacity int) *Semaphore {
	if capacity <= 0 {
		capacity = 1
	}

	// 创建一个带缓冲的通道作为令牌桶
	tokens := make(chan struct{}, capacity)

	// 初始化令牌桶
	for i := 0; i < capacity; i++ {
		tokens <- struct{}{}
	}

	return &Semaphore{
		capacity: capacity,
		tokens:   tokens,
		waiting:  0,
		acquired: 0,
	}
}

// Acquire 获取一个资源，如果没有可用资源则阻塞
func (s *Semaphore) Acquire() {
	s.mu.Lock()
	s.waiting++
	s.mu.Unlock()

	// 从令牌通道获取一个令牌（阻塞操作）
	<-s.tokens

	s.mu.Lock()
	s.waiting--
	s.acquired++
	s.mu.Unlock()
}

// TryAcquire 尝试获取一个资源，如果没有可用资源则立即返回false
func (s *Semaphore) TryAcquire() bool {
	select {
	case <-s.tokens:
		s.mu.Lock()
		s.acquired++
		s.mu.Unlock()
		return true
	default:
		return false
	}
}

// AcquireWithTimeout 尝试在指定超时时间内获取资源
func (s *Semaphore) AcquireWithTimeout(timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.AcquireWithContext(ctx)
}

// AcquireWithContext 尝试在上下文取消前获取资源
func (s *Semaphore) AcquireWithContext(ctx context.Context) bool {
	s.mu.Lock()
	s.waiting++
	s.mu.Unlock()

	// 尝试在上下文取消前获取令牌
	select {
	case <-s.tokens:
		s.mu.Lock()
		s.waiting--
		s.acquired++
		s.mu.Unlock()
		return true
	case <-ctx.Done():
		s.mu.Lock()
		s.waiting--
		s.mu.Unlock()
		return false
	}
}

// Release 释放一个资源
func (s *Semaphore) Release() {
	s.mu.Lock()
	// 只有在已获取资源的情况下才释放
	if s.acquired > 0 {
		s.acquired--
		s.mu.Unlock()
		// 将令牌放回通道
		s.tokens <- struct{}{}
	} else {
		s.mu.Unlock()
	}
}

// AvailablePermits 返回当前可用的资源数量
func (s *Semaphore) AvailablePermits() int {
	return len(s.tokens)
}

// Stats 返回信号量的统计信息
func (s *Semaphore) Stats() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]interface{}{
		"capacity":  s.capacity,
		"available": len(s.tokens),
		"acquired":  s.acquired,
		"waiting":   s.waiting,
	}
}

// 场景示例：模拟数据库连接池
func SemaphoreDemo() {
	fmt.Println("数据库连接池场景（信号量使用示例）:")

	// 创建一个容量为3的信号量，模拟最多3个并发数据库连接
	dbConnPool := NewSemaphore(3)

	// 模拟数据库查询
	queryDB := func(id int, query string, duration time.Duration) {
		fmt.Printf("客户端 %d: 尝试获取数据库连接执行查询: %s\n", id, query)

		// 尝试获取连接，超时时间为500毫秒
		acquired := dbConnPool.AcquireWithTimeout(time.Millisecond * 500)
		if !acquired {
			fmt.Printf("客户端 %d: 获取连接超时，查询失败: %s\n", id, query)
			return
		}

		// 成功获取连接
		fmt.Printf("客户端 %d: 成功获取连接，执行查询: %s\n", id, query)

		// 模拟查询执行时间
		time.Sleep(duration)

		// 释放连接
		dbConnPool.Release()
		fmt.Printf("客户端 %d: 查询完成，释放连接: %s\n", id, query)
	}

	// 启动多个客户端并发请求数据库连接
	var wg sync.WaitGroup

	// 模拟不同查询和执行时间
	queries := []struct {
		id       int
		query    string
		duration time.Duration
	}{
		{1, "SELECT * FROM users", time.Millisecond * 200},
		{2, "SELECT * FROM products", time.Millisecond * 300},
		{3, "UPDATE orders SET status = 'shipped'", time.Millisecond * 500},
		{4, "DELETE FROM cart WHERE user_id = 123", time.Millisecond * 200},
		{5, "INSERT INTO logs VALUES (...)", time.Millisecond * 100},
		{6, "SELECT COUNT(*) FROM events", time.Millisecond * 400},
		{7, "UPDATE inventory SET quantity = quantity - 1", time.Millisecond * 450},
		{8, "SELECT * FROM large_table", time.Millisecond * 800},
	}

	// 启动所有查询
	wg.Add(len(queries))
	for _, q := range queries {
		go func(id int, query string, duration time.Duration) {
			defer wg.Done()
			queryDB(id, query, duration)
		}(q.id, q.query, q.duration)

		// 稍微延迟下一个查询的启动时间
		time.Sleep(time.Millisecond * 50)
	}

	// 等待所有查询完成
	wg.Wait()

	// 显示最终统计信息
	stats := dbConnPool.Stats()
	fmt.Println("\n连接池统计:")
	fmt.Printf("总容量: %d\n", stats["capacity"])
	fmt.Printf("可用连接: %d\n", stats["available"])
	fmt.Printf("已获取连接: %d\n", stats["acquired"])
	fmt.Printf("等待连接: %d\n", stats["waiting"])
}
