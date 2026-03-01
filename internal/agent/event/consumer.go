package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Duke1616/ework-runner/internal/agent/domain"
	"github.com/Duke1616/ework-runner/internal/agent/service"
	"github.com/Duke1616/ework-runner/pkg/grpc/registry"
	"github.com/ecodeclub/mq-api"
	"github.com/gotomicro/ego/core/elog"
)

type ExecuteConsumer struct {
	consumer    mq.Consumer
	producer    TaskExecuteResultProducer
	svc         service.Service
	reg         registry.Registry
	instance    registry.ServiceInstance
	workerCount int
	logger      *elog.Component
}

func NewExecuteConsumer(q mq.MQ, svc service.Service, topic string, producer TaskExecuteResultProducer,
	reg registry.Registry, instance registry.ServiceInstance, workerCount int) (*ExecuteConsumer, error) {
	groupID := "task_receive_execute"
	consumer, err := q.Consumer(topic, groupID)
	if err != nil {
		return nil, err
	}

	if workerCount <= 0 {
		workerCount = 1
	}

	return &ExecuteConsumer{
		consumer:    consumer,
		producer:    producer,
		svc:         svc,
		reg:         reg,
		instance:    instance,
		workerCount: workerCount,
		logger:      elog.DefaultLogger.With(elog.FieldComponentName("agent.consumer")),
	}, nil
}

func (c *ExecuteConsumer) Start(ctx context.Context) {
	// 启动 worker 协程
	for i := 0; i < c.workerCount; i++ {
		go func(workerID int) {
			c.logger.Info("启动任务工作协程", elog.Int("worker", workerID))

			for {
				if err := c.Consume(ctx); err != nil {
					// 核心退出判断：如果是 Context 取消，优雅退出循环
					if errors.Is(err, context.Canceled) || ctx.Err() != nil {
						c.logger.Info("工作协程退出", elog.Int("worker", workerID))
						return
					}

					c.logger.Error("处理失败",
						elog.Int("worker", workerID),
						elog.FieldErr(err))
				}
			}
		}(i)
	}
}

func (c *ExecuteConsumer) Consume(ctx context.Context) error {
	cm, err := c.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}
	var evt ExecuteReceive
	if err = json.Unmarshal(cm.Value, &evt); err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	// 封转成 Json 数据
	args, err := json.Marshal(evt.Args)
	if err != nil {
		return err
	}

	c.logger.Info("开始执行任务", elog.Int64("任务ID", evt.TaskId))

	output, status, err := c.svc.Receive(ctx, domain.ExecuteReceive{
		TaskId:    evt.TaskId,
		Language:  evt.Language,
		Handler:   evt.Handler,
		Code:      evt.Code,
		Args:      string(args),
		Variables: evt.Variables,
	})

	if err != nil {
		c.logger.Error("执行任务失败", elog.Any("错误", err), elog.Any("任务ID", evt.TaskId))
	} else {
		c.logger.Info("执行任务完成", elog.Int64("任务ID", evt.TaskId))
	}

	err = c.producer.Produce(ctx, ExecuteResultEvent{
		TaskId:     evt.TaskId,
		WantResult: c.wantResult(output),
		Result:     output,
		Status:     Status(status),
	})

	if err != nil {
		c.logger.Error("发送消息队列失败", elog.Any("错误", err), elog.Any("任务ID", evt.TaskId))
	}

	return err
}

func (c *ExecuteConsumer) wantResult(output string) string {
	outputStr := strings.TrimSpace(output)
	// 检查输出是否为空
	if outputStr == "" {
		c.logger.Error("No output from command.", elog.String("output", output))
		return `{"status": "Error"}`
	}

	// 分割输出为多行并过滤掉空行
	lines := strings.Split(outputStr, "\n")
	var validLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			validLines = append(validLines, line)
		}
	}

	// 检查 validLines 是否为空
	if len(validLines) == 0 {
		c.logger.Error("No valid output lines.", elog.Any("lines", validLines))
		return `{"status": "Error"}`
	}

	// 获取最后一行
	lastLine := validLines[len(validLines)-1]

	return lastLine
}

func (c *ExecuteConsumer) Stop(ctx context.Context) error {
	// 1. 服务注销
	if err := c.reg.UnRegister(ctx, c.instance); err != nil {
		c.logger.Error("agent_unregister_failed", elog.FieldErr(err))
	}

	// 2. 关闭消费者
	return c.consumer.Close()
}
