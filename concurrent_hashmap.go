package main

import (
	"fmt"
	"sync"
)

// ConcurrentHashMap 是一个线程安全的哈希映射实现
type ConcurrentHashMap struct {
	mu    sync.RWMutex
	items map[string]interface{}
}

// NewConcurrentHashMap 创建一个新的并发哈希映射
func NewConcurrentHashMap() *ConcurrentHashMap {
	return &ConcurrentHashMap{
		items: make(map[string]interface{}),
	}
}

// Set 添加或更新键值对
func (m *ConcurrentHashMap) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key] = value
}

// Get 获取指定键的值
func (m *ConcurrentHashMap) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.items[key]
	return value, exists
}

// Delete 删除指定键值对
func (m *ConcurrentHashMap) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
}

// Size 返回映射大小
func (m *ConcurrentHashMap) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}

// Keys 返回所有键的列表
func (m *ConcurrentHashMap) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.items))
	for k := range m.items {
		keys = append(keys, k)
	}
	return keys
}

// ConcurrentHashMapDemo 演示并发哈希映射的使用
func ConcurrentHashMapDemo() {
	hashMap := NewConcurrentHashMap()
	var wg sync.WaitGroup

	// 并发写入
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id)
			hashMap.Set(key, id*10)
		}(i)
	}

	// 并发读取
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id)
			if val, exists := hashMap.Get(key); exists {
				fmt.Printf("读取: %s = %v\n", key, val)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("最终哈希映射大小: %d\n", hashMap.Size())
	fmt.Printf("所有键: %v\n", hashMap.Keys())
}
