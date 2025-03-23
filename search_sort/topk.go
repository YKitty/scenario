package search_sort

/*
TopK 问题

原理：
TopK 问题是指在一组数据中找出最大或最小的 K 个元素。
这类问题在大数据处理、信息检索、推荐系统等领域有广泛应用。

关键特点：
1. 通常处理大量数据的子集选择
2. 关注排序或选择的效率，避免全量排序
3. 有多种实现方法，如堆、快速选择、桶排序等
4. 可以处理静态数据或数据流

实现方式：
- 堆方法：维护一个K大小的小顶堆（求最大K个）或大顶堆（求最小K个）
- 快速选择：类似快速排序的分区思想，但只处理一侧的数据
- 计数排序：适用于有限范围的整数

应用场景：
- 搜索引擎返回最相关的K条结果
- 推荐系统筛选最匹配的K个推荐项
- 实时数据分析中的热点项统计
- 大数据集中的异常检测

优缺点：
- 优点：避免完全排序，提高效率
- 缺点：根据实现方式不同，有不同的局限性

以下实现了多种方法解决TopK问题，包括堆方法、快速选择法和桶排序法。
*/

import (
	"container/heap"
	"fmt"
	"math/rand"
	"sort"
	"time"
)

// 使用最小堆实现的TopK（找最大的K个元素）
type MinHeapTopK struct {
	data []int // 存储数据的堆
	k    int   // 保留的元素个数
}

// 初始化一个容量为k的最小堆
func NewMinHeapTopK(k int) *MinHeapTopK {
	return &MinHeapTopK{
		data: make([]int, 0, k),
		k:    k,
	}
}

// 添加元素并维护堆结构
func (h *MinHeapTopK) Add(num int) {
	if len(h.data) < h.k {
		// 堆还未满，直接添加
		h.data = append(h.data, num)
		h.siftUp(len(h.data) - 1)
	} else if num > h.data[0] {
		// 堆已满且当前元素大于堆顶（最小元素），替换堆顶
		h.data[0] = num
		h.siftDown(0)
	}
}

// 上浮操作
func (h *MinHeapTopK) siftUp(i int) {
	for {
		parent := (i - 1) / 2
		if parent < 0 || h.data[parent] <= h.data[i] {
			break
		}
		h.data[parent], h.data[i] = h.data[i], h.data[parent]
		i = parent
	}
}

// 下沉操作
func (h *MinHeapTopK) siftDown(i int) {
	n := len(h.data)
	for {
		smallest := i
		left := 2*i + 1
		right := 2*i + 2

		if left < n && h.data[left] < h.data[smallest] {
			smallest = left
		}

		if right < n && h.data[right] < h.data[smallest] {
			smallest = right
		}

		if smallest == i {
			break
		}

		h.data[i], h.data[smallest] = h.data[smallest], h.data[i]
		i = smallest
	}
}

// 获取TopK结果并排序
func (h *MinHeapTopK) Result() []int {
	result := make([]int, len(h.data))
	copy(result, h.data)
	sort.Sort(sort.Reverse(sort.IntSlice(result))) // 从大到小排序
	return result
}

// 使用标准库堆接口实现的TopK
type IntHeap []int

func (h IntHeap) Len() int           { return len(h) }
func (h IntHeap) Less(i, j int) bool { return h[i] < h[j] } // 小顶堆
func (h IntHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *IntHeap) Push(x interface{}) {
	*h = append(*h, x.(int))
}

func (h *IntHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// 使用标准库堆实现的TopK
func FindTopKWithHeap(nums []int, k int) []int {
	if k <= 0 || len(nums) == 0 {
		return []int{}
	}

	if k >= len(nums) {
		result := make([]int, len(nums))
		copy(result, nums)
		sort.Sort(sort.Reverse(sort.IntSlice(result)))
		return result
	}

	h := &IntHeap{}
	heap.Init(h)

	// 填充前k个元素
	for i := 0; i < k; i++ {
		heap.Push(h, nums[i])
	}

	// 处理剩余元素
	for i := k; i < len(nums); i++ {
		if nums[i] > (*h)[0] {
			heap.Pop(h)
			heap.Push(h, nums[i])
		}
	}

	// 转换为数组并排序
	result := make([]int, k)
	for i := k - 1; i >= 0; i-- {
		result[i] = heap.Pop(h).(int)
	}

	return result
}

// 使用快速选择算法（类似快速排序）实现的TopK
func FindTopKWithQuickSelect(nums []int, k int) []int {
	if k <= 0 || len(nums) == 0 {
		return []int{}
	}

	if k >= len(nums) {
		result := make([]int, len(nums))
		copy(result, nums)
		sort.Sort(sort.Reverse(sort.IntSlice(result)))
		return result
	}

	// 创建一个副本以避免修改原数组
	numsCopy := make([]int, len(nums))
	copy(numsCopy, nums)

	// 找到第k大的元素的索引位置
	quickSelect(numsCopy, 0, len(numsCopy)-1, len(numsCopy)-k)

	// 返回前k大的元素（排序后）
	result := numsCopy[len(numsCopy)-k:]
	sort.Sort(sort.Reverse(sort.IntSlice(result)))
	return result
}

// 快速选择算法核心函数
func quickSelect(nums []int, left, right, kSmallest int) {
	if left == right {
		return
	}

	// 随机选择pivot，减少最坏情况发生的概率
	pivotIndex := left + rand.Intn(right-left+1)
	pivotIndex = partition(nums, left, right, pivotIndex)

	// 根据pivot的位置决定继续处理哪部分
	if pivotIndex == kSmallest {
		return
	} else if pivotIndex < kSmallest {
		quickSelect(nums, pivotIndex+1, right, kSmallest)
	} else {
		quickSelect(nums, left, pivotIndex-1, kSmallest)
	}
}

// 分区函数，返回pivot的最终位置
func partition(nums []int, left, right, pivotIndex int) int {
	pivotValue := nums[pivotIndex]

	// 将pivot移到最右边
	nums[pivotIndex], nums[right] = nums[right], nums[pivotIndex]

	// 将所有小于pivot的元素移到左边
	storeIndex := left
	for i := left; i < right; i++ {
		if nums[i] < pivotValue {
			nums[storeIndex], nums[i] = nums[i], nums[storeIndex]
			storeIndex++
		}
	}

	// 将pivot放到最终位置
	nums[right], nums[storeIndex] = nums[storeIndex], nums[right]

	return storeIndex
}

// 使用桶排序实现的TopK（适用于有限范围的整数）
func FindTopKWithBucketSort(nums []int, k int, maxVal int) []int {
	if k <= 0 || len(nums) == 0 {
		return []int{}
	}

	if k >= len(nums) {
		result := make([]int, len(nums))
		copy(result, nums)
		sort.Sort(sort.Reverse(sort.IntSlice(result)))
		return result
	}

	// 创建计数桶
	buckets := make([]int, maxVal+1)
	for _, num := range nums {
		buckets[num]++
	}

	// 从大到小收集结果
	result := make([]int, 0, k)
	for i := maxVal; i >= 0 && len(result) < k; i-- {
		for j := 0; j < buckets[i] && len(result) < k; j++ {
			result = append(result, i)
		}
	}

	return result
}

// 计算运行时间的辅助函数
func timeFunction(name string, f func()) {
	start := time.Now()
	f()
	duration := time.Since(start)
	fmt.Printf("%s 执行时间: %v\n", name, duration)
}

// 场景示例：网站最热门文章排行
func TopKDemo() {
	fmt.Println("TopK问题示例 - 网站热门文章排行榜:")

	// 模拟文章ID和其访问量
	type Article struct {
		ID        int
		Title     string
		ViewCount int
	}

	// 生成模拟数据
	rand.Seed(time.Now().UnixNano())
	articles := make([]Article, 1000)
	viewCounts := make([]int, 1000)

	for i := 0; i < 1000; i++ {
		viewCount := rand.Intn(10000)
		articles[i] = Article{
			ID:        i + 1,
			Title:     fmt.Sprintf("文章 #%d", i+1),
			ViewCount: viewCount,
		}
		viewCounts[i] = viewCount
	}

	// 查找访问量最高的前10篇文章
	k := 10

	// 方法1: 使用自定义实现的最小堆
	var topK1 []int
	timeFunction("自定义最小堆", func() {
		minHeap := NewMinHeapTopK(k)
		for _, count := range viewCounts {
			minHeap.Add(count)
		}
		topK1 = minHeap.Result()
	})

	// 方法2: 使用标准库的堆实现
	var topK2 []int
	timeFunction("标准库堆实现", func() {
		topK2 = FindTopKWithHeap(viewCounts, k)
	})

	// 方法3: 使用快速选择算法
	var topK3 []int
	timeFunction("快速选择算法", func() {
		topK3 = FindTopKWithQuickSelect(viewCounts, k)
	})

	// 方法4: 使用桶排序（适用于已知范围的整数）
	var topK4 []int
	timeFunction("桶排序", func() {
		topK4 = FindTopKWithBucketSort(viewCounts, k, 10000)
	})

	// 验证结果是否一致
	isEqual := func(a, b []int) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	fmt.Println("\n所有方法的结果一致性验证:")
	fmt.Printf("自定义堆 vs 标准库堆: %v\n", isEqual(topK1, topK2))
	fmt.Printf("自定义堆 vs 快速选择: %v\n", isEqual(topK1, topK3))
	fmt.Printf("自定义堆 vs 桶排序: %v\n", isEqual(topK1, topK4))

	// 找出对应的文章信息
	fmt.Println("\n访问量最高的10篇文章:")

	// 创建访问量到文章的映射
	viewToArticle := make(map[int][]Article)
	for _, article := range articles {
		viewToArticle[article.ViewCount] = append(viewToArticle[article.ViewCount], article)
	}

	// 输出结果
	for i, count := range topK1 {
		fmt.Printf("%d. ", i+1)
		for _, article := range viewToArticle[count] {
			fmt.Printf("文章ID: %d, 标题: %s, 访问量: %d\n", article.ID, article.Title, article.ViewCount)
		}
	}
}
