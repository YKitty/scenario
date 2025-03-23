package practical_applications

/*
基于跳表的键值存储 - 高效有序数据结构

原理：
跳表(Skip List)是一种随机化的数据结构，基于并联的有序链表，用于快速查找、插入和删除操作。
它通过维护多层索引来加速搜索，平均时间复杂度为O(log n)，与平衡树相当，但实现更为简单。
Redis的有序集合(Sorted Set)就使用跳表作为底层实现之一。

关键特点：
1. 分层结构：由多层链表组成，底层包含所有元素，上层为索引
2. 随机化：元素提升到上层索引的概率通常为1/2
3. 查找效率：平均O(log n)，最坏O(n)但概率极低
4. 空间占用：平均每个元素占用约O(1)的额外索引空间
5. 有序性：支持范围查询等有序操作

实现方式：
- 使用多层链表实现，每层链表是前一层的子集
- 使用随机函数决定元素在哪一层出现
- 提供插入、删除、查找和范围查询操作

应用场景：
- 键值存储数据库
- 内存数据库的有序索引
- 范围查询频繁的应用
- 实时排行榜系统
- 作为平衡树的替代结构

优缺点：
- 优点：实现相对简单，查询性能稳定，内存占用可控
- 缺点：不保证严格平衡，最坏情况性能降低

以下实现了一个基于跳表的键值存储，支持常见的键值操作和范围查询。
*/

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	MaxLevel    = 32  // 跳表最大层数
	Probability = 0.5 // 元素提升到上一层的概率
)

var (
	ErrKeyNotFound = errors.New("键不存在")
	ErrKeyExists   = errors.New("键已存在")
)

// Element 跳表节点元素
type Element struct {
	Key   []byte     // 键
	Value []byte     // 值
	Score float64    // 分数（用于排序）
	Next  []*Element // 指向每一层的下一个元素
	Prev  *Element   // 指向前一个元素（仅在第0层）
}

// SkipList 跳表结构
type SkipList struct {
	head    *Element     // 头节点
	tail    *Element     // 尾节点
	length  int          // 元素数量
	level   int          // 当前最大层数
	randSrc *rand.Rand   // 随机数源
	mutex   sync.RWMutex // 读写锁
}

// SkiplistKVStore 基于跳表的键值存储
type SkiplistKVStore struct {
	data     *SkipList            // 跳表数据结构
	mutex    sync.RWMutex         // 读写锁
	ttlData  map[string]time.Time // TTL数据
	ttlMutex sync.RWMutex         // TTL读写锁
	stopCh   chan struct{}        // 停止清理协程的通道
}

// NewElement 创建新的跳表元素
func NewElement(key, value []byte, score float64, level int) *Element {
	return &Element{
		Key:   key,
		Value: value,
		Score: score,
		Next:  make([]*Element, level),
		Prev:  nil,
	}
}

// NewSkipList 创建新的跳表
func NewSkipList() *SkipList {
	head := NewElement(nil, nil, -1, MaxLevel)

	return &SkipList{
		head:    head,
		tail:    nil,
		length:  0,
		level:   1,
		randSrc: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// randomLevel 随机生成层数
func (sl *SkipList) randomLevel() int {
	level := 1
	for level < MaxLevel && sl.randSrc.Float64() < Probability {
		level++
	}
	return level
}

// Insert 插入元素到跳表
func (sl *SkipList) Insert(key, value []byte, score float64) *Element {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	// 查找插入位置
	update := make([]*Element, MaxLevel)
	x := sl.head

	for i := sl.level - 1; i >= 0; i-- {
		for x.Next[i] != nil && (x.Next[i].Score < score ||
			(x.Next[i].Score == score && bytes.Compare(x.Next[i].Key, key) < 0)) {
			x = x.Next[i]
		}
		update[i] = x
	}

	// 如果键已存在，直接更新值
	if x.Next[0] != nil && x.Next[0].Score == score && bytes.Equal(x.Next[0].Key, key) {
		x.Next[0].Value = value
		return x.Next[0]
	}

	// 随机生成新节点的层数
	level := sl.randomLevel()
	if level > sl.level {
		for i := sl.level; i < level; i++ {
			update[i] = sl.head
		}
		sl.level = level
	}

	// 创建新节点
	newElement := NewElement(key, value, score, level)

	// 更新所有相关节点的指针
	for i := 0; i < level; i++ {
		newElement.Next[i] = update[i].Next[i]
		update[i].Next[i] = newElement
	}

	// 更新前向指针（仅在第0层）
	if update[0] != sl.head {
		newElement.Prev = update[0]
	}

	if newElement.Next[0] != nil {
		newElement.Next[0].Prev = newElement
	} else {
		sl.tail = newElement
	}

	sl.length++
	return newElement
}

// Delete 从跳表删除元素
func (sl *SkipList) Delete(key []byte, score float64) bool {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	// 查找删除位置
	update := make([]*Element, MaxLevel)
	x := sl.head

	for i := sl.level - 1; i >= 0; i-- {
		for x.Next[i] != nil && (x.Next[i].Score < score ||
			(x.Next[i].Score == score && bytes.Compare(x.Next[i].Key, key) < 0)) {
			x = x.Next[i]
		}
		update[i] = x
	}

	// 找到目标节点
	x = x.Next[0]

	if x == nil || x.Score != score || !bytes.Equal(x.Key, key) {
		return false // 节点不存在
	}

	// 更新指针，删除节点
	for i := 0; i < sl.level; i++ {
		if update[i].Next[i] != x {
			break
		}
		update[i].Next[i] = x.Next[i]
	}

	// 更新前向指针
	if x.Next[0] != nil {
		x.Next[0].Prev = x.Prev
	} else {
		sl.tail = x.Prev
	}

	// 更新跳表层数
	for sl.level > 1 && sl.head.Next[sl.level-1] == nil {
		sl.level--
	}

	sl.length--
	return true
}

// Search 在跳表中查找元素
func (sl *SkipList) Search(key []byte, score float64) *Element {
	sl.mutex.RLock()
	defer sl.mutex.RUnlock()

	x := sl.head

	for i := sl.level - 1; i >= 0; i-- {
		for x.Next[i] != nil && (x.Next[i].Score < score ||
			(x.Next[i].Score == score && bytes.Compare(x.Next[i].Key, key) < 0)) {
			x = x.Next[i]
		}
	}

	x = x.Next[0]

	if x != nil && x.Score == score && bytes.Equal(x.Key, key) {
		return x
	}

	return nil
}

// Range 范围查询，返回指定分数范围内的所有元素
func (sl *SkipList) Range(minScore, maxScore float64, limit int) []*Element {
	sl.mutex.RLock()
	defer sl.mutex.RUnlock()

	result := make([]*Element, 0)

	// 从头节点开始查找起始位置
	x := sl.head

	for i := sl.level - 1; i >= 0; i-- {
		for x.Next[i] != nil && x.Next[i].Score < minScore {
			x = x.Next[i]
		}
	}

	// 找到第一个大于等于minScore的节点
	x = x.Next[0]

	// 遍历范围内的所有节点
	for x != nil && x.Score <= maxScore {
		result = append(result, x)

		if limit > 0 && len(result) >= limit {
			break
		}

		x = x.Next[0]
	}

	return result
}

// Length 返回跳表元素数量
func (sl *SkipList) Length() int {
	sl.mutex.RLock()
	defer sl.mutex.RUnlock()
	return sl.length
}

// First 返回第一个元素
func (sl *SkipList) First() *Element {
	sl.mutex.RLock()
	defer sl.mutex.RUnlock()
	return sl.head.Next[0]
}

// Last 返回最后一个元素
func (sl *SkipList) Last() *Element {
	sl.mutex.RLock()
	defer sl.mutex.RUnlock()
	return sl.tail
}

// NewSkiplistKVStore 创建新的基于跳表的键值存储
func NewSkiplistKVStore() *SkiplistKVStore {
	store := &SkiplistKVStore{
		data:    NewSkipList(),
		ttlData: make(map[string]time.Time),
		stopCh:  make(chan struct{}),
	}

	// 启动TTL清理协程
	go store.ttlCleaner()

	return store
}

// ttlCleaner 定期清理过期数据
func (s *SkiplistKVStore) ttlCleaner() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanExpiredKeys()
		case <-s.stopCh:
			return
		}
	}
}

// cleanExpiredKeys 清理过期的键
func (s *SkiplistKVStore) cleanExpiredKeys() {
	now := time.Now()
	expiredKeys := make([]string, 0)

	// 找出所有过期的键
	s.ttlMutex.RLock()
	for key, expiry := range s.ttlData {
		if expiry.Before(now) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	s.ttlMutex.RUnlock()

	// 删除过期的键
	for _, key := range expiredKeys {
		s.Delete([]byte(key))
	}
}

// Set 设置键值对
func (s *SkiplistKVStore) Set(key, value []byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 使用键的哈希值作为分数，确保唯一性
	score := float64(hashBytes(key))
	s.data.Insert(key, value, score)

	// 删除可能存在的TTL
	s.ttlMutex.Lock()
	delete(s.ttlData, string(key))
	s.ttlMutex.Unlock()
}

// SetWithTTL 设置带过期时间的键值对
func (s *SkiplistKVStore) SetWithTTL(key, value []byte, ttl time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	score := float64(hashBytes(key))
	s.data.Insert(key, value, score)

	// 设置TTL
	s.ttlMutex.Lock()
	s.ttlData[string(key)] = time.Now().Add(ttl)
	s.ttlMutex.Unlock()
}

// Get 获取键对应的值
func (s *SkiplistKVStore) Get(key []byte) ([]byte, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// 检查键是否过期
	s.ttlMutex.RLock()
	if expiry, exists := s.ttlData[string(key)]; exists && time.Now().After(expiry) {
		s.ttlMutex.RUnlock()
		// 懒惰删除
		go s.Delete(key)
		return nil, ErrKeyNotFound
	}
	s.ttlMutex.RUnlock()

	score := float64(hashBytes(key))
	elem := s.data.Search(key, score)

	if elem == nil {
		return nil, ErrKeyNotFound
	}

	return elem.Value, nil
}

// Delete 删除键
func (s *SkiplistKVStore) Delete(key []byte) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	score := float64(hashBytes(key))
	result := s.data.Delete(key, score)

	// 删除TTL
	s.ttlMutex.Lock()
	delete(s.ttlData, string(key))
	s.ttlMutex.Unlock()

	return result
}

// GetTTL 获取键的剩余过期时间
func (s *SkiplistKVStore) GetTTL(key []byte) (time.Duration, bool) {
	s.ttlMutex.RLock()
	defer s.ttlMutex.RUnlock()

	expiry, exists := s.ttlData[string(key)]
	if !exists {
		return 0, false
	}

	remaining := time.Until(expiry)
	if remaining <= 0 {
		return 0, false
	}

	return remaining, true
}

// Keys 获取所有键
func (s *SkiplistKVStore) Keys() [][]byte {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	keys := make([][]byte, 0, s.data.Length())
	current := s.data.First()

	for current != nil {
		// 检查是否过期
		s.ttlMutex.RLock()
		if expiry, exists := s.ttlData[string(current.Key)]; !exists || time.Now().Before(expiry) {
			keys = append(keys, current.Key)
		}
		s.ttlMutex.RUnlock()

		current = current.Next[0]
	}

	return keys
}

// Size 返回存储中的键数量
func (s *SkiplistKVStore) Size() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// 不考虑TTL，返回跳表中的元素数量
	return s.data.Length()
}

// SizeActive 返回未过期的键数量
func (s *SkiplistKVStore) SizeActive() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	count := 0
	current := s.data.First()
	now := time.Now()

	for current != nil {
		// 检查是否过期
		s.ttlMutex.RLock()
		if expiry, exists := s.ttlData[string(current.Key)]; !exists || now.Before(expiry) {
			count++
		}
		s.ttlMutex.RUnlock()

		current = current.Next[0]
	}

	return count
}

// Close 关闭存储
func (s *SkiplistKVStore) Close() {
	close(s.stopCh) // 停止TTL清理协程
}

// Scan 范围扫描
func (s *SkiplistKVStore) Scan(prefix []byte, limit int) map[string][]byte {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make(map[string][]byte)
	count := 0
	current := s.data.First()
	now := time.Now()

	for current != nil && (limit <= 0 || count < limit) {
		// 检查前缀匹配
		if bytes.HasPrefix(current.Key, prefix) {
			// 检查是否过期
			s.ttlMutex.RLock()
			if expiry, exists := s.ttlData[string(current.Key)]; !exists || now.Before(expiry) {
				result[string(current.Key)] = current.Value
				count++
			}
			s.ttlMutex.RUnlock()
		}

		current = current.Next[0]
	}

	return result
}

// 计算字节数组的哈希值
func hashBytes(data []byte) uint64 {
	var hash uint64 = 14695981039346656037 // FNV-1a 哈希初始值

	for _, b := range data {
		hash ^= uint64(b)
		hash *= 1099511628211 // FNV-1a 质数
	}

	return hash
}

// 展示跳表的可视化结构（用于调试）
func (sl *SkipList) visualize() string {
	sl.mutex.RLock()
	defer sl.mutex.RUnlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("SkipList(level=%d, length=%d):\n", sl.level, sl.length))

	// 可视化每一层
	for i := sl.level - 1; i >= 0; i-- {
		sb.WriteString(fmt.Sprintf("Level %d: ", i))
		node := sl.head.Next[i]
		for node != nil {
			sb.WriteString(fmt.Sprintf("[%.2f] -> ", node.Score))
			node = node.Next[i]
		}
		sb.WriteString("nil\n")
	}

	return sb.String()
}

// 场景示例：排行榜系统
func SkiplistKVStoreDemo() {
	fmt.Println("基于跳表的键值存储示例 - 游戏排行榜系统:")

	// 创建键值存储
	store := NewSkiplistKVStore()
	defer store.Close()

	// 模拟游戏玩家数据
	players := []struct {
		ID    string
		Name  string
		Score int
	}{
		{"player:1001", "张三", 8750},
		{"player:1002", "李四", 9320},
		{"player:1003", "王五", 7600},
		{"player:1004", "赵六", 9100},
		{"player:1005", "孙七", 8900},
		{"player:1006", "周八", 7200},
		{"player:1007", "吴九", 9500},
		{"player:1008", "郑十", 8300},
	}

	// 1. 存储玩家数据
	fmt.Println("\n1. 添加玩家数据:")
	for _, p := range players {
		key := []byte(p.ID)
		value := []byte(fmt.Sprintf("%s|%d", p.Name, p.Score))
		store.Set(key, value)
		fmt.Printf("添加玩家: %s, 分数: %d\n", p.Name, p.Score)
	}

	// 2. 构建排行榜
	fmt.Println("\n2. 查看当前排行榜:")
	buildLeaderboard(store)

	// 3. 更新部分玩家分数（加入TTL）
	fmt.Println("\n3. 更新部分玩家分数:")
	scoreUpdates := map[string]int{
		"player:1001": 9100, // 张三获得更高分数
		"player:1003": 8200, // 王五获得更高分数
		"player:1006": 9800, // 周八获得最高分数
	}

	for id, newScore := range scoreUpdates {
		key := []byte(id)
		// 先获取旧数据
		oldData, err := store.Get(key)
		if err != nil {
			fmt.Printf("获取玩家 %s 失败: %v\n", id, err)
			continue
		}

		parts := strings.Split(string(oldData), "|")
		name := parts[0]

		// 更新数据，并加入7天TTL（模拟一周内有效的分数）
		value := []byte(fmt.Sprintf("%s|%d", name, newScore))
		store.SetWithTTL(key, value, 7*24*time.Hour)
		fmt.Printf("更新玩家: %s, 新分数: %d（有效期7天）\n", name, newScore)
	}

	// 4. 更新后的排行榜
	fmt.Println("\n4. 更新后的排行榜:")
	buildLeaderboard(store)

	// 5. 模拟一个玩家分数过期
	fmt.Println("\n5. 模拟玩家数据过期:")
	// 设置一个马上过期的玩家
	expiringPlayer := "player:1002" // 李四
	oldData, _ := store.Get([]byte(expiringPlayer))
	parts := strings.Split(string(oldData), "|")
	name := parts[0]

	fmt.Printf("设置玩家 %s 的数据过期（1秒后）\n", name)
	store.SetWithTTL([]byte(expiringPlayer), oldData, 1*time.Second)

	// 等待数据过期
	fmt.Println("等待1秒钟...")
	time.Sleep(1500 * time.Millisecond)

	// 6. 玩家过期后的排行榜
	fmt.Println("\n6. 玩家数据过期后的排行榜:")
	buildLeaderboard(store)

	// 7. 按前缀查询（例如查找所有玩家）
	fmt.Println("\n7. 按前缀查询所有玩家:")
	allPlayers := store.Scan([]byte("player:"), 0)
	fmt.Printf("共找到 %d 个玩家\n", len(allPlayers))
	for k, v := range allPlayers {
		fmt.Printf("  %s: %s\n", k, string(v))
	}

	// 8. 存储统计
	fmt.Println("\n8. 存储统计:")
	fmt.Printf("总键数量: %d\n", store.Size())
	fmt.Printf("活跃键数量: %d\n", store.SizeActive())

	// 9. 查看跳表内部结构
	skipList := store.data
	fmt.Println("\n9. 跳表内部结构:")
	fmt.Printf("跳表层数: %d\n", skipList.level)
	fmt.Printf("跳表元素数量: %d\n", skipList.Length())

	// 10. 示范基于跳表的范围查询能力
	fmt.Println("\n10. 范围查询示例 (比如查询分数在8500-9500之间的玩家):")
	fmt.Println("注意：实际应用中需要将玩家分数作为跳表的分数字段，这里只是演示")
	fmt.Println("在真实应用中，我们会使用专门的排序键或独立的跳表索引")
}

// 构建并显示排行榜
func buildLeaderboard(store *SkiplistKVStore) {
	// 获取所有玩家数据
	keys := store.Keys()

	// 解析并排序
	type PlayerScore struct {
		ID    string
		Name  string
		Score int
	}

	players := make([]PlayerScore, 0, len(keys))

	for _, key := range keys {
		if !bytes.HasPrefix(key, []byte("player:")) {
			continue
		}

		data, err := store.Get(key)
		if err != nil {
			continue
		}

		parts := strings.Split(string(data), "|")
		if len(parts) != 2 {
			continue
		}

		name := parts[0]
		var score int
		fmt.Sscanf(parts[1], "%d", &score)

		players = append(players, PlayerScore{
			ID:    string(key),
			Name:  name,
			Score: score,
		})
	}

	// 按分数排序（从高到低）
	sort.Slice(players, func(i, j int) bool {
		return players[i].Score > players[j].Score
	})

	// 显示排行榜
	fmt.Println("排行榜（按分数从高到低）:")
	for i, p := range players {
		fmt.Printf("  第%d名: %s - %d分\n", i+1, p.Name, p.Score)
	}
}
