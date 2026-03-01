package ioc

import (
	"time"

	executorv1 "github.com/Duke1616/ework-runner/api/proto/gen/etask/executor/v1"
	reporterv1 "github.com/Duke1616/ework-runner/api/proto/gen/etask/reporter/v1"
	taskv1 "github.com/Duke1616/ework-runner/api/proto/gen/etask/task/v1"
	grpcapi "github.com/Duke1616/ework-runner/internal/grpc"
	"github.com/Duke1616/ework-runner/internal/grpc/scripts"
	grpcpkg "github.com/Duke1616/ework-runner/pkg/grpc"
	"github.com/Duke1616/ework-runner/pkg/grpc/pool"
	registrysdk "github.com/Duke1616/ework-runner/pkg/grpc/registry"
	"github.com/Duke1616/ework-runner/sdk/executor"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// InitExecutor 初始化原生 gRPC 执行器节点
func InitExecutor(reg registrysdk.Registry) *executor.Executor {
	var serverCfg grpcpkg.ServerConfig
	if err := viper.UnmarshalKey("grpc.server.executor", &serverCfg); err != nil {
		panic(err)
	}

	var clientCfg grpcpkg.ClientConfig
	if err := viper.UnmarshalKey("grpc.client.scheduler", &clientCfg); err != nil {
		panic(err)
	}

	mode := viper.GetString("executor.mode")
	if mode == "" {
		mode = "PUSH"
	}
	desc := viper.GetString("executor.desc")

	cfg := executor.Config{
		Mode:   mode,
		Desc:   desc,
		Server: serverCfg,
		Client: clientCfg,
	}

	exec, err := executor.NewExecutor(cfg, reg)
	if err != nil {
		panic(err)
	}

	// 注册默认支持的处理器
	exec.RegisterHandler(scripts.GetDefaultHandlers()...)

	// 立即初始化组件，确保 Server() 等方法能够返回有效对象
	if err = exec.InitComponents(); err != nil {
		panic(err)
	}

	return exec
}

// InitSchedulerNodeGRPCServer 初始化 Scheduler gRPC 服务器
func InitSchedulerNodeGRPCServer(registry registrysdk.Registry, reporter *grpcapi.ReporterServer,
	task *grpcapi.TaskServer, agent *grpcapi.AgentServer) *grpcpkg.Server {
	var cfg grpcpkg.ServerConfig
	if err := viper.UnmarshalKey("grpc.server.scheduler", &cfg); err != nil {
		panic(err)
	}

	server := grpcpkg.NewServer(cfg, registry, grpcpkg.WithJWTAuth(cfg.AuthToken))
	reporterv1.RegisterReporterServiceServer(server.Server, reporter)
	taskv1.RegisterTaskServiceServer(server.Server, task)
	executorv1.RegisterAgentServiceServer(server.Server, agent)
	executorv1.RegisterTaskExecutionServiceServer(server.Server, agent)

	return server
}

func InitExecutorServiceGRPCClients(reg registrysdk.Registry) *pool.Clients[executorv1.ExecutorServiceClient] {
	const defaultTimeout = time.Second
	return pool.NewClients(
		reg,
		defaultTimeout,
		func(conn *grpc.ClientConn) executorv1.ExecutorServiceClient {
			return executorv1.NewExecutorServiceClient(conn)
		})
}
