package main

import (
	"fmt"
	"sync"
)

// AlternatePrintWithMutex 使用互斥锁实现交替打印
func AlternatePrintWithMutex() {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	turn := 1 // 1代表线程1的回合，2代表线程2的回合

	var wg sync.WaitGroup
	wg.Add(2)

	// 线程1打印奇数
	go func() {
		defer wg.Done()
		for i := 1; i <= 100; i += 2 {
			mu.Lock()
			for turn != 1 {
				cond.Wait() // 等待直到轮到线程1
			}
			fmt.Printf("互斥锁-线程1: %d\n", i)
			turn = 2      // 下一回合轮到线程2
			cond.Signal() // 通知等待的线程
			mu.Unlock()
		}
	}()

	// 线程2打印偶数
	go func() {
		defer wg.Done()
		for i := 2; i <= 100; i += 2 {
			mu.Lock()
			for turn != 2 {
				cond.Wait() // 等待直到轮到线程2
			}
			fmt.Printf("互斥锁-线程2: %d\n", i)
			turn = 1      // 下一回合轮到线程1
			cond.Signal() // 通知等待的线程
			mu.Unlock()
		}
	}()

	wg.Wait()
	fmt.Println("互斥锁实现完成")
}
