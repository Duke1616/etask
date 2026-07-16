package execution

import (
	"context"
	"sync"
	"time"

	executorv1 "github.com/Duke1616/etask/api/proto/gen/etask/executor/v1"
	"google.golang.org/protobuf/proto"
)

const defaultRetention = 30 * time.Minute

type entry struct {
	state       *executorv1.ExecutionState
	cancel      context.CancelFunc
	completedAt time.Time
}

// Store 并发安全地管理当前节点的执行状态和取消函数。
type Store struct {
	mu        sync.RWMutex
	entries   map[int64]entry
	retention time.Duration
}

// NewStore 创建使用默认终态保存时间的执行状态仓库。
func NewStore() *Store {
	return &Store{entries: make(map[int64]entry), retention: defaultRetention}
}

// Begin 登记一次运行；相同 ID 尚未结束时返回当前状态和 false。
func (s *Store) Begin(state *executorv1.ExecutionState, cancel context.CancelFunc) (*executorv1.ExecutionState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 新执行进入时顺便回收过期终态，避免额外常驻清理协程。
	s.cleanupLocked(time.Now())
	if current, exists := s.entries[state.GetId()]; exists && current.completedAt.IsZero() {
		return clone(current.state), false
	}
	stored := clone(state)
	s.entries[state.GetId()] = entry{state: stored, cancel: cancel}
	return clone(stored), true
}

// Finish 将执行更新为终态并释放取消函数。
func (s *Store) Finish(id int64, status executorv1.ExecutionStatus, result string) (*executorv1.ExecutionState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.entries[id]
	if !exists {
		return nil, false
	}
	current.state.Status = status
	current.state.TaskResult = result
	if status == executorv1.ExecutionStatus_SUCCESS {
		current.state.RunningProgress = 100
	}
	current.cancel = nil
	current.completedAt = time.Now()
	s.entries[id] = current
	return clone(current.state), true
}

// Get 返回执行状态副本，调用方无法修改仓库内部数据。
func (s *Store) Get(id int64) (*executorv1.ExecutionState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	current, exists := s.entries[id]
	if !exists {
		return nil, false
	}
	return clone(current.state), true
}

// Cancel 触发运行中的取消函数并返回触发前的状态副本。
func (s *Store) Cancel(id int64) (*executorv1.ExecutionState, bool) {
	// 锁内只取得状态和取消函数，真正 cancel 放在锁外避免回调阻塞仓库。
	s.mu.RLock()
	current, exists := s.entries[id]
	if !exists || current.cancel == nil {
		s.mu.RUnlock()
		return nil, false
	}
	state := clone(current.state)
	cancel := current.cancel
	s.mu.RUnlock()
	cancel()
	return state, true
}

func (s *Store) cleanupLocked(now time.Time) {
	for id, current := range s.entries {
		if !current.completedAt.IsZero() && now.Sub(current.completedAt) >= s.retention {
			delete(s.entries, id)
		}
	}
}

func clone(state *executorv1.ExecutionState) *executorv1.ExecutionState {
	if state == nil {
		return nil
	}
	return proto.Clone(state).(*executorv1.ExecutionState)
}
