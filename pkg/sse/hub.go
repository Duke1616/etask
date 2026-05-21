package sse

import (
	"sync"
)

// Hub 泛型线程安全单事件广播订阅中心
type Hub[T any] struct {
	mu      sync.RWMutex
	clients map[chan T]bool
}

// NewHub 实例化一个新的全局事件 Hub
func NewHub[T any]() *Hub[T] {
	return &Hub[T]{
		clients: make(map[chan T]bool),
	}
}

// Subscribe 订阅状态变更事件，返回一个只读/读写 Channel
func (h *Hub[T]) Subscribe() chan T {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan T, 10)
	h.clients[ch] = true
	return ch
}

// Unsubscribe 取消订阅并安全关闭 Channel，防止泄漏
func (h *Hub[T]) Unsubscribe(ch chan T) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.clients[ch]; exists {
		delete(h.clients, ch)
		close(ch)
	}
}

// Broadcast 广播事件给所有订阅的客户端。
// 使用 select + default 规避慢速连接/满缓冲区造成的整体阻塞。
func (h *Hub[T]) Broadcast(evt T) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- evt:
		default:
			// 慢速客户端/缓冲区满，跳过以避免阻塞主调度或消息消费协程
		}
	}
}
