package ioc

import (
	"github.com/Duke1616/etask/internal/grpc/scripts"
	config "github.com/Duke1616/etask/pkg/config"
)

// InitScriptRuntime 创建 Agent 和 Executor 共享的脚本运行时。
func InitScriptRuntime() *scripts.Runtime {
	var cfg scripts.RuntimeConfig
	if err := config.UnmarshalKey("runtime", &cfg); err != nil {
		panic(err)
	}
	runtime, err := scripts.NewRuntime(cfg)
	if err != nil {
		panic(err)
	}
	return runtime
}
