package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	artifactv1 "github.com/Duke1616/etask/api/proto/gen/etask/artifact/v1"
	"github.com/Duke1616/etask/internal/agent/domain"
	internaldomain "github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/sdk/executor"
	"github.com/gotomicro/ego/core/elog"
)

const executionRetention = 30 * time.Minute

// Service 定义独立 Kafka Agent 的执行能力。
type Service interface {
	// Receive 幂等执行一条 Kafka 命令。
	Receive(ctx context.Context, dispatchID string, command internaldomain.TaskExecution) (domain.ExecutionOutput, error)
	// ListHandlers 列出支持的任务处理器详情。
	ListHandlers() []executor.HandlerMeta
}

type service struct {
	registry       *executor.HandlerRegistry
	engine         *executor.ExecutionEngine
	artifactClient artifactv1.ArtifactServiceClient
	logger         *elog.Component
	mu             sync.Mutex
	executions     map[string]*executionEntry
}

type executionEntry struct {
	startedAt time.Time
	done      chan struct{}
	output    domain.ExecutionOutput
	err       error
}

// NewService 创建独立 Agent 执行服务。
func NewService(handlers []executor.TaskHandler, preparer executor.ArtifactPreparer,
	artifactClient artifactv1.ArtifactServiceClient) Service {
	registry := executor.NewHandlerRegistry()
	registry.Register(handlers...)
	return &service{
		registry: registry, engine: executor.NewExecutionEngine(registry, preparer),
		artifactClient: artifactClient,
		logger:         elog.DefaultLogger.With(elog.FieldComponentName("agent.execution")),
		executions:     make(map[string]*executionEntry),
	}
}

// ListHandlers 列出支持的任务处理器详情。
func (s *service) ListHandlers() []executor.HandlerMeta {
	return s.registry.ListMetas()
}

// Receive 幂等执行一条 Kafka 命令。
func (s *service) Receive(ctx context.Context, dispatchID string,
	execution internaldomain.TaskExecution) (domain.ExecutionOutput, error) {
	if dispatchID == "" || execution.ID <= 0 || execution.Task.GrpcConfig == nil {
		return domain.ExecutionOutput{}, fmt.Errorf("agent 执行命令缺少派发 ID、执行 ID 或处理器配置")
	}
	// dispatchID 是消息重投的幂等键；非 owner 等待首次执行结果而不重复运行。
	entry, owner := s.begin(dispatchID)
	if !owner {
		select {
		case <-ctx.Done():
			return domain.ExecutionOutput{}, ctx.Err()
		case <-entry.done:
			return entry.output, entry.err
		}
	}
	// Kafka Agent 使用内存日志器，随最终结果一次性返回调度中心。
	logger := &captureLogger{}
	refs, err := internaldomain.ArtifactRefsToProto(execution.Artifacts)
	if err != nil {
		s.finish(entry, domain.ExecutionOutput{}, err)
		return domain.ExecutionOutput{}, err
	}
	// 与独立 Executor 复用同一个 Engine，制品和 Handler 行为保持一致。
	result, err := s.engine.Execute(ctx, executor.ExecutionCommand{
		Context: ctx,
		Task: executor.TaskInfo{
			ExecutionID: execution.ID, TaskID: execution.Task.ID,
			Name: execution.Task.Name, Handler: execution.Task.GrpcConfig.HandlerName,
		},
		Params: execution.GRPCParams(), Parameters: s.handlerMetadata(execution.Task.GrpcConfig.HandlerName),
		Artifacts: refs, ArtifactClient: s.artifactClient,
		Logger: s.logger, TaskLogger: logger,
	})
	output := domain.ExecutionOutput{Result: result.Value, Logs: logger.Logs()}
	s.finish(entry, output, err)
	return output, err
}

func (s *service) handlerMetadata(name string) []executor.Parameter {
	handler, ok := s.registry.Get(name)
	if !ok {
		return nil
	}
	return handler.Metadata()
}

func (s *service) begin(dispatchID string) (*executionEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	// 新命令进入时顺便清理过期终态，避免额外维护后台清理协程。
	for id, entry := range s.executions {
		select {
		case <-entry.done:
			if now.Sub(entry.startedAt) >= executionRetention {
				delete(s.executions, id)
			}
		default:
		}
	}
	if entry := s.executions[dispatchID]; entry != nil {
		return entry, false
	}
	entry := &executionEntry{startedAt: now, done: make(chan struct{})}
	s.executions[dispatchID] = entry
	return entry, true
}

func (s *service) finish(entry *executionEntry, output domain.ExecutionOutput, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry.output = output
	entry.err = err
	close(entry.done)
}

type captureLogger struct {
	mu   sync.Mutex
	logs []string
}

func (c *captureLogger) Log(format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logs = append(c.logs, fmt.Sprintf(format, args...))
}

func (c *captureLogger) Close() {}

func (c *captureLogger) Logs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.logs...)
}
