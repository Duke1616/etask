package executor

import "github.com/Duke1616/etask/sdk/executor/internal/runtime"

const (
	RoleName           = runtime.RoleName
	ModePush           = runtime.ModePush
	ModePull           = runtime.ModePull
	IsolationShared    = runtime.IsolationShared
	IsolationDedicated = runtime.IsolationDedicated
)

type (
	// Config 描述 Executor 节点运行配置。
	Config = runtime.Config
)
