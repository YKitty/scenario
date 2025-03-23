package main

import (
	"fmt"
	"sync"
)

// SpecificRulePrint 一个线程打印3的倍数，另一个打印其他数
func SpecificRulePrint() {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)

	currentNum := 1
	var wg sync.WaitGroup
	wg.Add(2)

	// 线程1打印3的倍数
	go func() {
		defer wg.Done()
		for {
			mu.Lock()
			for currentNum <= 100 && currentNum%3 != 0 {
				cond.Wait() // 等待直到轮到3的倍数
			}

			if currentNum > 100 {
				mu.Unlock()
				break
			}

			fmt.Printf("规则-线程1: %d (3的倍数)\n", currentNum)
			currentNum++
			cond.Broadcast() // 通知所有等待的线程
			mu.Unlock()
		}
	}()

	// 线程2打印非3的倍数
	go func() {
		defer wg.Done()
		for {
			mu.Lock()
			for currentNum <= 100 && currentNum%3 == 0 {
				cond.Wait() // 等待直到轮到非3的倍数
			}

			if currentNum > 100 {
				mu.Unlock()
				break
			}

			fmt.Printf("规则-线程2: %d\n", currentNum)
			currentNum++
			cond.Broadcast() // 通知所有等待的线程
			mu.Unlock()
		}
	}()

	wg.Wait()
	fmt.Println("特定规则打印完成")
}
