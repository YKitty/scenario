package main

import (
	"fmt"
	"hash/fnv"
)

// 哈希表节点
type Node struct {
	key   string
	value any
	next  *Node
}

// 哈希表实现
type HashMap struct {
	buckets  []*Node
	size     int
	capacity int
}

// 创建新的哈希表
func NewHashMap() *HashMap {
	capacity := 16 // 初始容量
	return &HashMap{
		buckets:  make([]*Node, capacity),
		size:     0,
		capacity: capacity,
	}
}

// 获取字符串的哈希值
func hash(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// 获取键在桶中的索引
func (h *HashMap) getIndex(key string) int {
	return int(hash(key) % uint32(h.capacity))
}

// 向哈希表中插入键值对
func (h *HashMap) Put(key string, value any) {
	index := h.getIndex(key)

	// 如果桶为空，直接创建新节点
	if h.buckets[index] == nil {
		h.buckets[index] = &Node{key: key, value: value}
		h.size++
		return
	}

	// 遍历链表，如果找到键则更新值，否则添加到链表末尾
	current := h.buckets[index]
	if current.key == key {
		current.value = value
		return
	}

	for current.next != nil {
		current = current.next
		if current.key == key {
			current.value = value
			return
		}
	}

	// 添加到链表末尾
	current.next = &Node{key: key, value: value}
	h.size++

	// 检查是否需要扩容
	if float64(h.size)/float64(h.capacity) > 0.75 {
		h.resize()
	}
}

// 从哈希表中获取值
func (h *HashMap) Get(key string) (any, bool) {
	index := h.getIndex(key)

	current := h.buckets[index]
	for current != nil {
		if current.key == key {
			return current.value, true
		}
		current = current.next
	}

	return nil, false
}

// 从哈希表中删除键值对
func (h *HashMap) Remove(key string) {
	index := h.getIndex(key)

	// 如果桶为空，无需操作
	if h.buckets[index] == nil {
		return
	}

	// 如果是链表头
	if h.buckets[index].key == key {
		h.buckets[index] = h.buckets[index].next
		h.size--
		return
	}

	// 遍历链表寻找要删除的节点
	current := h.buckets[index]
	for current.next != nil {
		if current.next.key == key {
			current.next = current.next.next
			h.size--
			return
		}
		current = current.next
	}
}

// 检查哈希表中是否存在指定的键
func (h *HashMap) Contains(key string) bool {
	_, exists := h.Get(key)
	return exists
}

// 返回哈希表中键值对的数量
func (h *HashMap) Size() int {
	return h.size
}

// 哈希表扩容
func (h *HashMap) resize() {
	oldBuckets := h.buckets
	h.capacity *= 2
	h.buckets = make([]*Node, h.capacity)
	h.size = 0

	// 重新插入所有元素
	for _, bucket := range oldBuckets {
		current := bucket
		for current != nil {
			h.Put(current.key, current.value)
			current = current.next
		}
	}
}

// HashMapDemo 演示哈希表的使用
func HashMapDemo() {
	// 创建一个新的哈希映射
	hashMap := NewHashMap()

	// 测试插入键值对
	hashMap.Put("name", "张三")
	hashMap.Put("age", 25)
	hashMap.Put("email", "zhangsan@example.com")

	// 测试获取值
	if name, exists := hashMap.Get("name"); exists {
		fmt.Printf("姓名: %v\n", name)
	}

	if age, exists := hashMap.Get("age"); exists {
		fmt.Printf("年龄: %v\n", age)
	}

	// 测试检查键是否存在
	fmt.Printf("是否包含'email'键: %v\n", hashMap.Contains("email"))
	fmt.Printf("是否包含'phone'键: %v\n", hashMap.Contains("phone"))

	// 测试获取哈希映射大小
	fmt.Printf("哈希映射大小: %d\n", hashMap.Size())

	// 测试删除键值对
	hashMap.Remove("email")
	fmt.Printf("删除'email'后是否还存在: %v\n", hashMap.Contains("email"))

	// 测试获取哈希映射大小
	fmt.Printf("删除后哈希映射大小: %d\n", hashMap.Size())
}
