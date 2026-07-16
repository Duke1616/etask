package ioc

import (
	"github.com/Duke1616/etask/internal/service/picker"
	"github.com/Duke1616/etask/pkg/grpc/registry"
)

// InitExecutorNodePicker 初始化 gRPC PUSH 节点选择器。
func InitExecutorNodePicker(reg registry.Registry) picker.ExecutorNodePicker {
	return picker.NewRandomPicker(reg)
}
