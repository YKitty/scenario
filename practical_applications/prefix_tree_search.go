package practical_applications

/*
前缀树搜索引擎 - 自动完成和搜索提示功能

原理：
前缀树（Trie Tree）又称字典树，是一种有序树形数据结构，用于保存关联数组，
其键通常是字符串。前缀树的优点在于利用字符串的公共前缀来节省存储空间，并且查询效率高。
它的关键特性是：从根节点到特定节点的路径上的字符连接起来，即是该节点对应的字符串。

关键特点：
1. 字符串查询：O(m)时间复杂度，m是字符串长度
2. 节省空间：共享前缀，减少存储冗余
3. 前缀匹配：高效查找具有相同前缀的所有单词
4. 字典序排序：树的遍历天然地按字典序返回单词
5. 可设置权重：为词条添加权重，实现热门推荐

实现方式：
- 每个节点包含多个子节点，对应不同字符
- 使用哈希表或数组存储子节点
- 关键字的终止通过特殊标记表示

应用场景：
- 搜索引擎的自动补全功能
- 拼写检查器
- IP路由表查询
- 单词游戏和文字处理
- 智能输入法的联想功能

优缺点：
- 优点：查找效率高，支持前缀匹配，节省空间
- 缺点：相比哈希表，插入和删除操作较慢，可能占用较多内存

以下实现了一个基本的前缀树搜索引擎，支持单词添加、查找、前缀匹配和自动完成功能。
*/

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// TrieNode 前缀树节点
type TrieNode struct {
	children map[rune]*TrieNode // 子节点
	isEnd    bool               // 是否是单词结尾
	word     string             // 存储完整单词（只在单词结尾节点有效）
	weight   int                // 单词权重/热度
	count    int                // 单词出现次数
}

// Trie 前缀树
type Trie struct {
	root      *TrieNode      // 根节点
	size      int            // 单词数量
	mutex     sync.RWMutex   // 读写锁
	hotWords  map[string]int // 热词表
	timestamp time.Time      // 上次更新时间
}

// Suggestion 搜索建议
type Suggestion struct {
	Word   string // 单词
	Weight int    // 权重
	Count  int    // 出现次数
}

// PrefixSearchEngine 前缀树搜索引擎
type PrefixSearchEngine struct {
	trie              *Trie           // 前缀树
	recentSearches    []string        // 最近搜索
	maxRecentSearches int             // 最大最近搜索数量
	visitLog          map[string]int  // 访问日志
	mutex             sync.RWMutex    // 读写锁
	stopWords         map[string]bool // 停用词
}

// NewTrieNode 创建新的前缀树节点
func NewTrieNode() *TrieNode {
	return &TrieNode{
		children: make(map[rune]*TrieNode),
		isEnd:    false,
		weight:   0,
		count:    0,
	}
}

// NewTrie 创建新的前缀树
func NewTrie() *Trie {
	return &Trie{
		root:      NewTrieNode(),
		size:      0,
		hotWords:  make(map[string]int),
		timestamp: time.Now(),
	}
}

// Insert 插入单词到前缀树
func (t *Trie) Insert(word string, weight int) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// 转换为小写并规范化
	word = normalizeWord(word)
	if word == "" {
		return
	}

	current := t.root
	for _, char := range word {
		if _, exists := current.children[char]; !exists {
			current.children[char] = NewTrieNode()
		}
		current = current.children[char]
	}

	// 如果第一次添加该单词，增加size
	if !current.isEnd {
		t.size++
	}

	current.isEnd = true
	current.word = word
	current.count++

	// 更新权重，取较大值
	if weight > current.weight {
		current.weight = weight
	}

	// 更新热词表
	t.hotWords[word] = current.weight
}

// Search 查找单词是否在前缀树中
func (t *Trie) Search(word string) bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	word = normalizeWord(word)
	if word == "" {
		return false
	}

	node := t.findNode(word)
	return node != nil && node.isEnd
}

// findNode 查找节点
func (t *Trie) findNode(prefix string) *TrieNode {
	current := t.root
	for _, char := range prefix {
		if _, exists := current.children[char]; !exists {
			return nil
		}
		current = current.children[char]
	}
	return current
}

// StartsWith 检查是否有以给定前缀开始的单词
func (t *Trie) StartsWith(prefix string) bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	prefix = normalizeWord(prefix)
	return t.findNode(prefix) != nil
}

// GetByPrefix 获取具有给定前缀的所有单词
func (t *Trie) GetByPrefix(prefix string, limit int) []Suggestion {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	result := make([]Suggestion, 0)
	prefix = normalizeWord(prefix)

	// 找到前缀对应的节点
	node := t.findNode(prefix)
	if node == nil {
		return result
	}

	// 从前缀节点开始深度优先搜索
	t.collectWords(node, &result, limit)

	// 根据权重和计数排序
	sort.Slice(result, func(i, j int) bool {
		if result[i].Weight != result[j].Weight {
			return result[i].Weight > result[j].Weight // 权重高的排前面
		}
		return result[i].Count > result[j].Count // 计数高的排前面
	})

	// 限制结果数量
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result
}

// collectWords 收集所有单词（深度优先搜索）
func (t *Trie) collectWords(node *TrieNode, result *[]Suggestion, limit int) {
	if limit > 0 && len(*result) >= limit {
		return
	}

	if node.isEnd {
		*result = append(*result, Suggestion{
			Word:   node.word,
			Weight: node.weight,
			Count:  node.count,
		})
	}

	for _, child := range node.children {
		t.collectWords(child, result, limit)
	}
}

// Delete 从前缀树中删除单词
func (t *Trie) Delete(word string) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	word = normalizeWord(word)
	if word == "" {
		return false
	}

	return t.deleteWord(t.root, word, 0)
}

// deleteWord 递归删除单词
func (t *Trie) deleteWord(current *TrieNode, word string, index int) bool {
	// 到达单词末尾
	if index == len(word) {
		if !current.isEnd {
			return false // 单词不存在
		}

		current.isEnd = false
		current.word = ""
		t.size--

		// 从热词表移除
		delete(t.hotWords, word)

		return len(current.children) == 0 // 可以删除节点
	}

	char := rune(word[index])
	child, exists := current.children[char]
	if !exists {
		return false // 单词不存在
	}

	shouldDeleteChild := t.deleteWord(child, word, index+1)

	if shouldDeleteChild {
		delete(current.children, char)
		return len(current.children) == 0 && !current.isEnd
	}

	return false
}

// Size 返回前缀树中的单词数量
func (t *Trie) Size() int {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.size
}

// GetHotWords 获取热门单词
func (t *Trie) GetHotWords(limit int) []Suggestion {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	results := make([]Suggestion, 0, len(t.hotWords))
	for word, weight := range t.hotWords {
		node := t.findNode(word)
		if node != nil && node.isEnd {
			results = append(results, Suggestion{
				Word:   word,
				Weight: weight,
				Count:  node.count,
			})
		}
	}

	// 根据权重排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Weight > results[j].Weight
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// NewPrefixSearchEngine 创建新的前缀树搜索引擎
func NewPrefixSearchEngine() *PrefixSearchEngine {
	return &PrefixSearchEngine{
		trie:              NewTrie(),
		recentSearches:    make([]string, 0),
		maxRecentSearches: 10,
		visitLog:          make(map[string]int),
		stopWords:         make(map[string]bool),
	}
}

// AddStopWord 添加停用词
func (e *PrefixSearchEngine) AddStopWord(word string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.stopWords[normalizeWord(word)] = true
}

// IsStopWord 检查是否是停用词
func (e *PrefixSearchEngine) IsStopWord(word string) bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.stopWords[normalizeWord(word)]
}

// AddDocument 添加文档/词条
func (e *PrefixSearchEngine) AddDocument(text string, weight int) {
	words := tokenize(text)

	e.mutex.Lock()
	defer e.mutex.Unlock()

	for _, word := range words {
		// 跳过停用词
		if e.stopWords[word] {
			continue
		}
		e.trie.Insert(word, weight)
	}
}

// Search 执行搜索
func (e *PrefixSearchEngine) Search(query string, limit int) []Suggestion {
	query = normalizeWord(query)
	if query == "" {
		return e.GetHotSearches(limit)
	}

	// 记录搜索
	e.recordSearch(query)

	return e.trie.GetByPrefix(query, limit)
}

// recordSearch 记录搜索词
func (e *PrefixSearchEngine) recordSearch(query string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// 更新访问日志
	e.visitLog[query]++

	// 更新最近搜索
	// 检查是否已存在
	for i, s := range e.recentSearches {
		if s == query {
			// 移到最前面
			e.recentSearches = append(e.recentSearches[:i], e.recentSearches[i+1:]...)
			e.recentSearches = append([]string{query}, e.recentSearches...)
			return
		}
	}

	// 添加到最前面
	e.recentSearches = append([]string{query}, e.recentSearches...)

	// 限制大小
	if len(e.recentSearches) > e.maxRecentSearches {
		e.recentSearches = e.recentSearches[:e.maxRecentSearches]
	}
}

// GetRecentSearches 获取最近搜索
func (e *PrefixSearchEngine) GetRecentSearches() []string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	result := make([]string, len(e.recentSearches))
	copy(result, e.recentSearches)
	return result
}

// GetHotSearches 获取热门搜索
func (e *PrefixSearchEngine) GetHotSearches(limit int) []Suggestion {
	e.mutex.RLock()

	// 收集访问日志
	suggestions := make([]Suggestion, 0, len(e.visitLog))
	for query, count := range e.visitLog {
		if !e.IsStopWord(query) {
			suggestions = append(suggestions, Suggestion{
				Word:   query,
				Count:  count,
				Weight: count, // 使用计数作为权重
			})
		}
	}

	e.mutex.RUnlock()

	// 排序
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Count > suggestions[j].Count
	})

	if limit > 0 && len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions
}

// AutoComplete 自动补全功能
func (e *PrefixSearchEngine) AutoComplete(prefix string, limit int) []Suggestion {
	prefix = normalizeWord(prefix)
	if prefix == "" {
		return e.GetHotSearches(limit)
	}

	// 执行前缀查询
	return e.trie.GetByPrefix(prefix, limit)
}

// Suggest 建议相关搜索
func (e *PrefixSearchEngine) Suggest(query string, limit int) []Suggestion {
	query = normalizeWord(query)

	// 如果输入为空，返回热门搜索
	if query == "" {
		return e.GetHotSearches(limit)
	}

	// 首先尝试精确匹配
	suggestions := e.trie.GetByPrefix(query, limit)

	// 如果精确匹配不足，尝试宽松匹配
	if len(suggestions) < limit {
		words := tokenize(query)
		for _, word := range words {
			if len(word) < 3 || e.IsStopWord(word) {
				continue
			}

			wordSuggestions := e.trie.GetByPrefix(word, limit-len(suggestions))
			suggestions = append(suggestions, wordSuggestions...)

			if len(suggestions) >= limit {
				break
			}
		}
	}

	// 去重
	seen := make(map[string]bool)
	unique := make([]Suggestion, 0, len(suggestions))

	for _, s := range suggestions {
		if !seen[s.Word] {
			seen[s.Word] = true
			unique = append(unique, s)
		}
	}

	return unique
}

// tokenize 将文本分割成词元
func tokenize(text string) []string {
	text = strings.ToLower(text)
	words := make([]string, 0)

	// 简单的按非字母数字字符分割
	var builder strings.Builder
	for _, char := range text {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			builder.WriteRune(char)
		} else {
			if builder.Len() > 0 {
				words = append(words, builder.String())
				builder.Reset()
			}
		}
	}

	if builder.Len() > 0 {
		words = append(words, builder.String())
	}

	return words
}

// normalizeWord 规范化单词（转小写、去除特殊字符）
func normalizeWord(word string) string {
	word = strings.ToLower(strings.TrimSpace(word))
	var builder strings.Builder

	for _, char := range word {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == ' ' {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

// 场景示例：搜索引擎的自动补全
func PrefixTreeSearchDemo() {
	fmt.Println("前缀树搜索引擎示例 - 电商网站搜索:")

	// 创建搜索引擎
	engine := NewPrefixSearchEngine()

	// 添加停用词
	stopWords := []string{"的", "了", "和", "与", "或", "在", "是", "有", "a", "an", "the", "and", "or", "in", "on", "at"}
	for _, word := range stopWords {
		engine.AddStopWord(word)
		fmt.Printf("添加停用词: %s\n", word)
	}

	// 添加商品数据
	products := []struct {
		Name   string
		Weight int
	}{
		{"苹果手机 iPhone 13 Pro 256GB", 90},
		{"苹果手机 iPhone 13 128GB", 85},
		{"苹果手机 iPhone 12 Pro 128GB", 80},
		{"苹果平板 iPad Pro 11英寸", 75},
		{"苹果笔记本电脑 MacBook Pro 14英寸", 85},
		{"华为手机 P40 Pro 5G 256GB", 80},
		{"华为手机 Mate 40 Pro 5G 256GB", 85},
		{"华为平板 MatePad Pro 12.6英寸", 70},
		{"小米手机 11 Ultra 256GB", 75},
		{"小米平板 5 Pro 11英寸", 65},
		{"三星手机 Galaxy S21 Ultra 5G", 75},
		{"三星平板 Galaxy Tab S7+", 65},
		{"笔记本电脑 联想 ThinkPad X1 Carbon", 70},
		{"笔记本电脑 戴尔 XPS 13", 70},
		{"游戏笔记本 华硕 ROG 幻15", 65},
		{"游戏台式机 外星人 Aurora R12", 60},
		{"智能手表 苹果 Apple Watch Series 7", 60},
		{"智能手表 华为 Watch GT 3", 55},
		{"智能手环 小米手环 6", 50},
		{"无线耳机 苹果 AirPods Pro", 65},
		{"无线耳机 索尼 WF-1000XM4", 60},
		{"机械键盘 罗技 G915", 50},
		{"无线鼠标 罗技 MX Master 3", 45},
		{"显示器 三星 奥德赛 G9", 40},
		{"投影仪 极米 H3S", 35},
	}

	fmt.Println("\n添加商品数据:")
	for _, p := range products {
		engine.AddDocument(p.Name, p.Weight)
		fmt.Printf("添加商品: %s (权重: %d)\n", p.Name, p.Weight)
	}

	// 1. 测试自动补全功能
	fmt.Println("\n1. 自动补全功能测试:")

	prefixes := []string{"苹果", "华为", "小米", "笔记本", "手机"}
	for _, prefix := range prefixes {
		fmt.Printf("\n输入: '%s'\n", prefix)
		suggestions := engine.AutoComplete(prefix, 5)
		fmt.Printf("自动补全结果:\n")
		for i, s := range suggestions {
			fmt.Printf("  %d. %s (热度: %d, 计数: %d)\n", i+1, s.Word, s.Weight, s.Count)
		}
	}

	// 2. 模拟用户搜索，记录搜索历史
	fmt.Println("\n2. 模拟用户搜索历史:")

	searches := []string{
		"苹果手机", "华为手机", "小米手机",
		"笔记本电脑", "苹果平板", "游戏本",
		"智能手表", "无线耳机", "苹果手机",
		"显示器", "苹果手机", "华为手机",
	}

	for _, query := range searches {
		fmt.Printf("用户搜索: %s\n", query)
		engine.Search(query, 5)
	}

	// 3. 展示最近搜索
	fmt.Println("\n3. 最近搜索历史:")
	recentSearches := engine.GetRecentSearches()
	for i, s := range recentSearches {
		fmt.Printf("  %d. %s\n", i+1, s)
	}

	// 4. 展示热门搜索
	fmt.Println("\n4. 热门搜索:")
	hotSearches := engine.GetHotSearches(5)
	for i, s := range hotSearches {
		fmt.Printf("  %d. %s (搜索次数: %d)\n", i+1, s.Word, s.Count)
	}

	// 5. 搜索建议功能
	fmt.Println("\n5. 搜索建议功能:")

	queryTests := []string{"手机", "苹果", "游戏", ""}
	for _, query := range queryTests {
		if query == "" {
			fmt.Println("\n空查询 (显示热门推荐):")
		} else {
			fmt.Printf("\n查询: '%s'\n", query)
		}

		suggestions := engine.Suggest(query, 5)
		fmt.Println("建议结果:")
		for i, s := range suggestions {
			fmt.Printf("  %d. %s (相关度: %d)\n", i+1, s.Word, s.Weight)
		}
	}

	// 6. 模拟打字过程中的实时补全
	fmt.Println("\n6. 模拟用户输入过程中的实时补全:")

	typingSequence := []string{"i", "ip", "iph", "ipho", "iphon", "iphone"}
	for _, typed := range typingSequence {
		fmt.Printf("\n用户输入: '%s'\n", typed)
		suggestions := engine.AutoComplete(typed, 3)
		if len(suggestions) > 0 {
			fmt.Println("补全建议:")
			for i, s := range suggestions {
				fmt.Printf("  %d. %s\n", i+1, s.Word)
			}
		} else {
			fmt.Println("没有匹配结果")
		}
	}
}
