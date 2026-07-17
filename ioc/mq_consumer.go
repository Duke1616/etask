package ioc

import (
	"context"
	"fmt"

	"github.com/Duke1616/etask/internal/event/complete"
	executionevent "github.com/Duke1616/etask/internal/event/execution"
	"github.com/Duke1616/etask/internal/service/acquirer"
	"github.com/Duke1616/etask/internal/service/task"
	internalSSE "github.com/Duke1616/etask/internal/sse"
	mqx "github.com/Duke1616/etask/pkg/mpx"
	"github.com/ecodeclub/mq-api"
)

func InitCompleteEventConsumer(q mq.MQ,
	taskSvc task.Service,
	execSvc task.ExecutionService,
	acquire acquirer.TaskAcquirer,
	events *internalSSE.Hubs,
) *CompleteConsumer {
	topic := "complete_topic"
	group := "reporter"
	con := mqx.NewConsumer(name(topic, group), q, topic)
	comConsumer := complete.NewConsumer(execSvc, taskSvc, acquire, events)
	return &CompleteConsumer{
		com:      con,
		Consumer: comConsumer,
	}
}

// InitAgentResultConsumer 创建调度侧 Agent 结果消费者。
func InitAgentResultConsumer(q mq.MQ, executions task.ExecutionService) *AgentResultConsumer {
	consumer := mqx.NewConsumer(name(executionevent.ResultTopic, "scheduler"), q, executionevent.ResultTopic)
	return &AgentResultConsumer{consumer: consumer, handler: executionevent.NewResultConsumer(executions)}
}

// AgentResultConsumer 负责启动 Agent 结果消费循环。
type AgentResultConsumer struct {
	consumer *mqx.Consumer
	handler  *executionevent.ResultConsumer
}

// Start 启动 Agent 结果消费循环。
func (c *AgentResultConsumer) Start(ctx context.Context) {
	if err := c.consumer.Start(ctx, c.handler.Consume); err != nil {
		panic(err)
	}
}

type CompleteConsumer struct {
	*complete.Consumer
	com *mqx.Consumer
}

func (c *CompleteConsumer) Start(ctx context.Context) {
	err := c.com.Start(ctx, c.Consume)
	if err != nil {
		panic(err)
	}
}

func name(eventName, group string) string {
	return fmt.Sprintf("%s-%s", eventName, group)
}
