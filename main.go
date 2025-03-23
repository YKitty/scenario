package main

import (
	"fmt"
)

func main() {
	// 运行哈希表演示
	HashMapDemo()

	// 选择要运行的演示
	runDemo()
}

// 提供选择运行不同的演示
func runDemo() {
	fmt.Println("\n请选择要运行的演示:")
	fmt.Println("1. 原始Channel实现交替打印")
	fmt.Println("2. 互斥锁和条件变量实现交替打印")
	fmt.Println("3. 三线程交替打印")
	fmt.Println("4. 原子操作实现交替打印")
	fmt.Println("5. 特定规则交替打印")
	fmt.Println("6. 并发哈希映射演示")
	fmt.Println("7. LRU缓存演示 (标准库实现)")
	fmt.Println("8. LFU缓存演示 (标准库实现)")
	fmt.Println("9. LRU缓存演示 (自定义链表实现)")
	fmt.Println("10. LFU缓存演示 (自定义链表实现)")

	var choice int
	fmt.Print("请输入选择 (1-10): ")
	fmt.Scan(&choice)

	fmt.Println("\n--- 开始演示 ---")
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
	case 7:
		LRUCacheDemo()
	case 8:
		LFUCacheDemo()
	case 9:
		CustomLRUCacheDemo()
	case 10:
		CustomLFUCacheDemo()
	default:
		fmt.Println("无效选择，默认运行哈希表演示")
		HashMapDemo()
	}
}
