//go:build wireinject
// +build wireinject

package ioc

import (
	"github.com/Duke1616/ecmdb/pkg/policy"
	"github.com/Duke1616/etask/internal/agent"
	grpcpkg "github.com/Duke1616/etask/pkg/grpc"
	"github.com/Duke1616/etask/sdk/executor"
	"github.com/google/wire"
	"github.com/gotomicro/ego/server/egin"
)

// InitBase 初始化所有共享基础设施（建立连接，但不运行业务）
func InitBase() *Base {
	wire.Build(
		BaseSet,
		WebSetup,
		wire.Struct(new(Base), "*"),
	)
	return new(Base)
}

// InitSchedulerModule 专门用于构造 Scheduler 模块及其配套后台任务
func InitSchedulerModule(base *Base) *SchedulerModule {
	wire.Build(
		TaskSet,
		TaskExecutionSet,
		SchedulerSet,
		CompensatorSet,
		ConsumerSet,
		ProducerSet,
		GrpcSet,
		InitRunner,
		InitInvoker,
		// 从 Base 中提取基础资源
		wire.FieldsOf(new(*Base), "Registry", "MQ"),
		InitTasks,
		wire.Struct(new(SchedulerModule), "Svc", "Tasks"),
	)
	return nil
}

// InitExecutorModule 专门用于构造原生执行器模块
func InitExecutorModule(base *Base) *executor.Executor {
	wire.Build(
		InitExecutor,
		wire.FieldsOf(new(*Base), "Registry"),
	)
	return nil
}

// InitAgentModule 专门用于构造异步代理模块 (包含 Kafka 消费者)
func InitAgentModule(base *Base) *agent.Module {
	wire.Build(
		AgentSet,
		wire.FieldsOf(new(*Base), "MQ", "Etcd"),
	)
	return nil
}

// InitSchedulerServerModule 构造调度中心的 gRPC 服务端 (负责接收上报、下发任务及服务注册)
func InitSchedulerServerModule(base *Base) *grpcpkg.Server {
	wire.Build(
		TaskSet,
		TaskExecutionSet,
		SchedulerSet,
		AppSet,
		ProducerSet,
		// 从 Base 中提取依赖
		wire.FieldsOf(new(*Base), "Registry", "MQ"),
	)
	return nil
}

// InitWebModule 专门用于构造管理后台 Web 路由
func InitWebModule(base *Base) *egin.Component {
	wire.Build(
		TaskSet,
		TaskExecutionSet,
		ExecutorSet,
		AgentWebSet,
		// 只保留 Web 处理器的构造逻辑，基础资源从 base 拿
		InitGinWebServer,
		InitGinMiddlewares,
		policy.NewSDK,

		// 从 Base 中提取依赖，避免重复绑定 BaseSet/WebSetup
		wire.FieldsOf(new(*Base), "Registry", "Listener"),
	)
	return nil
}
