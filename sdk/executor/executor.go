package executor

import (
	"github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/Duke1616/etask/sdk/executor/internal/runtime"
)

// Executor 是任务执行节点的公开类型。
type Executor struct {
	*runtime.Executor
}

// Option 配置 Executor 的可选基础设施能力。
type Option = runtime.Option

// WithArtifactPreparer 注入可选的制品本地物化实现。
func WithArtifactPreparer(preparer ArtifactPreparer) Option {
	return runtime.WithArtifactPreparer(preparer)
}

// NewExecutor 创建 Executor 节点。
func NewExecutor(config Config, reg registry.Registry, options ...Option) (*Executor, error) {
	inner, err := runtime.NewExecutor(config, reg, options...)
	if err != nil {
		return nil, err
	}
	return &Executor{Executor: inner}, nil
}
