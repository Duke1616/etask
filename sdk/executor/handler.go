// Package executor 提供任务处理器和执行节点 SDK。
package executor

import "github.com/Duke1616/etask/sdk/executor/internal/task"

type (
	// Variable 描述传给任务处理器的一个变量。
	Variable = task.Variable
	// Parameter 描述任务处理器支持的一个参数。
	Parameter = task.Parameter
	// Binding 定义运行阶段的参数绑定行为。
	Binding = task.Binding
	// BindingOption 描述参数绑定的前端配置和可选解析函数。
	BindingOption = task.BindingOption
	// TaskHandler 定义 Executor 可以调度的一类任务。
	TaskHandler = task.TaskHandler
	// HandlerMeta 是处理器展示和注册元数据。
	HandlerMeta = task.HandlerMeta
	// HandlerRegistry 并发安全地管理任务处理器。
	HandlerRegistry = task.HandlerRegistry
)

// NewHandlerRegistry 创建任务处理器注册中心。
func NewHandlerRegistry() *HandlerRegistry {
	return task.NewHandlerRegistry()
}
