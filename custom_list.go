package main

/*
自定义双向链表实现

该实现不依赖Go标准库的container/list包，而是从零开始实现了一个完整的双向链表。
提供了常见的链表操作，包括：
- 在头部/尾部添加节点
- 移除指定节点
- 将节点移动到头部/尾部
- 获取头部/尾部节点
- 获取链表长度
*/

// ListNode 双向链表节点
type ListNode struct {
	Value interface{} // 节点值
	prev  *ListNode   // 前一个节点指针
	next  *ListNode   // 后一个节点指针
	list  *List       // 所属链表的引用
}

// Next 返回下一个节点
func (n *ListNode) Next() *ListNode {
	if n.next == n.list.root {
		return nil
	}
	return n.next
}

// Prev 返回前一个节点
func (n *ListNode) Prev() *ListNode {
	if n.prev == n.list.root {
		return nil
	}
	return n.prev
}

// List 双向链表
type List struct {
	root *ListNode // 哨兵节点，root.next指向第一个元素，root.prev指向最后一个元素
	len  int       // 链表长度（不包括哨兵节点）
}

// NewList 创建新的双向链表
func NewList() *List {
	l := new(List)
	l.root = &ListNode{}
	l.root.next = l.root
	l.root.prev = l.root
	l.root.list = l
	l.len = 0
	return l
}

// Len 返回链表长度
func (l *List) Len() int {
	return l.len
}

// Front 返回链表第一个节点，如果链表为空则返回nil
func (l *List) Front() *ListNode {
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

// Back 返回链表最后一个节点，如果链表为空则返回nil
func (l *List) Back() *ListNode {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

// 在节点at之前插入节点
func (l *List) insertBefore(v interface{}, at *ListNode) *ListNode {
	n := &ListNode{
		Value: v,
		prev:  at.prev,
		next:  at,
		list:  l,
	}
	n.prev.next = n
	n.next.prev = n
	l.len++
	return n
}

// 在节点at之后插入节点
func (l *List) insertAfter(v interface{}, at *ListNode) *ListNode {
	n := &ListNode{
		Value: v,
		prev:  at,
		next:  at.next,
		list:  l,
	}
	n.prev.next = n
	n.next.prev = n
	l.len++
	return n
}

// 移除链表中的节点n
func (l *List) remove(n *ListNode) {
	if n.list != l {
		return // 节点不属于该链表
	}
	n.prev.next = n.next
	n.next.prev = n.prev
	n.next = nil // 避免内存泄漏
	n.prev = nil
	n.list = nil
	l.len--
}

// PushFront 在链表头部添加节点
func (l *List) PushFront(v interface{}) *ListNode {
	return l.insertAfter(v, l.root)
}

// PushBack 在链表尾部添加节点
func (l *List) PushBack(v interface{}) *ListNode {
	return l.insertBefore(v, l.root)
}

// Remove 移除链表中的节点n，如果节点不属于该链表则不操作
func (l *List) Remove(n *ListNode) {
	l.remove(n)
}

// MoveToFront 将节点n移动到链表头部
func (l *List) MoveToFront(n *ListNode) {
	if n.list != l || l.root.next == n {
		return
	}
	// 从当前位置删除
	n.prev.next = n.next
	n.next.prev = n.prev

	// 插入到头部
	n.prev = l.root
	n.next = l.root.next
	n.prev.next = n
	n.next.prev = n
}

// MoveToBack 将节点n移动到链表尾部
func (l *List) MoveToBack(n *ListNode) {
	if n.list != l || l.root.prev == n {
		return
	}
	// 从当前位置删除
	n.prev.next = n.next
	n.next.prev = n.prev

	// 插入到尾部
	n.next = l.root
	n.prev = l.root.prev
	n.prev.next = n
	n.next.prev = n
}
