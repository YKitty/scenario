package search_sort

/*
外部排序

原理：
外部排序是一种处理超出内存容量的大量数据的排序技术。它将大数据集分割成多个小块，
对每个小块单独排序，然后将排序好的小块合并成最终排序结果。

关键特点：
1. 适用于处理无法一次性加载到内存中的大数据集
2. 通常包括两个阶段：分割-排序阶段和归并阶段
3. 使用文件系统存储中间结果
4. 可以通过多路归并提高效率

实现方式：
- 将大文件分割成多个可以装入内存的小文件
- 使用内部排序算法对每个小文件进行排序
- 将排序后的小文件通过多路归并算法合并
- 使用缓冲区减少I/O操作次数

应用场景：
- 大型数据库的排序操作
- 日志文件的处理和分析
- 大数据集的预处理
- 海量文件系统中的文件排序

优缺点：
- 优点：能够处理超大数据集，内存使用量可控
- 缺点：I/O操作较多，性能受磁盘速度限制

以下实现了一个简化的外部排序算法，用于对大型整数文件进行排序。
*/

import (
	"bufio"
	"container/heap"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 用于多路归并的优先队列项
type heapItem struct {
	value     int            // 当前值
	chunkID   int            // 源块ID
	nextValue *int           // 下一个值（预读）
	scanner   *bufio.Scanner // 扫描器，用于读取更多数据
}

// 用于优先队列的堆接口实现
type minHeap []*heapItem

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].value < h[j].value }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *minHeap) Push(x interface{}) {
	*h = append(*h, x.(*heapItem))
}

func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // 避免内存泄漏
	*h = old[0 : n-1]
	return item
}

// ExternalSort 外部排序函数
// 输入: 大文件路径，内存限制（每个块的最大行数），临时目录
// 输出: 排序后的文件路径
func ExternalSort(inputFile string, maxLinesPerChunk int, tempDir string) (string, error) {
	// 1. 分割-排序阶段: 将大文件分割成多个小块并分别排序
	chunkFiles, err := splitAndSort(inputFile, maxLinesPerChunk, tempDir)
	if err != nil {
		return "", fmt.Errorf("分割排序阶段失败: %v", err)
	}

	// 2. 归并阶段: 将排序好的小块合并成最终结果
	outputFile := filepath.Join(tempDir, "sorted_output.txt")
	err = mergeChunks(chunkFiles, outputFile)
	if err != nil {
		return "", fmt.Errorf("归并阶段失败: %v", err)
	}

	// 3. 删除临时文件
	for _, file := range chunkFiles {
		os.Remove(file)
	}

	return outputFile, nil
}

// 分割大文件并对每个小块排序
func splitAndSort(inputFile string, maxLinesPerChunk int, tempDir string) ([]string, error) {
	// 打开输入文件
	file, err := os.Open(inputFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var chunkFiles []string
	var lines []int
	chunkID := 0

	// 扫描文件中的每一行
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// 将字符串转换为整数
		num, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err != nil {
			continue // 忽略无效行
		}

		lines = append(lines, num)

		// 当达到块大小时，对当前块排序并写入磁盘
		if len(lines) >= maxLinesPerChunk {
			chunkFile, err := sortAndWriteChunk(lines, chunkID, tempDir)
			if err != nil {
				return chunkFiles, err
			}
			chunkFiles = append(chunkFiles, chunkFile)
			chunkID++
			lines = nil // 清空当前块
		}
	}

	// 处理最后一个不完整的块
	if len(lines) > 0 {
		chunkFile, err := sortAndWriteChunk(lines, chunkID, tempDir)
		if err != nil {
			return chunkFiles, err
		}
		chunkFiles = append(chunkFiles, chunkFile)
	}

	if err := scanner.Err(); err != nil {
		return chunkFiles, err
	}

	return chunkFiles, nil
}

// 对一个块进行排序并写入磁盘
func sortAndWriteChunk(lines []int, chunkID int, tempDir string) (string, error) {
	// 对块内数据排序
	sort.Ints(lines)

	// 创建输出文件
	chunkFile := filepath.Join(tempDir, fmt.Sprintf("chunk_%d.txt", chunkID))
	outFile, err := os.Create(chunkFile)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	// 将排序后的数据写入文件
	for _, num := range lines {
		fmt.Fprintf(outFile, "%d\n", num)
	}

	return chunkFile, nil
}

// 合并多个排序好的块
func mergeChunks(chunkFiles []string, outputFile string) error {
	if len(chunkFiles) == 0 {
		return nil
	}

	// 打开所有输入文件
	scanners := make([]*bufio.Scanner, len(chunkFiles))
	for i, file := range chunkFiles {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		scanners[i] = bufio.NewScanner(f)
	}

	// 创建输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 创建一个缓冲写入器以提高性能
	writer := bufio.NewWriter(outFile)
	defer writer.Flush()

	// 创建优先队列用于多路归并
	h := &minHeap{}
	heap.Init(h)

	// 从每个块中读取第一个元素
	for i, scanner := range scanners {
		if scanner.Scan() {
			// 读取第一个值
			val, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
			if err != nil {
				continue
			}

			// 预读下一个值
			var nextVal *int
			if scanner.Scan() {
				nv, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
				if err == nil {
					nextVal = &nv
				}
			}

			// 添加到堆中
			heap.Push(h, &heapItem{
				value:     val,
				chunkID:   i,
				nextValue: nextVal,
				scanner:   scanner,
			})
		}
	}

	// 开始多路归并
	for h.Len() > 0 {
		// 获取最小元素
		item := heap.Pop(h).(*heapItem)

		// 将当前最小值写入输出文件
		fmt.Fprintf(writer, "%d\n", item.value)

		// 如果已经预读了下一个值，则直接使用
		if item.nextValue != nil {
			item.value = *item.nextValue
			item.nextValue = nil

			// 继续读取下一个值作为预读
			if item.scanner.Scan() {
				nv, err := strconv.Atoi(strings.TrimSpace(item.scanner.Text()))
				if err == nil {
					item.nextValue = &nv
				}
			}

			// 将更新后的项放回堆中
			heap.Push(h, item)
		} else if item.scanner.Scan() {
			// 如果没有预读，尝试读取下一个值
			val, err := strconv.Atoi(strings.TrimSpace(item.scanner.Text()))
			if err != nil {
				continue
			}

			item.value = val

			// 预读下一个值
			if item.scanner.Scan() {
				nv, err := strconv.Atoi(strings.TrimSpace(item.scanner.Text()))
				if err == nil {
					item.nextValue = &nv
				}
			}

			// 将更新后的项放回堆中
			heap.Push(h, item)
		}
		// 如果没有更多数据，则此块已处理完毕
	}

	return nil
}

// GenerateTestFile 生成用于测试的大型整数文件
func GenerateTestFile(filename string, numLines int, maxValue int) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	defer writer.Flush()

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < numLines; i++ {
		num := rand.Intn(maxValue)
		fmt.Fprintf(writer, "%d\n", num)
	}

	return nil
}

// VerifySortedFile 验证文件是否已排序
func VerifySortedFile(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var prev int
	isFirst := true

	for scanner.Scan() {
		num, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err != nil {
			continue
		}

		if isFirst {
			prev = num
			isFirst = false
			continue
		}

		if num < prev {
			return false, nil
		}

		prev = num
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return true, nil
}

// 输出排序后文件的部分内容
func outputPreview(filename string, lines int) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0

	for scanner.Scan() && count < lines {
		fmt.Println(scanner.Text())
		count++
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// 场景示例：对大型日志文件中的时间戳进行排序
func ExternalSortDemo() {
	fmt.Println("外部排序示例 - 对大型日志文件中的时间戳进行排序:")

	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "external_sort")
	if err != nil {
		fmt.Printf("创建临时目录失败: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir)

	// 生成测试输入文件（模拟时间戳）
	inputFile := filepath.Join(tempDir, "timestamps.txt")
	numLines := 100000
	fmt.Printf("生成测试输入文件，包含 %d 个随机时间戳...\n", numLines)

	err = GenerateTestFile(inputFile, numLines, 1000000)
	if err != nil {
		fmt.Printf("生成测试文件失败: %v\n", err)
		return
	}

	// 配置排序参数
	maxLinesPerChunk := 10000 // 每个块最多10000行

	// 执行外部排序
	fmt.Printf("开始排序，每个内存块包含 %d 行...\n", maxLinesPerChunk)
	startTime := time.Now()

	outputFile, err := ExternalSort(inputFile, maxLinesPerChunk, tempDir)
	if err != nil {
		fmt.Printf("排序失败: %v\n", err)
		return
	}

	duration := time.Since(startTime)
	fmt.Printf("排序完成，耗时: %v\n", duration)

	// 验证排序结果
	fmt.Println("验证排序结果...")
	isSorted, err := VerifySortedFile(outputFile)
	if err != nil {
		fmt.Printf("验证失败: %v\n", err)
		return
	}

	if isSorted {
		fmt.Println("验证成功: 文件已正确排序!")
	} else {
		fmt.Println("验证失败: 文件未正确排序")
	}

	// 统计排序前后文件大小
	inputInfo, _ := os.Stat(inputFile)
	outputInfo, _ := os.Stat(outputFile)

	fmt.Printf("\n排序统计信息:\n")
	fmt.Printf("源文件大小: %.2f MB\n", float64(inputInfo.Size())/(1024*1024))
	fmt.Printf("结果文件大小: %.2f MB\n", float64(outputInfo.Size())/(1024*1024))
	fmt.Printf("每秒处理: %.2f MB\n", float64(inputInfo.Size())/(1024*1024)/duration.Seconds())

	// 输出排序后文件的部分内容
	fmt.Println("\n排序后文件的前10行:")
	outputPreview(outputFile, 10)
}
