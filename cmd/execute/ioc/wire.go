//go:build wireinject

package ioc

import (
	"github.com/Duke1616/ework-runner/internal/grpc"
	"github.com/Duke1616/ework-runner/internal/grpc/scripts"
	"github.com/Duke1616/ework-runner/ioc"
	grpcpkg "github.com/Duke1616/ework-runner/pkg/grpc"
	"github.com/Duke1616/ework-runner/pkg/grpc/registry"
	"github.com/Duke1616/ework-runner/pkg/grpc/registry/etcd"
	"github.com/Duke1616/ework-runner/sdk/executor"
	"github.com/google/wire"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	BaseSet = wire.NewSet(
		ioc.InitEtcdClient,
		InitRegistry,
	)

	ExecutorSet = wire.NewSet(
		InitConfig,
		InitExecutor,
		InitExecutorServer,
	)
)

func InitExecuteApp() *ExecuteApp {
	wire.Build(
		// 基础设施
		BaseSet,
		// Agent 组件
		ExecutorSet,
		wire.Struct(new(ExecuteApp), "*"),
	)

	return new(ExecuteApp)
}

// InitRegistry 初始化注册中心
func InitRegistry(client *clientv3.Client) registry.Registry {
	// NOTE: 统一使用 service 前缀
	reg, err := etcd.NewRegistry(client)
	if err != nil {
		panic(err)
	}
	return reg
}

// InitConfig 初始化配置
func InitConfig() executor.Config {
	var server grpcpkg.ServerConfig
	if err := viper.UnmarshalKey("grpc.server.executor", &server); err != nil {
		panic(err)
	}

	var client grpcpkg.ClientConfig
	if err := viper.UnmarshalKey("grpc.client.scheduler", &client); err != nil {
		panic(err)
	}

	// Mode 和 Desc 从独立的 executor 配置节点读取
	// Mode 如果未配置，默认使用 PUSH 模式（调度中心主动推送）
	mode := viper.GetString("executor.mode")
	if mode == "" {
		mode = "PUSH"
	}
	desc := viper.GetString("executor.desc")

	return executor.Config{
		Mode:   mode,
		Desc:   desc,
		Server: server,
		Client: client,
	}
}

// InitExecutor 初始化 SDK Agent 实例
func InitExecutor(cfg executor.Config, reg registry.Registry) *executor.Executor {
	exec, err := executor.NewExecutor(cfg, reg)
	if err != nil {
		panic(err)
	}

	// 注册处理函数
	exec.RegisterHandler(&grpc.DemoTaskHandler{})
	exec.RegisterHandler(scripts.NewShellTaskHandler())
	exec.RegisterHandler(scripts.NewPythonTaskHandler())

	// 初始化内部组件(连接Reporter等)
	if err = exec.InitComponents(); err != nil {
		panic(err)
	}

	return exec
}

// InitExecutorServer 从 Agent 中提取 ego Server
func InitExecutorServer(exec *executor.Executor) *grpcpkg.Server {
	return exec.Server()
}

type ExecuteApp struct {
	Server *grpcpkg.Server
}
