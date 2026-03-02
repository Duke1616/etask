//go:build wireinject
// +build wireinject

package ioc

import (
	"github.com/Duke1616/etask/ioc"
	grpcpkg "github.com/Duke1616/etask/pkg/grpc"
	"github.com/Duke1616/etask/sdk/executor"
	"github.com/google/wire"
)

// ExecuteApp 仅包含 Executor 的服务器实例
type ExecuteApp struct {
	Server *grpcpkg.Server
}

// InitExecuteApp 该注入器支持原有的 executor 二进制启动，但代码逻辑复用根目录的公共 ioc
func InitExecuteApp() *ExecuteApp {
	wire.Build(
		// 1. 复用根目录的基础基础设施
		ioc.InitEtcdClient,
		ioc.InitRegistry,

		// 2. 复用根目录的 Executor 初始化逻辑 (包含配置解析和 Handler 注册)
		ioc.InitExecutor,

		// 3. 提取 gRPC Server
		InitExecutorServer,

		// 4. 构建 App
		wire.Struct(new(ExecuteApp), "*"),
	)
	return new(ExecuteApp)
}

// InitExecutorServer 是一个小适配器，将公共的 executor 对象转换为二进制需要的 server
func InitExecutorServer(exec *executor.Executor) *grpcpkg.Server {
	return exec.Server()
}
