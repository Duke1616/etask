package ioc

import (
	"time"

	executorv1 "github.com/Duke1616/ework-runner/api/proto/gen/etask/executor/v1"
	reporterv1 "github.com/Duke1616/ework-runner/api/proto/gen/etask/reporter/v1"
	taskv1 "github.com/Duke1616/ework-runner/api/proto/gen/etask/task/v1"
	grpcapi "github.com/Duke1616/ework-runner/internal/grpc"
	grpcpkg "github.com/Duke1616/ework-runner/pkg/grpc"
	"github.com/Duke1616/ework-runner/pkg/grpc/pool"
	registrysdk "github.com/Duke1616/ework-runner/pkg/grpc/registry"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

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
