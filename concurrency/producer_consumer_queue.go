package concurrency

/*
生产者-消费者队列

原理：
生产者-消费者模式是一种并发设计模式，它将任务的生产和消费解耦，通过共享队列在二者之间传递数据。
生产者负责创建数据并放入队列，消费者负责从队列取出数据并处理。

关键特点：
1. 生产者和消费者可以以不同的速率工作
2. 队列作为缓冲区，平衡生产和消费的速率
3. 支持阻塞操作（队列满时生产者阻塞，队列空时消费者阻塞）
4. 支持多个生产者和多个消费者

实现方式：
- 使用通道(channel)作为共享队列
- 使用互斥锁和条件变量实现阻塞行为
- 提供优雅关闭机制

应用场景：
- 并发数据处理
- 消息队列系统
- 任务调度系统
- 日志收集和处理

优缺点：
- 优点：解耦任务生产和消费，提高并发处理能力
- 缺点：需要额外的同步机制，可能增加复杂性

以下实现了一个线程安全的生产者-消费者队列，支持阻塞操作和优雅关闭。
*/

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// 错误定义
var (
	ErrQueueClosed = errors.New("队列已关闭")
	ErrQueueFull   = errors.New("队列已满")
)

// BoundedQueue 有界队列，支持生产者-消费者模式
type BoundedQueue struct {
	items        []interface{} // 队列项
	capacity     int           // 队列容量
	head         int           // 队列头索引
	tail         int           // 队列尾索引
	count        int           // 队列中的项数
	mu           sync.Mutex    // 互斥锁
	notEmpty     *sync.Cond    // 非空条件变量
	notFull      *sync.Cond    // 非满条件变量
	closed       int32         // 关闭标志
	enqueueCount int64         // 入队计数
	dequeueCount int64         // 出队计数
}

// NewBoundedQueue 创建新的有界队列
func NewBoundedQueue(capacity int) *BoundedQueue {
	if capacity <= 0 {
		capacity = 10
	}

	q := &BoundedQueue{
		items:    make([]interface{}, capacity),
		capacity: capacity,
		head:     0,
		tail:     0,
		count:    0,
		closed:   0,
	}

	q.notEmpty = sync.NewCond(&q.mu)
	q.notFull = sync.NewCond(&q.mu)

	return q
}

// Enqueue 将项添加到队列，如果队列已满则阻塞
func (q *BoundedQueue) Enqueue(item interface{}) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 检查队列是否已关闭
	if atomic.LoadInt32(&q.closed) != 0 {
		return ErrQueueClosed
	}

	// 等待直到队列非满或关闭
	for q.count == q.capacity && atomic.LoadInt32(&q.closed) == 0 {
		q.notFull.Wait()
	}

	// 再次检查队列是否已关闭（等待期间可能已关闭）
	if atomic.LoadInt32(&q.closed) != 0 {
		return ErrQueueClosed
	}

	// 添加项到队尾
	q.items[q.tail] = item
	q.tail = (q.tail + 1) % q.capacity
	q.count++

	// 增加入队计数
	atomic.AddInt64(&q.enqueueCount, 1)

	// 通知等待的消费者
	q.notEmpty.Signal()

	return nil
}

// EnqueueWithTimeout 将项添加到队列，如果队列已满则在超时后返回错误
func (q *BoundedQueue) EnqueueWithTimeout(item interface{}, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// 创建一个完成通道
	done := make(chan struct{})

	// 使用goroutine尝试入队
	var enqueueErr error
	go func() {
		enqueueErr = q.Enqueue(item)
		close(done)
	}()

	// 等待入队完成或超时
	select {
	case <-done:
		return enqueueErr
	case <-timer.C:
		// 超时，但goroutine可能仍在尝试入队，无法取消
		// 如果后续入队成功，数据会被加入队列，这是预期行为
		return ErrQueueFull
	}
}

// Dequeue 从队列中取出项，如果队列为空则阻塞
func (q *BoundedQueue) Dequeue() (interface{}, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 等待直到队列非空或关闭
	for q.count == 0 && atomic.LoadInt32(&q.closed) == 0 {
		q.notEmpty.Wait()
	}

	// 如果队列为空且已关闭，返回错误
	if q.count == 0 && atomic.LoadInt32(&q.closed) != 0 {
		return nil, ErrQueueClosed
	}

	// 从队头取出项
	item := q.items[q.head]
	q.items[q.head] = nil // 避免内存泄漏
	q.head = (q.head + 1) % q.capacity
	q.count--

	// 增加出队计数
	atomic.AddInt64(&q.dequeueCount, 1)

	// 通知等待的生产者
	q.notFull.Signal()

	return item, nil
}

// DequeueWithTimeout 从队列中取出项，如果队列为空则在超时后返回错误
func (q *BoundedQueue) DequeueWithTimeout(timeout time.Duration) (interface{}, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// 创建一个完成通道
	done := make(chan struct{})

	// 使用goroutine尝试出队
	var item interface{}
	var dequeueErr error
	go func() {
		item, dequeueErr = q.Dequeue()
		close(done)
	}()

	// 等待出队完成或超时
	select {
	case <-done:
		return item, dequeueErr
	case <-timer.C:
		return nil, errors.New("出队超时")
	}
}

// Close 关闭队列，阻止进一步入队，允许已入队的项被出队
func (q *BoundedQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if atomic.SwapInt32(&q.closed, 1) == 0 {
		// 通知所有等待的生产者和消费者
		q.notFull.Broadcast()
		q.notEmpty.Broadcast()
	}
}

// Size 返回队列中的项数
func (q *BoundedQueue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.count
}

// Capacity 返回队列容量
func (q *BoundedQueue) Capacity() int {
	return q.capacity
}

// IsClosed 返回队列是否已关闭
func (q *BoundedQueue) IsClosed() bool {
	return atomic.LoadInt32(&q.closed) != 0
}

// Stats 返回队列的统计信息
func (q *BoundedQueue) Stats() map[string]interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	return map[string]interface{}{
		"capacity":     q.capacity,
		"size":         q.count,
		"enqueueCount": atomic.LoadInt64(&q.enqueueCount),
		"dequeueCount": atomic.LoadInt64(&q.dequeueCount),
		"closed":       atomic.LoadInt32(&q.closed) != 0,
	}
}

// 场景示例：日志收集系统
func ProducerConsumerDemo() {
	// 创建一个容量为5的有界队列
	queue := NewBoundedQueue(5)

	fmt.Println("日志收集系统场景（生产者-消费者模式）:")

	// 创建停止信号
	stop := make(chan struct{})

	// 启动3个生产者协程
	var producerWg sync.WaitGroup
	producerWg.Add(3)

	for i := 0; i < 3; i++ {
		go func(id int) {
			defer producerWg.Done()

			generator := func() string {
				return fmt.Sprintf("日志-生产者%d-%d", id, time.Now().UnixNano())
			}

			// 每个生产者产生10个日志条目
			for j := 0; j < 10; j++ {
				select {
				case <-stop:
					return
				default:
					log := generator()
					err := queue.EnqueueWithTimeout(log, time.Second)
					if err != nil {
						fmt.Printf("生产者%d: 入队失败: %v\n", id, err)
					} else {
						fmt.Printf("生产者%d: 产生日志 %s\n", id, log)
					}

					// 生产者以不同速率工作
					time.Sleep(time.Duration(50*(id+1)) * time.Millisecond)
				}
			}
		}(i)
	}

	// 启动2个消费者协程
	var consumerWg sync.WaitGroup
	consumerWg.Add(2)

	for i := 0; i < 2; i++ {
		go func(id int) {
			defer consumerWg.Done()

			// 消费者处理日志直到队列关闭且为空
			for {
				// 尝试从队列获取日志条目
				log, err := queue.DequeueWithTimeout(time.Millisecond * 500)

				if err != nil {
					if err == ErrQueueClosed && queue.Size() == 0 {
						fmt.Printf("消费者%d: 队列已关闭并为空，退出\n", id)
						return
					}
					// 其他错误（如超时）继续尝试
					continue
				}

				// 处理日志
				fmt.Printf("消费者%d: 处理日志 %s\n", id, log)

				// 消费者以不同速率工作
				time.Sleep(time.Duration(100*(id+1)) * time.Millisecond)
			}
		}(i)
	}

	// 等待生产者完成
	producerWg.Wait()
	fmt.Println("\n所有生产者完成生产")

	// 关闭队列，不再接受新的日志
	queue.Close()
	fmt.Println("队列已关闭，不再接受新日志")

	// 等待消费者处理完所有日志
	consumerWg.Wait()
	fmt.Println("所有消费者已完成处理")

	// 显示队列统计信息
	stats := queue.Stats()
	fmt.Println("\n队列统计:")
	fmt.Printf("容量: %d\n", stats["capacity"])
	fmt.Printf("最终大小: %d\n", stats["size"])
	fmt.Printf("总入队数: %d\n", stats["enqueueCount"])
	fmt.Printf("总出队数: %d\n", stats["dequeueCount"])
}
