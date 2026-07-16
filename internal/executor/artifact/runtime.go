// Package artifact 实现 Executor 节点的制品下载、缓存和本地物化。
package artifact

import (
	"context"

	artifactv1 "github.com/Duke1616/etask/api/proto/gen/etask/artifact/v1"
	"github.com/Duke1616/etask/sdk/executor"
)

// Prepared 描述任务可使用的制品目录和清理函数。
type Prepared struct {
	DefaultRoot      string
	DependenciesRoot string
	cleanup          func()
}

// Cleanup 清理本次任务创建的临时制品聚合目录。
func (p Prepared) Cleanup() {
	if p.cleanup != nil {
		p.cleanup()
	}
}

// Runtime 管理制品缓存和任务制品目录准备。
type Runtime struct {
	cache *artifactCache
}

// NewRuntime 创建制品运行时。
func NewRuntime(config Config) *Runtime {
	return &Runtime{cache: newArtifactCache(config)}
}

// Prune 清理未完成下载和超出容量限制的缓存层。
func (r *Runtime) Prune() error {
	return r.cache.Prune()
}

// Prepare 下载并组合任务声明的系统制品层和具名依赖层。
func (r *Runtime) Prepare(ctx context.Context, client artifactv1.ArtifactServiceClient,
	refs []*artifactv1.ArtifactRef) (executor.PreparedArtifacts, error) {
	prepared, err := (artifactPreparer{cache: r.cache, client: client}).Prepare(ctx, refs)
	if err != nil {
		return nil, err
	}
	return prepared, nil
}

var _ executor.ArtifactPreparer = (*Runtime)(nil)
