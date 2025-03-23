package concurrency

/*
协程池（Goroutine Pool）实现

原理：
协程池是一种资源管理模式，预先创建一定数量的goroutine，通过任务队列向这些goroutine分配工作，
从而避免频繁创建和销毁goroutine带来的开销。

关键特点：
1. 控制并发度，限制同时运行的goroutine数量
2. 重用goroutine，避免频繁的创建和销毁
3. 管理任务队列，提供优雅的提交和处理机制
4. 支持优雅关闭，等待所有任务完成

实现方式：
- 使用通道(channel)作为任务队列
- 创建固定数量的worker goroutine处理任务
- 提供提交任务和关闭池的接口

应用场景：
- Web服务器处理大量并发请求
- 批量数据处理
- 需要控制资源使用的高并发应用
- 防止goroutine泄漏

优缺点：
- 优点：控制系统资源使用，提高性能，避免goroutine泄漏
- 缺点：增加代码复杂度，并不是所有场景都需要

以下实现了一个基本的协程池，支持提交任务、关闭池和等待所有任务完成。
*/

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// GoroutineTask 表示要执行的任务
type GoroutineTask func() error

// GoroutinePool 协程池
type GoroutinePool struct {
	workers      int                // 工作协程数量
	taskQueue    chan GoroutineTask // 任务队列
	ctx          context.Context    // 用于控制池生命周期的上下文
	cancel       context.CancelFunc // 取消函数
	wg           sync.WaitGroup     // 等待所有工作协程完成
	running      int32              // 是否正在运行的标志
	taskCount    int32              // 已提交任务数
	errorCount   int32              // 错误任务数
	successCount int32              // 成功任务数
}

// NewGoroutinePool 创建新的协程池
func NewGoroutinePool(workers int, queueSize int) *GoroutinePool {
	if workers <= 0 {
		workers = 1
	}

	if queueSize <= 0 {
		queueSize = 100
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &GoroutinePool{
		workers:   workers,
		taskQueue: make(chan GoroutineTask, queueSize),
		ctx:       ctx,
		cancel:    cancel,
		running:   1, // 初始为运行状态
	}

	// 启动工作协程
	pool.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go pool.worker(i)
	}

	return pool
}

// worker 工作协程主循环
func (p *GoroutinePool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			// 池已关闭，退出
			return
		case task, ok := <-p.taskQueue:
			if !ok {
				// 任务队列已关闭，退出
				return
			}

			// 执行任务
			err := task()
			if err != nil {
				atomic.AddInt32(&p.errorCount, 1)
			} else {
				atomic.AddInt32(&p.successCount, 1)
			}
		}
	}
}

// Submit 提交任务到池
func (p *GoroutinePool) Submit(task GoroutineTask) error {
	if atomic.LoadInt32(&p.running) == 0 {
		return errors.New("协程池已关闭")
	}

	select {
	case <-p.ctx.Done():
		return errors.New("协程池已关闭")
	case p.taskQueue <- task:
		atomic.AddInt32(&p.taskCount, 1)
		return nil
	}
}

// Shutdown 关闭协程池并等待所有任务完成
func (p *GoroutinePool) Shutdown() {
	// 如果已经关闭，直接返回
	if atomic.SwapInt32(&p.running, 0) == 0 {
		return
	}

	// 发送取消信号
	p.cancel()

	// 关闭任务队列
	close(p.taskQueue)

	// 等待所有工作协程退出
	p.wg.Wait()
}

// Stats 返回协程池统计信息
func (p *GoroutinePool) Stats() map[string]interface{} {
	return map[string]interface{}{
		"workers":      p.workers,
		"running":      atomic.LoadInt32(&p.running) == 1,
		"taskCount":    atomic.LoadInt32(&p.taskCount),
		"errorCount":   atomic.LoadInt32(&p.errorCount),
		"successCount": atomic.LoadInt32(&p.successCount),
		"pendingTasks": len(p.taskQueue),
	}
}

// 场景示例：Web服务器请求处理
func GoroutinePoolDemo() {
	// 创建一个有5个工作协程的池，任务队列容量为20
	pool := NewGoroutinePool(5, 20)

	fmt.Println("Web服务器请求处理场景（使用协程池）:")

	// 模拟50个并发请求
	requestCount := 50

	// 创建一个通道用于收集任务执行结果
	results := make(chan string, requestCount)

	// 提交请求处理任务
	for i := 0; i < requestCount; i++ {
		requestID := i

		// 创建并提交任务
		err := pool.Submit(func() error {
			// 模拟请求处理
			processingTime := time.Duration(50+(requestID%100)) * time.Millisecond
			time.Sleep(processingTime)

			// 模拟一些随机失败（每10个请求中有1个失败）
			if requestID%10 == 0 {
				results <- fmt.Sprintf("请求-%d: 失败 (处理时间: %v)", requestID, processingTime)
				return fmt.Errorf("请求-%d 处理失败", requestID)
			}

			// 请求成功
			results <- fmt.Sprintf("请求-%d: 成功 (处理时间: %v)", requestID, processingTime)
			return nil
		})

		if err != nil {
			fmt.Printf("提交任务失败: %v\n", err)
		}
	}

	// 收集前10个结果并显示
	fmt.Println("\n前10个请求处理结果:")
	for i := 0; i < 10; i++ {
		select {
		case result := <-results:
			fmt.Println(result)
		case <-time.After(time.Second):
			fmt.Println("等待处理结果超时")
		}
	}

	// 等待所有请求处理完成
	fmt.Println("\n等待剩余请求处理完成...")
	for i := 0; i < requestCount-10; i++ {
		<-results
	}

	// 显示池统计信息
	stats := pool.Stats()
	fmt.Println("\n协程池统计:")
	fmt.Printf("工作协程: %d\n", stats["workers"])
	fmt.Printf("提交任务总数: %d\n", stats["taskCount"])
	fmt.Printf("成功任务数: %d\n", stats["successCount"])
	fmt.Printf("失败任务数: %d\n", stats["errorCount"])

	// 关闭池
	pool.Shutdown()
	fmt.Println("\n协程池已关闭")
}
