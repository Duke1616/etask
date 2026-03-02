package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Duke1616/etask/internal/agent/domain"
	"github.com/Duke1616/etask/internal/grpc/scripts"
	"github.com/Duke1616/etask/sdk/executor"
	"github.com/ecodeclub/mq-api"
	"github.com/gotomicro/ego/core/elog"
)

type Service interface {
	// Receive 接收任务并执行
	// 返回值: (结构化结果, 原始日志, 状态, 错误)
	Receive(ctx context.Context, req domain.ExecuteReceive) (string, string, domain.Status, error)

	// ListHandlers 列出支持的任务处理器详情
	ListHandlers() []executor.HandlerMeta
}

type service struct {
	mq       mq.MQ
	registry *executor.HandlerRegistry
	logger   *elog.Component
}

func (s *service) ListHandlers() []executor.HandlerMeta {
	return s.registry.ListMetas()
}

func NewService(mq mq.MQ) Service {
	s := &service{
		mq:       mq,
		registry: executor.NewHandlerRegistry(),
		logger:   elog.DefaultLogger.With(elog.FieldComponentName("execute.service")),
	}

	s.registerHandler(scripts.GetDefaultHandlers()...)
	return s
}

func (s *service) registerHandler(handlers ...executor.TaskHandler) {
	s.registry.Register(handlers...)
}

func (s *service) Receive(ctx context.Context, req domain.ExecuteReceive) (string, string, domain.Status, error) {
	// 1. 查找处理器 (优先使用 Handler，如果没有则退化到使用 Language)
	handlerName := req.Handler
	if handlerName == "" {
		handlerName = req.Language
	}

	h, ok := s.registry.Get(handlerName)
	if !ok {
		return "", "", domain.FAILED, fmt.Errorf("未找到对应的任务处理器: %s", handlerName)
	}

	// 2. 准备输出捕获器
	capLogger := &captureLogger{}

	// 3. 构建参数 map (将分散的各个属性聚合为 Handler 认识的 Context 参数)
	params := map[string]string{
		"code":      req.Code,
		"args":      req.Args,
		"variables": req.Variables,
	}

	// 4. 创建 Context
	taskCtx := executor.NewContextWithLogger(
		0, // Kafka 模式下暂无 ExecutionID
		req.TaskId,
		"kafka_task", // 任务名称
		handlerName,
		params,
		s.logger,
		capLogger,
	)
	defer taskCtx.Close()

	// 5. 调用 Handler 执行
	err := h.Run(taskCtx)

	// 6. 整理汇总结果
	result := taskCtx.GetResultJson() // 这是来自 FD 3 的纯净数据
	output := capLogger.GetContent()  // 这是来自 Stdout 的执行日志

	if err != nil {
		return result, output, domain.FAILED, err
	}

	return result, output, domain.SUCCESS, nil
}

// captureLogger 内部实现，用于捕获来自 Handler 的 Log 调用并存入 string 汇总
type captureLogger struct {
	buffer strings.Builder
}

func (c *captureLogger) Log(format string, args ...any) {
	c.buffer.WriteString(fmt.Sprintf(format, args...))
	c.buffer.WriteByte('\n')
}

func (c *captureLogger) Close() {}

func (c *captureLogger) GetContent() string {
	return c.buffer.String()
}
