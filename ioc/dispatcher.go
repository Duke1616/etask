package ioc

import (
	"github.com/Duke1616/etask/internal/event"
	"github.com/Duke1616/etask/internal/service/acquirer"
	"github.com/Duke1616/etask/internal/service/dispatcher"
	"github.com/Duke1616/etask/internal/service/invoker"
	"github.com/Duke1616/etask/internal/service/task"
)

func InitDispatcher(
	nodeID string,
	taskSvc task.Service,
	execSvc task.ExecutionService,
	taskAcquirer acquirer.TaskAcquirer,
	invoker invoker.Invoker,
	producer event.CompleteProducer,
) dispatcher.Dispatcher {
	return dispatcher.NewTaskDispatcher(
		nodeID,
		taskSvc,
		execSvc,
		taskAcquirer,
		invoker,
		producer,
	)
}
