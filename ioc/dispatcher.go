package ioc

import (
	"github.com/Duke1616/etask/internal/repository"
	"github.com/Duke1616/etask/internal/service/acquirer"
	"github.com/Duke1616/etask/internal/service/dispatcher"
	"github.com/Duke1616/etask/internal/service/invoker"
	"github.com/Duke1616/etask/internal/service/picker"
	"github.com/Duke1616/etask/internal/service/task"
)

// InitDispatcher 创建任务派发器。
func InitDispatcher(
	nodeID string,
	execSvc task.ExecutionService,
	taskAcquirer acquirer.TaskAcquirer,
	invoker invoker.Invoker,
	routes dispatcher.RoutePlanner,
) dispatcher.Dispatcher {
	return dispatcher.NewTaskDispatcher(
		nodeID,
		execSvc,
		taskAcquirer,
		invoker,
		routes,
	)
}

// InitRoutePlanner 创建任务派发路由规划器。
func InitRoutePlanner(
	poolRepo repository.ExecutionPoolRepository,
	targetPicker picker.ExecutorNodePicker,
) dispatcher.RoutePlanner {
	return dispatcher.NewRoutePlanner(poolRepo, targetPicker)
}
