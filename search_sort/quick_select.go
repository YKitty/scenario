package search_sort

/*
快速选择算法

原理：
快速选择是一种在未排序数组中找到第k小（或第k大）元素的算法。
它基于快速排序的分区思想，但与快速排序不同的是，快速选择只处理一侧的数据，因此效率更高。

关键特点：
1. 平均时间复杂度为O(n)，最坏情况为O(n²)
2. 不需要完全排序数组，只关注第k小的元素
3. 原地操作，不需要额外的空间
4. 可通过随机化选择pivot来避免最坏情况

实现方式：
- 选择一个pivot元素
- 将数组分区，使得pivot左侧的元素都小于pivot，右侧的元素都大于pivot
- 根据pivot的位置和k的关系，决定继续在哪一侧查找

应用场景：
- 查找数组中的中位数
- 查找第k大/小的元素
- 解决TopK问题
- 数据分析中的百分位数计算

优缺点：
- 优点：比排序后选择更高效
- 缺点：不稳定，最坏情况下可能退化为O(n²)

以下实现了基础的快速选择算法以及一些优化版本。
*/

import (
	"fmt"
	"math/rand"
	"time"
)

// 标准快速选择算法：查找数组中第k小的元素
// k从1开始计数，即k=1表示最小元素，k=len(arr)表示最大元素
func QuickSelect(arr []int, k int) (int, error) {
	if k < 1 || k > len(arr) {
		return 0, fmt.Errorf("k超出范围: %d，数组长度: %d", k, len(arr))
	}

	// 创建副本避免修改原数组
	tmp := make([]int, len(arr))
	copy(tmp, arr)

	// 转换为0-based索引
	kIndex := k - 1

	return quickSelectHelper(tmp, 0, len(tmp)-1, kIndex), nil
}

// 快速选择算法的核心递归函数
func quickSelectHelper(arr []int, left, right, k int) int {
	// 如果数组只包含一个元素，直接返回
	if left == right {
		return arr[left]
	}

	// 选择一个随机pivot并进行分区
	pivotIndex := left + rand.Intn(right-left+1)
	pivotIndex = partitionArray(arr, left, right, pivotIndex)

	// 根据pivot的位置和k的关系，决定在哪一侧继续查找
	if k == pivotIndex {
		return arr[k]
	} else if k < pivotIndex {
		return quickSelectHelper(arr, left, pivotIndex-1, k)
	} else {
		return quickSelectHelper(arr, pivotIndex+1, right, k)
	}
}

// 分区函数，将数组按照pivot分为两部分
func partitionArray(arr []int, left, right, pivotIndex int) int {
	pivotValue := arr[pivotIndex]

	// 将pivot移到最右边
	arr[pivotIndex], arr[right] = arr[right], arr[pivotIndex]

	// 将所有小于pivot的元素移到左边
	storeIndex := left
	for i := left; i < right; i++ {
		if arr[i] < pivotValue {
			arr[storeIndex], arr[i] = arr[i], arr[storeIndex]
			storeIndex++
		}
	}

	// 将pivot放到最终位置
	arr[right], arr[storeIndex] = arr[storeIndex], arr[right]

	return storeIndex
}

// 查找数组中的中位数
func FindMedian(arr []int) (float64, error) {
	n := len(arr)
	if n == 0 {
		return 0, fmt.Errorf("数组为空")
	}

	// 对于奇数长度的数组，中位数是中间的元素
	if n%2 == 1 {
		median, err := QuickSelect(arr, n/2+1)
		return float64(median), err
	}

	// 对于偶数长度的数组，中位数是中间两个元素的平均值
	lower, err1 := QuickSelect(arr, n/2)
	upper, err2 := QuickSelect(arr, n/2+1)

	if err1 != nil {
		return 0, err1
	}
	if err2 != nil {
		return 0, err2
	}

	return float64(lower+upper) / 2.0, nil
}

// BFPRT算法（又称为"中位数的中位数算法"）
// 它是快速选择的优化版本，通过智能选择pivot来确保最坏情况下的时间复杂度为O(n)
func QuickSelectBFPRT(arr []int, k int) (int, error) {
	if k < 1 || k > len(arr) {
		return 0, fmt.Errorf("k超出范围: %d，数组长度: %d", k, len(arr))
	}

	// 创建副本避免修改原数组
	tmp := make([]int, len(arr))
	copy(tmp, arr)

	// 转换为0-based索引
	kIndex := k - 1

	return bfprtHelper(tmp, 0, len(tmp)-1, kIndex), nil
}

// BFPRT算法的辅助函数
func bfprtHelper(arr []int, left, right, k int) int {
	if left == right {
		return arr[left]
	}

	// 通过"中位数的中位数"选择pivot
	pivotIndex := getPivotIndexByBFPRT(arr, left, right)
	pivotIndex = partitionArray(arr, left, right, pivotIndex)

	if k == pivotIndex {
		return arr[k]
	} else if k < pivotIndex {
		return bfprtHelper(arr, left, pivotIndex-1, k)
	} else {
		return bfprtHelper(arr, pivotIndex+1, right, k)
	}
}

// 使用BFPRT方法选择pivot
func getPivotIndexByBFPRT(arr []int, left, right int) int {
	if right-left < 5 {
		return insertionSortAndGetMiddle(arr, left, right)
	}

	// 将数组分为大小为5的组，并找出每组的中位数
	numGroups := (right - left + 1) / 5
	for i := 0; i < numGroups; i++ {
		groupLeft := left + i*5
		groupRight := groupLeft + 4
		if groupRight > right {
			groupRight = right
		}

		// 对每个小组排序并找出中位数
		median := insertionSortAndGetMiddle(arr, groupLeft, groupRight)

		// 将中位数移动到数组的前面部分
		arr[left+i], arr[median] = arr[median], arr[left+i]
	}

	// 递归找出所有中位数的中位数
	mid := left + (numGroups)/2
	return bfprtHelper(arr, left, left+numGroups-1, mid)
}

// 使用插入排序对小数组排序并返回中位数的索引
func insertionSortAndGetMiddle(arr []int, left, right int) int {
	// 对子数组进行插入排序
	for i := left + 1; i <= right; i++ {
		j := i
		for j > left && arr[j-1] > arr[j] {
			arr[j-1], arr[j] = arr[j], arr[j-1]
			j--
		}
	}

	// 返回中位数的索引
	return left + (right-left)/2
}

// 场景示例：在大量访问日志中找出响应时间的中位数
func QuickSelectDemo() {
	fmt.Println("快速选择算法示例 - 响应时间分析:")

	// 模拟API响应时间数据（单位：毫秒）
	rand.Seed(time.Now().UnixNano())
	responseTimes := make([]int, 1000)
	for i := 0; i < len(responseTimes); i++ {
		// 生成一些随机响应时间，大部分在50-150ms之间，但有一些异常值
		base := 50 + rand.Intn(100)
		// 5%的概率生成一个较大的异常值
		if rand.Float64() < 0.05 {
			base += 500 + rand.Intn(1000)
		}
		responseTimes[i] = base
	}

	// 统计函数
	timeFunction := func(name string, f func() interface{}) interface{} {
		start := time.Now()
		result := f()
		fmt.Printf("%s 执行时间: %v\n", name, time.Since(start))
		return result
	}

	// 1. 找出响应时间的中位数
	median, _ := timeFunction("快速选择算法计算中位数", func() interface{} {
		median, err := FindMedian(responseTimes)
		if err != nil {
			return 0.0
		}
		return median
	}).(float64)

	fmt.Printf("API响应时间中位数: %.1f ms\n", median)

	// 2. 找出第90百分位的响应时间（可以表示大多数用户的体验）
	p90Index := int(float64(len(responseTimes)) * 0.9)
	p90, _ := timeFunction("快速选择算法计算P90", func() interface{} {
		p90, err := QuickSelect(responseTimes, p90Index)
		if err != nil {
			return 0
		}
		return p90
	}).(int)

	fmt.Printf("API响应时间P90: %d ms\n", p90)

	// 3. 使用BFPRT算法找出第95百分位的响应时间
	p95Index := int(float64(len(responseTimes)) * 0.95)
	p95, _ := timeFunction("BFPRT算法计算P95", func() interface{} {
		p95, err := QuickSelectBFPRT(responseTimes, p95Index)
		if err != nil {
			return 0
		}
		return p95
	}).(int)

	fmt.Printf("API响应时间P95: %d ms\n", p95)

	// 4. 找出最慢的10个请求的平均响应时间
	slowest10Avg := timeFunction("计算最慢10个请求的平均响应时间", func() interface{} {
		sum := 0
		for i := 0; i < 10; i++ {
			// 找出第i大的元素（即倒数第i+1个）
			val, _ := QuickSelect(responseTimes, len(responseTimes)-i)
			sum += val
		}
		return sum / 10
	}).(int)

	fmt.Printf("最慢10个请求的平均响应时间: %d ms\n", slowest10Avg)

	// 统计分布情况
	fmt.Println("\n响应时间分布:")

	// 创建时间区间
	buckets := []struct {
		min, max int
		count    int
	}{
		{0, 100, 0},
		{101, 200, 0},
		{201, 300, 0},
		{301, 500, 0},
		{501, 1000, 0},
		{1001, -1, 0}, // 超过1000ms的
	}

	// 统计每个区间的请求数
	for _, t := range responseTimes {
		for i := range buckets {
			if buckets[i].max == -1 || (t >= buckets[i].min && t <= buckets[i].max) {
				buckets[i].count++
				break
			}
		}
	}

	// 输出分布情况
	for _, b := range buckets {
		var rangeStr string
		if b.max == -1 {
			rangeStr = fmt.Sprintf("%d+", b.min)
		} else {
			rangeStr = fmt.Sprintf("%d-%d", b.min, b.max)
		}
		percentage := float64(b.count) / float64(len(responseTimes)) * 100
		fmt.Printf("%s ms: %d (%.1f%%)\n", rangeStr, b.count, percentage)
	}
}
