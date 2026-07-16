package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Duke1616/eiam/pkg/ctxutil"
	"github.com/Duke1616/etask/internal/agent/service"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/Duke1616/etask/pkg/mpx"
	"github.com/ecodeclub/mq-api"
	"github.com/gotomicro/ego/core/elog"
)

// ExecuteConsumer 消费 Agent 执行命令并发布结果。
type ExecuteConsumer struct {
	consumer    mq.Consumer
	producer    ExecuteResultProducer
	svc         service.Service
	reg         registry.Registry
	instance    registry.ServiceInstance
	workerCount int
	logger      *elog.Component
}

// NewExecuteConsumer 创建指定 Topic 的 Agent 消费者。
func NewExecuteConsumer(q mq.MQ, svc service.Service, topic string, producer ExecuteResultProducer,
	reg registry.Registry, instance registry.ServiceInstance, workerCount int) (*ExecuteConsumer, error) {
	consumer, err := q.Consumer(topic, "agent-execution-"+topic)
	if err != nil {
		return nil, err
	}
	if workerCount <= 0 {
		workerCount = 1
	}
	return &ExecuteConsumer{
		consumer: consumer, producer: producer, svc: svc, reg: reg, instance: instance,
		workerCount: workerCount,
		logger:      elog.DefaultLogger.With(elog.FieldComponentName("agent.consumer")),
	}, nil
}

// Start 启动 Kafka 数据面，并尽力将 Agent 能力注册到控制面。
func (c *ExecuteConsumer) Start(ctx context.Context) error {
	// Worker 先进入消费状态，避免注册信息可见后出现无人消费的短暂窗口。
	for workerID := 0; workerID < c.workerCount; workerID++ {
		go c.consumeLoop(ctx, workerID)
	}
	// Etcd 只承担能力发现；短暂故障不应关闭已经建立的 Kafka 数据面。
	regCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := c.reg.Register(regCtx, c.instance); err != nil {
		c.logger.Warn("注册 Agent 控制面失败，Kafka 消费继续运行", elog.FieldErr(err))
	}
	return nil
}

func (c *ExecuteConsumer) consumeLoop(ctx context.Context, workerID int) {
	for {
		if err := c.Consume(ctx); err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				return
			}
			c.logger.Error("处理 Agent 命令失败", elog.Int("worker", workerID), elog.FieldErr(err))
		}
	}
}

// Consume 消费并同步处理一条命令。
func (c *ExecuteConsumer) Consume(ctx context.Context) error {
	message, err := c.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取 Agent 执行命令失败: %w", err)
	}
	// 恢复生产端传播的链路和租户上下文，再解析业务命令。
	ctx = mqx.ExtractContext(ctx, message)
	var command ExecuteCommand
	if err = json.Unmarshal(message.Value, &command); err != nil {
		return fmt.Errorf("解析 Agent 执行命令失败: %w", err)
	}
	if err = command.Validate(); err != nil {
		return err
	}
	if ctxutil.GetTenantID(ctx).Int64() == 0 {
		// 传播头缺失时使用已校验的命令租户，保证后续租户插件仍受约束。
		ctx = ctxutil.WithTenantID(ctx, command.TenantID)
		ctx = ctxutil.WithOriginTenantID(ctx, command.TenantID)
	}
	// Agent 同步执行并捕获日志，最终只通过结果 Topic 回传状态。
	output, executeErr := c.svc.Receive(ctx, command.DispatchID, command.Execution())
	status := domain.TaskExecutionStatusSuccess
	result := output.Result
	if executeErr != nil {
		status = domain.TaskExecutionStatusFailed
		if result == "" {
			result = executeErr.Error()
		}
	}
	return c.producer.Produce(ctx, ExecuteResult{
		DispatchID: command.DispatchID,
		State: domain.ExecutionState{
			ID: command.ExecutionID, TaskID: command.TaskID, TaskName: command.TaskName,
			Status: status, ExecutorNodeID: c.instance.ID, TaskResult: result,
		},
		Logs: output.Logs,
	})
}

// Stop 注销 Agent 并关闭消费者。
func (c *ExecuteConsumer) Stop(ctx context.Context) error {
	return errors.Join(
		c.reg.UnRegister(ctx, c.instance),
		c.consumer.Close(),
	)
}
