package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/ecodeclub/mq-api"
)

// ResultHandler 定义 Agent 结果接入统一执行状态机所需的最小能力。
type ResultHandler interface {
	// FindByID 查询结果对应的执行快照。
	FindByID(ctx context.Context, id int64) (domain.TaskExecution, error)
	// HandleReports 保存日志并处理执行状态。
	HandleReports(ctx context.Context, reports []*domain.Report) error
}

// ResultConsumer 将 Agent 结果接回统一 TaskExecution 状态机。
type ResultConsumer struct {
	executions ResultHandler
	mu         sync.Mutex
	processed  map[string]*resultEntry
}

type resultEntry struct {
	startedAt time.Time
	done      chan struct{}
	err       error
}

// NewResultConsumer 创建 Agent 执行结果消费者。
func NewResultConsumer(executions ResultHandler) *ResultConsumer {
	return &ResultConsumer{executions: executions, processed: make(map[string]*resultEntry)}
}

// Consume 保存日志并处理 Agent 上报的最终状态。
func (c *ResultConsumer) Consume(ctx context.Context, message *mq.Message) error {
	var result Result
	if err := json.Unmarshal(message.Value, &result); err != nil {
		return fmt.Errorf("解析 Agent 执行结果失败: %w", err)
	}
	if result.State.ID <= 0 {
		return fmt.Errorf("Agent 执行结果缺少执行 ID")
	}
	if result.DispatchID == "" {
		return fmt.Errorf("Agent 执行结果缺少派发 ID")
	}
	entry, owner := c.begin(result.DispatchID)
	if !owner {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-entry.done:
			return entry.err
		}
	}
	var handleErr error
	defer func() { c.finish(result.DispatchID, entry, handleErr) }()
	execution, err := c.executions.FindByID(ctx, result.State.ID)
	if err != nil {
		handleErr = err
		return err
	}
	if execution.Status.IsTerminalStatus() {
		return nil
	}
	if err = c.executions.HandleReports(ctx, []*domain.Report{{
		ExecutionState: result.State,
		LogChunks:      result.Logs,
	}}); err != nil {
		handleErr = err
		return err
	}
	return nil
}

func (c *ResultConsumer) begin(dispatchID string) (*resultEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for id, entry := range c.processed {
		select {
		case <-entry.done:
			if now.Sub(entry.startedAt) >= 30*time.Minute {
				delete(c.processed, id)
			}
		default:
		}
	}
	if entry := c.processed[dispatchID]; entry != nil {
		return entry, false
	}
	entry := &resultEntry{startedAt: now, done: make(chan struct{})}
	c.processed[dispatchID] = entry
	return entry, true
}

func (c *ResultConsumer) finish(dispatchID string, entry *resultEntry, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry.err = err
	if err != nil {
		if current := c.processed[dispatchID]; current == entry {
			delete(c.processed, dispatchID)
		}
	}
	close(entry.done)
}
