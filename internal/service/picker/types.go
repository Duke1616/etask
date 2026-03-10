package picker

import (
	"context"

	"github.com/Duke1616/etask/internal/domain"
)

// ExecutorNodePicker 是执行节点选择器的通用接口。
// 属于调度层辺辑：根据 task 的配置选出最优节点。
type ExecutorNodePicker interface {
	// Name 返回选择器的可读名称。
	Name() string
	// Pick 根据 task 选择最优执行节点，返回节点 ID。
	Pick(ctx context.Context, task domain.Task) (nodeID string, err error)
}

// IExecModeResolver 负责感知节点的执行模式并进行持久化。
// 与 picker 责任分离：picker 只选节点，resolver 负责“感知模式 + 写 DB”。
type IExecModeResolver interface {
	// ResolveMode 查询选中节点在注册时声明的执行模式，
	// 并将结果写入 tasks.exec_mode 作为快照记录，返回模式。
	// 写 DB 失败不阻断调度，仅记录日志。
	ResolveMode(ctx context.Context, task domain.Task, nodeID string) domain.ExecMode
}
