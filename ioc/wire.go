//go:build wireinject
// +build wireinject

package ioc

import (
	"github.com/google/wire"
)

// InitApp 全量启动注入器 (包含 Scheduler + Agent + Executor)
func InitApp() *App {
	wire.Build(
		BaseSet,
		WebSetup,
		TaskSet,
		ExecutorSet,
		TaskExecutionSet,
		SchedulerSet,
		CompensatorSet,
		ConsumerSet,
		ProducerSet,
		GrpcSet,
		AgentSet,
		AppSet,
		InitExecutor,
		wire.Struct(new(App), "*"),
	)
	return new(App)
}

// InitSchedulerApp 纯净的调度中心注入器 (不含原生 Executor)
func InitSchedulerApp() *App {
	wire.Build(
		BaseSet,
		WebSetup,
		TaskSet,
		ExecutorSet,
		TaskExecutionSet,
		SchedulerSet,
		CompensatorSet,
		ConsumerSet,
		ProducerSet,
		GrpcSet,
		AgentWebSet,
		AppSet,
		// 显式字段注入，忽略 Executor 字段，避免引入依赖
		// 同时忽略 Agent 字段，因为 Scheduler 模式下不需要运行 Agent 消费者
		wire.Struct(new(App), "Web", "Server", "Scheduler", "Tasks", "EndpointSvc"),
	)
	return new(App)
}

// InitExecutorApp 纯净的原生执行器注入器 (不含 DB/Redis/Kafka/Scheduler 等后台逻辑)
func InitExecutorApp() *App {
	wire.Build(
		// 仅包含基础 Etcd 和注册中心
		InitEtcdClient,
		InitRegistry,
		InitExecutor,

		// 仅填充 Web (健康检查) 和 Executor
		wire.Struct(new(App), "Executor"),
	)
	return new(App)
}

// InitAgentApp 纯净的异步代理注入器 (不含 DB/Redis/Scheduler 等后台逻辑)
func InitAgentApp() *App {
	wire.Build(
		InitEtcdClient,
		InitMQ,
		AgentSet,

		// 仅填充 Agent 模块
		wire.Struct(new(App), "Agent"),
	)
	return new(App)
}
