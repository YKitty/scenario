package main

import (
	"fmt"
)

func main() {
	// 运行哈希表演示
	HashMapDemo()

	// 选择要运行的并发测试
	runConcurrentTest()
}

// 提供选择运行不同的并发测试
func runConcurrentTest() {
	fmt.Println("\n请选择要运行的并发测试:")
	fmt.Println("1. 原始Channel实现")
	fmt.Println("2. 互斥锁和条件变量实现")
	fmt.Println("3. 三线程交替打印")
	fmt.Println("4. 原子操作实现")
	fmt.Println("5. 特定规则实现")
	fmt.Println("6. 并发哈希映射演示")

	var choice int
	fmt.Print("请输入选择 (1-6): ")
	fmt.Scan(&choice)

	fmt.Println("\n--- 开始测试 ---")
	switch choice {
	case 1:
		AlternatePrintNumbers()
	case 2:
		AlternatePrintWithMutex()
	case 3:
		ThreeThreadsPrint()
	case 4:
		AtomicPrint()
	case 5:
		SpecificRulePrint()
	case 6:
		ConcurrentHashMapDemo()
	default:
		fmt.Println("无效选择，默认运行哈希表演示")
		HashMapDemo()
	}
}
