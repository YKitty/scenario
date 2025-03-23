package concurrency

/*
自定义读写锁（RWMutex）实现

原理：
读写锁是一种特殊的锁，允许多个读取操作同时进行，但写入操作必须独占。
基本原则是"读共享，写独占"，特别适合于读多写少的场景。

关键特点：
1. 允许多个读取者同时持有锁
2. 写入者必须等待所有读取者释放锁
3. 读取者必须等待写入者释放锁
4. 防止写入者饥饿（即优先处理等待的写入者）

实现方式：
- 使用两个锁(读锁和写锁)和计数器跟踪读取者和写入者
- 使用条件变量进行等待和通知

应用场景：
- 并发读取、偶尔写入的数据结构
- 共享配置信息
- 读多写少的缓存系统

优缺点：
- 优点：提高读操作的并发性能
- 缺点：实现复杂，锁升级/降级容易出错

以下实现了一个自定义的读写锁，不依赖于Go的sync.RWMutex。
*/

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// CustomRWMutex 自定义读写锁
type CustomRWMutex struct {
	mu            sync.Mutex // 保护内部状态的互斥锁
	readerCount   int32      // 当前持有读锁的数量
	writerWaiting int32      // 等待写锁的标志（0无等待，1有等待）
	writerActive  int32      // 活跃写锁的标志（0无活跃，1有活跃）

	readerCond *sync.Cond // 读取者条件变量
	writerCond *sync.Cond // 写入者条件变量
}

// NewCustomRWMutex 创建新的自定义读写锁
func NewCustomRWMutex() *CustomRWMutex {
	rw := &CustomRWMutex{}
	rw.readerCond = sync.NewCond(&rw.mu)
	rw.writerCond = sync.NewCond(&rw.mu)
	return rw
}

// RLock 获取读锁
func (rw *CustomRWMutex) RLock() {
	// 先获取互斥锁，以便安全检查和修改内部状态
	rw.mu.Lock()

	// 如果有写入者等待或活跃，读取者需要等待
	// 这样可以防止写入者饥饿
	for atomic.LoadInt32(&rw.writerWaiting) > 0 || atomic.LoadInt32(&rw.writerActive) > 0 {
		rw.readerCond.Wait()
	}

	// 增加读取者计数
	atomic.AddInt32(&rw.readerCount, 1)

	rw.mu.Unlock()
}

// RUnlock 释放读锁
func (rw *CustomRWMutex) RUnlock() {
	rw.mu.Lock()

	// 减少读取者计数
	if atomic.LoadInt32(&rw.readerCount) <= 0 {
		rw.mu.Unlock()
		panic("RUnlock called without a preceding RLock")
	}

	if atomic.AddInt32(&rw.readerCount, -1) == 0 {
		// 如果没有读取者了，通知等待的写入者
		rw.writerCond.Signal()
	}

	rw.mu.Unlock()
}

// Lock 获取写锁
func (rw *CustomRWMutex) Lock() {
	rw.mu.Lock()

	// 标记有写入者等待
	atomic.StoreInt32(&rw.writerWaiting, 1)

	// 等待直到没有读取者和其他写入者
	for atomic.LoadInt32(&rw.readerCount) > 0 || atomic.LoadInt32(&rw.writerActive) > 0 {
		rw.writerCond.Wait()
	}

	// 标记有活跃的写入者，并清除等待标志
	atomic.StoreInt32(&rw.writerActive, 1)
	atomic.StoreInt32(&rw.writerWaiting, 0)

	rw.mu.Unlock()
}

// Unlock 释放写锁
func (rw *CustomRWMutex) Unlock() {
	rw.mu.Lock()

	// 检查是否持有写锁
	if atomic.LoadInt32(&rw.writerActive) == 0 {
		rw.mu.Unlock()
		panic("Unlock called without a preceding Lock")
	}

	// 清除活跃写入者标志
	atomic.StoreInt32(&rw.writerActive, 0)

	// 优先唤醒等待的写入者，否则唤醒所有读取者
	if atomic.LoadInt32(&rw.writerWaiting) > 0 {
		rw.writerCond.Signal()
	} else {
		rw.readerCond.Broadcast()
	}

	rw.mu.Unlock()
}

// 场景示例：共享配置管理
type SharedConfig struct {
	mu   *CustomRWMutex
	data map[string]interface{}
}

// NewSharedConfig 创建新的共享配置
func NewSharedConfig() *SharedConfig {
	return &SharedConfig{
		mu:   NewCustomRWMutex(),
		data: make(map[string]interface{}),
	}
}

// Get 获取配置项
func (c *SharedConfig) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, ok := c.data[key]
	// 模拟一些读取延迟
	time.Sleep(time.Millisecond * 10)
	return value, ok
}

// Set 设置配置项
func (c *SharedConfig) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 模拟一些写入延迟
	time.Sleep(time.Millisecond * 50)
	c.data[key] = value
}

// GetAll 获取所有配置
func (c *SharedConfig) GetAll() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 创建一份拷贝返回
	result := make(map[string]interface{})
	for k, v := range c.data {
		result[k] = v
	}

	// 模拟一些读取延迟
	time.Sleep(time.Millisecond * 20)
	return result
}

// CustomRWMutexDemo 读写锁演示
func CustomRWMutexDemo() {
	config := NewSharedConfig()

	// 初始化一些配置
	config.Set("database.host", "localhost")
	config.Set("database.port", 5432)
	config.Set("cache.enabled", true)

	fmt.Println("共享配置管理场景（使用自定义读写锁）:")

	// 启动10个并发读取者
	var readWg sync.WaitGroup
	readWg.Add(10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer readWg.Done()

			// 每个读取者读取多次
			for j := 0; j < 3; j++ {
				if value, ok := config.Get("database.host"); ok {
					fmt.Printf("读取者 %d: database.host = %v\n", id, value)
				}

				if value, ok := config.Get("cache.enabled"); ok {
					fmt.Printf("读取者 %d: cache.enabled = %v\n", id, value)
				}

				// 短暂停顿
				time.Sleep(time.Millisecond * 5)
			}
		}(i)
	}

	// 启动2个并发写入者
	var writeWg sync.WaitGroup
	writeWg.Add(2)

	for i := 0; i < 2; i++ {
		go func(id int) {
			defer writeWg.Done()

			// 写入者更新配置
			config.Set("log.level", fmt.Sprintf("level-%d", id))
			fmt.Printf("写入者 %d: 设置 log.level = level-%d\n", id, id)

			// 短暂停顿
			time.Sleep(time.Millisecond * 10)

			config.Set("app.version", fmt.Sprintf("1.%d", id))
			fmt.Printf("写入者 %d: 设置 app.version = 1.%d\n", id, id)
		}(i)
	}

	// 等待所有读取者和写入者完成
	writeWg.Wait()
	readWg.Wait()

	// 显示最终配置
	fmt.Println("\n最终配置:")
	for key, value := range config.GetAll() {
		fmt.Printf("%s = %v\n", key, value)
	}
}
