package ioc

import (
	"fmt"
	"os"
	"time"

	artifactv1 "github.com/Duke1616/etask/api/proto/gen/etask/artifact/v1"
	codebookv1 "github.com/Duke1616/etask/api/proto/gen/etask/codebook/v1"
	executorv1 "github.com/Duke1616/etask/api/proto/gen/etask/executor/v1"
	reporterv1 "github.com/Duke1616/etask/api/proto/gen/etask/reporter/v1"
	runnerv1 "github.com/Duke1616/etask/api/proto/gen/etask/runner/v1"
	taskv1 "github.com/Duke1616/etask/api/proto/gen/etask/task/v1"
	executorartifact "github.com/Duke1616/etask/internal/executor/artifact"
	grpcapi "github.com/Duke1616/etask/internal/grpc"
	"github.com/Duke1616/etask/internal/grpc/scripts"
	grpcpkg "github.com/Duke1616/etask/pkg/grpc"
	"github.com/Duke1616/etask/pkg/grpc/pool"
	registrysdk "github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/Duke1616/etask/sdk/executor"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// InitExecutor 初始化原生 gRPC 执行器节点
func InitExecutor(etcdClient *clientv3.Client) *executor.Executor {
	var serverCfg grpcpkg.ServerConfig
	if err := viper.UnmarshalKey("grpc.server.executor", &serverCfg); err != nil {
		panic(err)
	}

	var clientCfg grpcpkg.ClientConfig
	if err := viper.UnmarshalKey("grpc.client.scheduler", &clientCfg); err != nil {
		panic(err)
	}
	var artifactCacheCfg executorartifact.Config
	if err := viper.UnmarshalKey("executor.artifact_cache", &artifactCacheCfg); err != nil {
		panic(err)
	}
	var scriptRuntimeCfg scripts.RuntimeConfig
	if err := viper.UnmarshalKey("runtime.script", &scriptRuntimeCfg); err != nil {
		panic(err)
	}
	scriptRuntime, err := scripts.NewRuntime(scriptRuntimeCfg)
	if err != nil {
		panic(err)
	}
	if err = scriptRuntime.Initialize(); err != nil {
		panic(err)
	}

	cfg := executor.Config{
		Mode:           resolveMode(),
		Desc:           viper.GetString("executor.desc"),
		IsolationLevel: viper.GetString("executor.isolation_level"),
		Server:         resolveServer(serverCfg),
		Client:         clientCfg,
	}

	reg := InitExecutorRegistry(etcdClient)
	exec, err := executor.NewExecutor(cfg, reg,
		executor.WithArtifactPreparer(executorartifact.NewRuntime(artifactCacheCfg)),
	)
	if err != nil {
		panic(err)
	}

	exec.RegisterHandler(scriptRuntime.Handlers()...)

	// 立即初始化组件，确保 Server() 等方法能够返回有效对象
	if err = exec.InitComponents(); err != nil {
		panic(err)
	}

	return exec
}

// InitSchedulerNodeGRPCServer 初始化 Scheduler gRPC 服务器
func InitSchedulerNodeGRPCServer(registry registrysdk.Registry, reporter *grpcapi.ReporterServer,
	task *grpcapi.TaskServer, agent *grpcapi.AgentServer, codebook *grpcapi.CodebookServer,
	runner *grpcapi.RunnerServer, artifact *grpcapi.ArtifactServer) *grpcpkg.Server {
	var cfg grpcpkg.ServerConfig
	if err := viper.UnmarshalKey("grpc.server.scheduler", &cfg); err != nil {
		panic(err)
	}

	server := grpcpkg.NewServer(cfg, registry, grpcpkg.WithJWTAuth(cfg.AuthToken))
	reporterv1.RegisterReporterServiceServer(server.Server, reporter)
	taskv1.RegisterTaskServiceServer(server.Server, task)
	executorv1.RegisterAgentServiceServer(server.Server, agent)
	executorv1.RegisterTaskExecutionServiceServer(server.Server, agent)
	codebookv1.RegisterCodebookServiceServer(server.Server, codebook)
	runnerv1.RegisterRunnerServiceServer(server.Server, runner)
	artifactv1.RegisterArtifactServiceServer(server.Server, artifact)

	return server
}

func InitExecutorServiceGRPCClients(reg registrysdk.Registry) *pool.Clients[executorv1.ExecutorServiceClient] {
	const defaultTimeout = time.Second
	var cfg grpcpkg.ClientConfig
	if err := viper.UnmarshalKey("grpc.client.executor", &cfg); err != nil {
		panic(err)
	}

	return pool.NewClients(
		reg,
		defaultTimeout,
		cfg.AuthToken,
		func(conn *grpc.ClientConn) executorv1.ExecutorServiceClient {
			return executorv1.NewExecutorServiceClient(conn)
		})
}

// resolveServer 确定最终的 NodeID
// 优先级：环境变量 EXECUTOR_NODE_ID > 配置文件中的原始 ID
// 最终格式：serviceName:nodeID
func resolveServer(sc grpcpkg.ServerConfig) grpcpkg.ServerConfig {
	// 优先级 1: 环境变量
	nodeID := os.Getenv("EXECUTOR_NODE_ID")

	// 优先级 2: 配置文件中的 executor.id
	if nodeID == "" {
		nodeID = viper.GetString("executor.id")
	}

	// 优先级 3: 配置文件中的 grpc.server.executor.id (即传入的 ServiceId)
	if nodeID == "" {
		nodeID = sc.ServiceId
	}

	if nodeID != "" {
		sc.ServiceId = fmt.Sprintf("%s:%s", sc.ServiceName, nodeID)
	}

	return sc
}

func resolveMode() string {
	mode := viper.GetString("executor.mode")
	if mode == "" {
		mode = "PUSH"
	}

	return mode
}
