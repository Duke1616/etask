package ioc

import (
	"context"

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

// Module 模块接口，所有可加载模块必须实现 Register 方法
// 替代原来的 any + type switch，让每个模块自己知道如何注册到 App 容器
type Module interface {
	Register(app *App)
}

// ModuleInitFunc 模块初始化函数，接收共享基础设施，返回 Module 实例
type ModuleInitFunc func(base *Base) Module

// modeModules 模式 → 模块初始化函数的映射表（表驱动设计）
// 新增模式只需在此表添加一行，无需修改 startServer / GetServers / StartBackgroundTasks
var modeModules = map[string][]ModuleInitFunc{
	ModeScheduler: {
		func(base *Base) Module { return InitWebModule(base) },
		func(base *Base) Module { return InitSchedulerModule(base) },
		wrapGRPCServer(InitSchedulerServerModule),
	},
	ModeExecutor: {
		wrapExecutor(InitExecutorModule),
	},
	ModeAgent: {
		wrapAgent(InitAgentModule),
	},
}

// --- 适配器：将外部包的类型包装为 Module ---
// grpcpkg.Server / agent.Module / executor.Executor 属于其他包，无法直接添加方法，
// 故用轻量 wrapper 实现 Module 接口

type grpcServerModule struct {
	server *grpcpkg.Server
}

func (m *grpcServerModule) Register(app *App) { app.Server = m.server }

type agentModuleWrapper struct {
	module *agent.Module
}

func (m *agentModuleWrapper) Register(app *App) { app.Agent = m.module }

type executorModuleWrapper struct {
	exec *executor.Executor
}

func (m *executorModuleWrapper) Register(app *App) { app.Executor = m.exec }

func wrapGRPCServer(fn func(base *Base) *grpcpkg.Server) ModuleInitFunc {
	return func(base *Base) Module { return &grpcServerModule{server: fn(base)} }
}

func wrapAgent(fn func(base *Base) *agent.Module) ModuleInitFunc {
	return func(base *Base) Module { return &agentModuleWrapper{module: fn(base)} }
}

func wrapExecutor(fn func(base *Base) *executor.Executor) ModuleInitFunc {
	return func(base *Base) Module { return &executorModuleWrapper{exec: fn(base)} }
}

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
	Registry registry.Registry
	MQ       mq.MQ
	Etcd     *clientv3.Client
}

// WebModule Web 模块资源
type WebModule struct {
	Web         *egin.Component
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

// Register 让 WebModule 自己注册到 App 容器
func (m *WebModule) Register(app *App) { app.Web = m.Web }

// Register 让 SchedulerModule 自己注册到 App 容器
func (m *SchedulerModule) Register(app *App) {
	app.Scheduler = m.Svc
	app.Tasks = append(app.Tasks, m.Tasks...)
}

// Load 加载模块到容器（通过 Module 接口，无需 any 和 type switch）
func (a *App) Load(m Module) {
	m.Register(a)
}

// LoadByModes 根据运行模式自动加载所需模块
// "all" 模式会加载所有模式的模块；其它模式仅加载对应模块
func (a *App) LoadByModes(base *Base, modes []string) {
	modeMap := a.resolveModes(modes)

	// "all" 模式：加载全部
	if modeMap[ModeAll] {
		for _, initFuncs := range modeModules {
			for _, fn := range initFuncs {
				a.Load(fn(base))
			}
		}
		return
	}

	// 指定模式：仅加载对应的模块
	for mode, initFuncs := range modeModules {
		if modeMap[mode] {
			for _, fn := range initFuncs {
				a.Load(fn(base))
			}
		}
	}
}

// GetServers 获取所有已加载模块的服务列表
// 由于 LoadByModes 只加载了当前模式需要的模块，未加载的模块为 nil，无需再做 mode 过滤
func (a *App) GetServers() []server.Server {
	var res []server.Server

	if a.Web != nil {
		res = append(res, a.Web)
	}
	if a.Server != nil {
		res = append(res, a.Server)
	}
	if a.Scheduler != nil {
		res = append(res, a.Scheduler)
	}
	if a.Agent != nil {
		res = append(res, a.Agent)
	}
	if a.Executor != nil {
		if s := a.Executor.Server(); s != nil {
			res = append(res, s)
		}
	}

	return res
}

// StartBackgroundTasks 启动所有已加载模块的后台任务
// 与 GetServers 同理：未加载的模块 Tasks 为空，无需 mode 过滤
func (a *App) StartBackgroundTasks(ctx context.Context) {
	if len(a.Tasks) > 0 {
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
