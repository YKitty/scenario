package main

import (
	"fmt"
	"sync"
)

// AlternatePrintNumbers 使用两个线程交替打印数字到100
func AlternatePrintNumbers() {
	var wg sync.WaitGroup
	ch1 := make(chan struct{})
	ch2 := make(chan struct{})

	// 添加两个等待组
	wg.Add(2)

	// 第一个线程打印奇数
	go func() {
		defer wg.Done()
		for i := 1; i <= 100; i += 2 {
			<-ch1 // 等待信号
			fmt.Printf("线程1: %d\n", i)
			ch2 <- struct{}{} // 发送信号给线程2
		}
	}()

	// 第二个线程打印偶数
	go func() {
		defer wg.Done()
		for i := 2; i <= 100; i += 2 {
			<-ch2 // 等待信号
			fmt.Printf("线程2: %d\n", i)
			if i < 100 {
				ch1 <- struct{}{} // 发送信号给线程1
			}
		}
	}()

	// 启动第一个线程
	ch1 <- struct{}{}

	// 等待两个线程完成
	wg.Wait()
	fmt.Println("交替打印完成")
}
