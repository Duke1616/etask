package ioc

import (
	"github.com/Duke1616/etask/internal/repository"
	"github.com/Duke1616/etask/internal/service/picker"
	"github.com/Duke1616/etask/pkg/grpc/registry"
)

// InitExecutorNodePicker 初始化节点选择器（只负责选节点）
func InitExecutorNodePicker(reg registry.Registry) picker.ExecutorNodePicker {
	return picker.NewRandomPicker(reg)
}

// InitExecModeResolver 初始化模式感知器（负责查 metadata + 写 tasks.exec_mode）
func InitExecModeResolver(reg registry.Registry, taskRepo repository.TaskRepository) picker.IExecModeResolver {
	return picker.NewExecModeResolver(reg, taskRepo)
}
