package dispatcher

import (
	"context"

	"github.com/Duke1616/etask/internal/domain"
)

// Dispatcher 定义任务首次派发、重试和重调度能力。
type Dispatcher interface {
	// Run 运行任务
	Run(ctx context.Context, task domain.Task) error
	// Retry 重试任务的一次执行
	Retry(ctx context.Context, execution domain.TaskExecution) error
	// Reschedule 重调度任务的一次执行
	Reschedule(ctx context.Context, execution domain.TaskExecution) error
}
