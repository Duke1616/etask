package executor

import "github.com/Duke1616/etask/sdk/executor/internal/engine"

type (
	// ExecutionCommand 描述传输适配器提交给进程内执行引擎的命令。
	ExecutionCommand = engine.Command
	// ExecutionResult 描述进程内执行引擎返回的结构化结果。
	ExecutionResult = engine.Result
	// ExecutionEngine 统一编排制品准备和 Handler 调用。
	ExecutionEngine = engine.Engine
)

// NewExecutionEngine 创建可由 Executor 或 Agent 独立持有的执行引擎。
func NewExecutionEngine(handlers *HandlerRegistry, artifacts ArtifactPreparer) *ExecutionEngine {
	return engine.New(handlers, artifacts)
}
