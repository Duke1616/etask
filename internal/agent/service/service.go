package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Duke1616/ework-runner/internal/agent/domain"
	"github.com/Duke1616/ework-runner/internal/grpc/scripts"
	"github.com/Duke1616/ework-runner/sdk/executor"
	"github.com/ecodeclub/mq-api"
	"github.com/gotomicro/ego/core/elog"
)

type HandlerDetail struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

type Service interface {
	// Receive 接收任务并执行
	Receive(ctx context.Context, req domain.ExecuteReceive) (string, domain.Status, error)

	// ListHandlers 列出支持的任务处理器详情
	ListHandlers() []HandlerDetail
}

type service struct {
	mq       mq.MQ
	handlers map[string]executor.TaskHandler
	logger   *elog.Component
}

func (s *service) ListHandlers() []HandlerDetail {
	metas := make([]HandlerDetail, 0, len(s.handlers))
	for _, h := range s.handlers {
		metas = append(metas, HandlerDetail{
			Name: h.Name(),
			Desc: h.Desc(),
		})
	}
	return metas
}

func NewService(mq mq.MQ) Service {
	s := &service{
		mq:       mq,
		handlers: make(map[string]executor.TaskHandler),
		logger:   elog.DefaultLogger.With(elog.FieldComponentName("execute.service")),
	}

	// 统一复用 gRPC 脚本处理器
	s.registerHandler(scripts.NewShellTaskHandler())
	s.registerHandler(scripts.NewPythonTaskHandler())

	return s
}

func (s *service) registerHandler(h executor.TaskHandler) {
	s.handlers[h.Name()] = h
}

func (s *service) Receive(ctx context.Context, req domain.ExecuteReceive) (string, domain.Status, error) {
	// 1. 查找处理器 (优先使用 Handler，如果没有则退化到使用 Language)
	handlerName := req.Handler
	if handlerName == "" {
		handlerName = req.Language
	}

	h, ok := s.handlers[handlerName]
	if !ok {
		return "", domain.FAILED, fmt.Errorf("未找到对应的任务处理器: %s", handlerName)
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
	output := capLogger.GetContent()
	if err != nil {
		return output, domain.FAILED, err
	}

	return output, domain.SUCCESS, nil
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
