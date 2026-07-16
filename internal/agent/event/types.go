package event

import executionevent "github.com/Duke1616/etask/internal/event/execution"

type (
	// ExecuteCommand 是 Agent 接收的不可变执行命令。
	ExecuteCommand = executionevent.Command
	// ExecuteResult 是 Agent 发布的执行结果。
	ExecuteResult = executionevent.Result
)

const ExecuteResultEventName = executionevent.ResultTopic
