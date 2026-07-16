// Package artifactport 定义 Executor 与可选制品物化实现之间的端口。
package artifactport

import (
	"context"

	artifactv1 "github.com/Duke1616/etask/api/proto/gen/etask/artifact/v1"
	"github.com/Duke1616/etask/sdk/executor/internal/task"
)

// PreparedArtifacts 是一次任务独占的制品运行现场。
type PreparedArtifacts interface {
	// Roots 返回 Handler 可读取的系统制品层和具名依赖层目录。
	Roots() task.ArtifactRoots
	// Close 清理本次准备产生的临时目录。
	Close() error
}

// Preparer 定义 Executor 可选的制品本地物化能力。
type Preparer interface {
	// Prune 清理无效或超限的本地制品缓存。
	Prune() error
	// Prepare 下载并准备任务使用的制品运行现场。
	Prepare(ctx context.Context, client artifactv1.ArtifactServiceClient,
		refs []*artifactv1.ArtifactRef) (PreparedArtifacts, error)
}
