package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// AtomicPrint 使用原子操作实现交替打印
func AtomicPrint() {
	var count int32 = 0
	var wg sync.WaitGroup
	var done int32 = 0 // 用于标记任务完成

	wg.Add(2)

	// 线程1打印奇数
	go func() {
		defer wg.Done()
		for {
			// 检查是否已完成
			if atomic.LoadInt32(&done) == 1 {
				break
			}

			// 获取当前计数值
			val := atomic.LoadInt32(&count)
			if val >= 100 {
				atomic.StoreInt32(&done, 1) // 设置完成标志
				break
			}

			// 如果当前值是偶数，尝试增加1使其变为奇数
			if val%2 == 0 {
				if atomic.CompareAndSwapInt32(&count, val, val+1) {
					fmt.Printf("原子-线程1: %d\n", val+1)
					// 如果打印到99，设置完成标志
					if val+1 >= 99 {
						atomic.StoreInt32(&done, 1)
						break
					}
				}
			}

			// 短暂等待，避免CPU过度消耗
			time.Sleep(time.Millisecond)
		}
	}()

	// 线程2打印偶数
	go func() {
		defer wg.Done()
		for {
			// 检查是否已完成
			if atomic.LoadInt32(&done) == 1 {
				break
			}

			// 获取当前计数值
			val := atomic.LoadInt32(&count)
			if val >= 99 {
				atomic.StoreInt32(&done, 1) // 设置完成标志
				break
			}

			// 如果当前值是奇数，尝试增加1使其变为偶数
			if val%2 == 1 {
				if atomic.CompareAndSwapInt32(&count, val, val+1) {
					fmt.Printf("原子-线程2: %d\n", val+1)
					// 如果打印到100，设置完成标志
					if val+1 >= 100 {
						atomic.StoreInt32(&done, 1)
						break
					}
				}
			}

			// 短暂等待，避免CPU过度消耗
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()
	fmt.Println("原子操作交替打印完成")
}
