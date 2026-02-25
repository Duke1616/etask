package picker

import (
	"context"
	"fmt"

	"github.com/Duke1616/ework-runner/internal/domain"
	"github.com/Duke1616/ework-runner/internal/repository"
	"github.com/Duke1616/ework-runner/pkg/grpc/registry"
	"github.com/gotomicro/ego/core/elog"
)

var _ IExecModeResolver = &ExecModeResolver{}

// ExecModeResolver 实现 IExecModeResolver 接口。
// 职责：查询 registry metadata 中 executor 声明的执行模式，并将结果快照写入 tasks 表。
type ExecModeResolver struct {
	reg      registry.Registry
	taskRepo repository.TaskRepository
	logger   *elog.Component
}

// NewExecModeResolver 创建 ExecModeResolver
func NewExecModeResolver(reg registry.Registry, taskRepo repository.TaskRepository) IExecModeResolver {
	return &ExecModeResolver{
		reg:      reg,
		taskRepo: taskRepo,
		logger:   elog.DefaultLogger.With(elog.FieldComponentName("picker.ExecModeResolver")),
	}
}

// ResolveMode 查询目标节点注册时声明的执行模式，将结果写入 tasks.exec_mode，并返回结果。
// NOTE: executor 在注册时通过 buildMetadata() 上报 mode 字段（PUSH/PULL）。
// 若节点不存在或未声明 mode，降级为默认 PUSH 模式。
// 写 DB 失败不阻断调度，仅 Warn 日志。
func (r *ExecModeResolver) ResolveMode(ctx context.Context, task domain.Task, nodeID string) domain.ExecMode {
	// 1. 从 registry 找到目标节点的 metadata
	services, err := r.reg.ListServices(ctx, task.GrpcConfig.ServiceName)
	if err != nil {
		r.logger.Warn("ResolveMode: 查询注册中心失败，降级为 PUSH",
			elog.Int64("taskID", task.ID), elog.FieldErr(err))
		return domain.ExecModePush
	}

	mode := domain.ExecModePush // 默认 PUSH
	for _, si := range services {
		if si.ID == nodeID {
			if modeVal, ok := si.Metadata["mode"]; ok {
				m := domain.ExecMode(fmt.Sprintf("%v", modeVal))
				if m.IsPull() {
					mode = domain.ExecModePull
				}
			}
			break
		}
	}

	// 2. 写入 tasks.exec_mode 作为快照（写失败不阻断调度）
	if err = r.taskRepo.UpdateExecMode(ctx, task.ID, mode); err != nil {
		r.logger.Warn("ResolveMode: 写 tasks.exec_mode 失败，继续调度",
			elog.Int64("taskID", task.ID), elog.FieldErr(err))
	}

	return mode
}
