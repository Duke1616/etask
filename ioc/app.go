package ioc

import (
	"context"
	"net"

	endpointv1 "github.com/Duke1616/ecmdb/api/proto/gen/ecmdb/endpoint/v1"
	"github.com/Duke1616/etask/internal/agent"
	"github.com/Duke1616/etask/internal/service/scheduler"
	grpcpkg "github.com/Duke1616/etask/pkg/grpc"
	"github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/Duke1616/etask/sdk/executor"
	"github.com/ecodeclub/mq-api"
	"github.com/gotomicro/ego/server"
	"github.com/gotomicro/ego/server/egin"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	ModeAll       = "all"
	ModeScheduler = "scheduler"
	ModeAgent     = "agent"
	ModeExecutor  = "executor"
)

// IsDBRequired 判断是否需要数据库连接
func IsDBRequired(modes []string) bool {
	for _, m := range modes {
		// 只有 all 模式或明确的 scheduler 模式需要数据库
		if m == ModeAll || m == ModeScheduler {
			return true
		}
	}
	return false
}

// Task 调度平台上的长任务 —— 各种补偿任务、消费者等
type Task interface {
	Start(ctx context.Context)
}

// Base 基础基础设施（共享连接、客户端等）
type Base struct {
	Registry    registry.Registry
	MQ          mq.MQ
	Etcd        *clientv3.Client
	Listener    net.Listener
	EndpointSvc endpointv1.EndpointServiceClient
}

// SchedulerModule 调度中心模块资源
type SchedulerModule struct {
	Svc   *scheduler.Scheduler
	Tasks []Task
}

// App 模块化容器
type App struct {
	Web       *egin.Component
	Server    *grpcpkg.Server
	Scheduler *scheduler.Scheduler
	Agent     *agent.Module
	Executor  *executor.Executor
	Tasks     []Task

	// 共享基础资源
	Base *Base
}

// Load 加载模块到容器
func (a *App) Load(m any) {
	switch mod := m.(type) {
	case *egin.Component:
		a.Web = mod
	case *grpcpkg.Server:
		a.Server = mod
	case *scheduler.Scheduler:
		a.Scheduler = mod
	case *SchedulerModule:
		a.Scheduler = mod.Svc
		a.Tasks = append(a.Tasks, mod.Tasks...)
	case *agent.Module:
		a.Agent = mod
	case *executor.Executor:
		a.Executor = mod
	case []Task:
		a.Tasks = append(a.Tasks, mod...)
	}
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
