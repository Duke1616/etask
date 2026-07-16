package event

import (
	"context"

	"github.com/Duke1616/etask/pkg/mpx"
	"github.com/ecodeclub/mq-api"
)

// ExecuteResultProducer 发布 Agent 执行结果。
type ExecuteResultProducer interface {
	// Produce 发布一条带执行 ID 的结果事件。
	Produce(ctx context.Context, result ExecuteResult) error
}

// NewExecuteResultProducer 创建统一结果 Topic 生产者。
func NewExecuteResultProducer(q mq.MQ) (ExecuteResultProducer, error) {
	return mqx.NewGeneralProducer[ExecuteResult](q, ExecuteResultEventName)
}
