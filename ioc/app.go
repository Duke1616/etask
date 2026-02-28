package ioc

import (
	"context"

	endpointv1 "github.com/Duke1616/ework-runner/api/proto/gen/ecmdb/endpoint/v1"
	"github.com/Duke1616/ework-runner/internal/agent"
	"github.com/Duke1616/ework-runner/internal/service/scheduler"
	grpcpkg "github.com/Duke1616/ework-runner/pkg/grpc"
	"github.com/gotomicro/ego/server/egin"
)

// Task 调度平台上的长任务 —— 各种补偿任务、消费者等
type Task interface {
	Start(ctx context.Context)
}

type App struct {
	Web         *egin.Component
	Server      *grpcpkg.Server
	Scheduler   *scheduler.Scheduler
	Agent       *agent.Module
	Tasks       []Task
	EndpointSvc endpointv1.EndpointServiceClient
}

func (a *App) StartTasks(ctx context.Context) {
	// NOTE: 混合模式下，启动全量任务
	a.StartAgent(ctx)
	a.StartSchedulerTasks(ctx)
}

func (a *App) StartAgent(ctx context.Context) {
	// 启动 Agent 任务消费逻辑 (从 Kafka 拉取抢占到的任务并执行)
	if a.Agent != nil {
		a.Agent.GetConsumer().Start(ctx)
	}
}

func (a *App) StartSchedulerTasks(ctx context.Context) {
	// 启动调度中心配套的各个异步任务（如补偿、重试、已完成任务上报等）
	for _, t := range a.Tasks {
		go func(t Task) {
			t.Start(ctx)
		}(t)
	}
}
