package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Duke1616/etask/internal/agent/domain"
	"github.com/Duke1616/etask/internal/agent/service"
	"github.com/Duke1616/etask/pkg/grpc/registry"
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
	// 1. 服务注册
	regCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	if err := c.reg.Register(regCtx, c.instance); err != nil {
		c.logger.Error("agent_register_failed", elog.FieldErr(err))
	}

	// 2. 启动 worker 协程
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

	// 执行任务，返回 (结构化结果, 原始日志, 状态, 错误)
	structResult, output, status, err := c.svc.Receive(ctx, domain.ExecuteReceive{
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

	// 最终结果提取策略：
	finalResult := structResult
	if finalResult == "" {
		// 降级：如果 FD 3 通道没传数据，则回退到扫描 Stdout 最后一行
		finalResult = c.wantResult(output)
	}

	err = c.producer.Produce(ctx, ExecuteResultEvent{
		TaskId:     evt.TaskId,
		WantResult: finalResult,
		Result:     output,
		Status:     Status(status),
	})

	if err != nil {
		c.logger.Error("发送消息队列失败", elog.Any("错误", err), elog.Any("任务ID", evt.TaskId))
	}

	return err
}

func (c *ExecuteConsumer) wantResult(input string) string {
	input = strings.TrimSpace(input)

	// 1. 彻底为空的兜底
	if input == "" {
		return `{"status": "no_result_detected"}`
	}

	// 2. 尝试判定是否已经是合法的 JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(input), &js); err == nil {
		// 已经是合法 JSON，直接返回
		return input
	}

	// 3. 如果是普通文本，则进行强约束封装
	// 这样可以确保调度端在解析结果时，永远不会因为格式问题报错
	res := map[string]any{
		"status":  "unstructured_fallback",
		"content": input,
	}

	bytes, _ := json.Marshal(res)
	return string(bytes)
}

func (c *ExecuteConsumer) Stop(ctx context.Context) error {
	// 1. 服务注销
	if err := c.reg.UnRegister(ctx, c.instance); err != nil {
		c.logger.Error("agent_unregister_failed", elog.FieldErr(err))
	}

	// 2. 关闭消费者
	return c.consumer.Close()
}
