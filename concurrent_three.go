package main

import (
	"fmt"
	"sync"
)

// ThreeThreadsPrint 使用三个线程交替打印
func ThreeThreadsPrint() {
	var wg sync.WaitGroup
	ch1 := make(chan struct{})
	ch2 := make(chan struct{})
	ch3 := make(chan struct{})

	wg.Add(3)

	// 第一个线程打印1,4,7...
	go func() {
		defer wg.Done()
		for i := 1; i <= 97; i += 3 {
			<-ch1 // 等待信号
			fmt.Printf("三线程-1: %d\n", i)
			ch2 <- struct{}{} // 发信号给线程2
		}
	}()

	// 第二个线程打印2,5,8...
	go func() {
		defer wg.Done()
		for i := 2; i <= 98; i += 3 {
			<-ch2 // 等待信号
			fmt.Printf("三线程-2: %d\n", i)
			ch3 <- struct{}{} // 发信号给线程3
		}
	}()

	// 第三个线程打印3,6,9...
	go func() {
		defer wg.Done()
		for i := 3; i <= 99; i += 3 {
			<-ch3 // 等待信号
			fmt.Printf("三线程-3: %d\n", i)
			if i < 99 {
				ch1 <- struct{}{} // 发信号给线程1
			}
		}
	}()

	// 启动第一个线程
	ch1 <- struct{}{}

	wg.Wait()
	fmt.Println("三线程交替打印完成")
}
