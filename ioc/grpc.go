package ioc

import (
	"fmt"
	"os"
	"time"

	notificationv1 "github.com/Duke1616/etask/api/proto/gen/ealert/notification/v1"
	templatev1 "github.com/Duke1616/etask/api/proto/gen/ealert/template/v1"
	artifactv1 "github.com/Duke1616/etask/api/proto/gen/etask/artifact/v1"
	codebookv1 "github.com/Duke1616/etask/api/proto/gen/etask/codebook/v1"
	executorv1 "github.com/Duke1616/etask/api/proto/gen/etask/executor/v1"
	reporterv1 "github.com/Duke1616/etask/api/proto/gen/etask/reporter/v1"
	runnerv1 "github.com/Duke1616/etask/api/proto/gen/etask/runner/v1"
	schedulerv1 "github.com/Duke1616/etask/api/proto/gen/etask/scheduler/v1"
	taskv1 "github.com/Duke1616/etask/api/proto/gen/etask/task/v1"
	grpcapi "github.com/Duke1616/etask/internal/grpc"
	"github.com/Duke1616/etask/internal/grpc/scripts"
	"github.com/Duke1616/etask/pkg/config"
	grpcpkg "github.com/Duke1616/etask/pkg/grpc"
	jwtinterceptor "github.com/Duke1616/etask/pkg/grpc/interceptors/jwt"
	"github.com/Duke1616/etask/pkg/grpc/pool"
	registrysdk "github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/Duke1616/etask/sdk/executor/artifact"
	"github.com/Duke1616/etask/sdk/executor/node"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// InitEAlertClientConn 初始化 EAlert gRPC 连接，供通知和模板客户端复用。
func InitEAlertClientConn(reg registrysdk.Registry) *grpc.ClientConn {
	var cfg grpcpkg.ClientConfig
	if err := config.UnmarshalKey("grpc.client.ealert", &cfg); err != nil {
		panic(err)
	}
	opts := []grpcpkg.ClientOption{grpcpkg.WithClientJWTAuth(cfg.AuthToken)}
	var (
		conn *grpc.ClientConn
		err  error
	)
	if cfg.Address != "" {
		conn, err = grpcpkg.NewDirectClientConn(cfg.Address, opts...)
	} else {
		conn, err = grpcpkg.NewClientConn(reg, append(opts, grpcpkg.WithServiceName(cfg.Name))...)
	}
	if err != nil {
		panic(err)
	}
	return conn
}

// InitEAlertNotificationClient 初始化 EAlert 通知客户端。
func InitEAlertNotificationClient(conn *grpc.ClientConn) notificationv1.NotificationServiceClient {
	return notificationv1.NewNotificationServiceClient(conn)
}

// InitEAlertTemplateClient 初始化 EAlert 模板服务客户端。
func InitEAlertTemplateClient(conn *grpc.ClientConn) templatev1.TemplateServiceClient {
	return templatev1.NewTemplateServiceClient(conn)
}

// InitExecutor 初始化原生 gRPC 执行器节点
func InitExecutor(reg registrysdk.Registry,
	artifactPreparer artifact.Preparer, scriptRuntime *scripts.Runtime) *node.Executor {
	var serverCfg grpcpkg.ServerConfig
	if err := config.UnmarshalKey("grpc.server.executor", &serverCfg); err != nil {
		panic(err)
	}

	var clientCfg grpcpkg.ClientConfig
	if err := config.UnmarshalKey("grpc.client.scheduler", &clientCfg); err != nil {
		panic(err)
	}
	if err := scriptRuntime.Initialize(); err != nil {
		panic(err)
	}

	cfg := node.Config{
		Mode:           resolveMode(),
		Desc:           viper.GetString("executor.desc"),
		IsolationLevel: viper.GetString("executor.isolation_level"),
		Server:         resolveServer(serverCfg),
		Client:         clientCfg,
	}

	nodeOpts := []node.Option{
		node.WithArtifactPreparer(artifactPreparer),
	}

	exec, err := node.NewExecutor(cfg, reg, nodeOpts...)
	if err != nil {
		panic(err)
	}

	if err = exec.RegisterHandlers(scriptRuntime.Handlers()...); err != nil {
		panic(err)
	}

	// 立即初始化组件，确保 Server() 等方法能够返回有效对象
	if err = exec.InitComponents(); err != nil {
		panic(err)
	}

	return exec
}

// InitSchedulerNodeGRPCServer 初始化 Scheduler gRPC 服务器
func InitSchedulerNodeGRPCServer(registry registrysdk.Registry, reporter *grpcapi.ReporterServer,
	task *grpcapi.TaskServer, agent *grpcapi.AgentServer, codebook *grpcapi.CodebookServer,
	runner *grpcapi.RunnerServer, artifact *grpcapi.ArtifactServer,
	scheduler *grpcapi.SchedulerServer, km jwtinterceptor.IClusterKeyManager) *grpcpkg.Server {
	var cfg grpcpkg.ServerConfig
	if err := config.UnmarshalKey("grpc.server.scheduler", &cfg); err != nil {
		panic(err)
	}

	serverOpts := []grpcpkg.ServerOption{grpcpkg.WithJWTAuth(cfg.AuthToken)}
	if km != nil {
		// 在向注册中心注册服务实例时，将集群 RSA 公钥广播到元数据中，使下游 Executor 免直连 Redis
		serverOpts = append(serverOpts, grpcpkg.WithMetadata(map[string]any{
			"public_key": km.ExportPublicKeyPEM(),
		}))
	}

	server := grpcpkg.NewServer(cfg, registry, serverOpts...)
	reporterv1.RegisterReporterServiceServer(server.Server, reporter)
	taskv1.RegisterTaskServiceServer(server.Server, task)
	executorv1.RegisterAgentServiceServer(server.Server, agent)
	executorv1.RegisterTaskExecutionServiceServer(server.Server, agent)
	codebookv1.RegisterCodebookServiceServer(server.Server, codebook)
	runnerv1.RegisterRunnerServiceServer(server.Server, runner)
	artifactv1.RegisterArtifactServiceServer(server.Server, artifact)
	schedulerv1.RegisterSchedulerServiceServer(server.Server, scheduler)

	return server
}

func InitExecutorServiceGRPCClients(reg registrysdk.Registry, km jwtinterceptor.IClusterKeyManager) *pool.Clients[executorv1.ExecutorServiceClient] {
	const defaultTimeout = time.Second
	var cfg grpcpkg.ClientConfig
	// 可选兼容读取：即使配置文件已删除该项，也安全保持零值并自动进入 RSA 动态签名
	_ = config.UnmarshalKey("grpc.client.executor", &cfg)

	poolOpts := make([]pool.ClientPoolOption, 0, 1)
	// 若未显式配置静态 authToken 且密钥管理器就绪，则自动挂载基于集群 RSA 的动态签名策略
	if cfg.AuthToken == "" && km != nil {
		poolOpts = append(poolOpts, pool.WithPoolTokenProvider(jwtinterceptor.NewRSATokenProvider(km)))
	}

	return pool.NewClients(
		reg,
		defaultTimeout,
		cfg.AuthToken,
		func(conn *grpc.ClientConn) executorv1.ExecutorServiceClient {
			return executorv1.NewExecutorServiceClient(conn)
		},
		poolOpts...,
	)
}

// resolveServer 确定最终的 NodeID
// 优先级：EXECUTOR_NODE_ID > executor.id > grpc.server.executor.id。
// 最终格式：serviceName:nodeID
func resolveServer(sc grpcpkg.ServerConfig) grpcpkg.ServerConfig {
	nodeID := os.Getenv("EXECUTOR_NODE_ID")

	if nodeID == "" {
		nodeID = viper.GetString("executor.id")
	}

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
