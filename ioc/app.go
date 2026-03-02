package ioc

import (
	"context"

	endpointv1 "github.com/Duke1616/etask/api/proto/gen/ecmdb/endpoint/v1"
	"github.com/Duke1616/etask/internal/agent"
	"github.com/Duke1616/etask/internal/service/scheduler"
	grpcpkg "github.com/Duke1616/etask/pkg/grpc"
	"github.com/Duke1616/etask/sdk/executor"
	"github.com/gotomicro/ego/server"
	"github.com/gotomicro/ego/server/egin"
)

const (
	ModeAll       = "all"
	ModeScheduler = "scheduler"
	ModeAgent     = "agent"
	ModeExecutor  = "executor"
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
	Executor    *executor.Executor
	Tasks       []Task
	EndpointSvc endpointv1.EndpointServiceClient
}

// GetServers 根据运行模式获取需要启动的服务列表
func (a *App) GetServers(modes []string) []server.Server {
	modeMap := a.resolveModes(modes)
	var res []server.Server

	// 1. 基础 Web
	if len(modeMap) > 0 && a.Web != nil {
		res = append(res, a.Web)
	}

	// 2. 调度中心模式：开启 gRPC 和 Scheduler
	if modeMap[ModeAll] || modeMap[ModeScheduler] {
		if a.Server != nil {
			res = append(res, a.Server)
		}
		if a.Scheduler != nil {
			res = append(res, a.Scheduler)
		}
	}

	// 3. 异步代理模式 (Agent)
	if (modeMap[ModeAll] || modeMap[ModeAgent]) && a.Agent != nil {
		res = append(res, a.Agent)
	}

	// 4. 原生执行器模式：开启 Executor 的 gRPC Server
	if (modeMap[ModeAll] || modeMap[ModeExecutor]) && a.Executor != nil {
		s := a.Executor.Server()
		if s != nil {
			res = append(res, s)
		}
	}

	return res
}

// StartBackgroundTasks 根据模式启动相关的后台任务
func (a *App) StartBackgroundTasks(ctx context.Context, modes []string) {
	modeMap := a.resolveModes(modes)

	if (modeMap[ModeAll] || modeMap[ModeScheduler]) && a.Scheduler != nil {
		a.StartSchedulerTasks(ctx)
	}
}

func (a *App) resolveModes(modes []string) map[string]bool {
	res := make(map[string]bool)
	for _, m := range modes {
		res[m] = true
	}
	return res
}

func (a *App) StartSchedulerTasks(ctx context.Context) {
	// 启动调度中心配套的各个异步任务（如补偿、重试、已完成任务上报等）
	for _, t := range a.Tasks {
		go func(t Task) {
			t.Start(ctx)
		}(t)
	}
}
