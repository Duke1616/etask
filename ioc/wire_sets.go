package ioc

import (
	"github.com/Duke1616/ecmdb/pkg/policy"
	agentSvc "github.com/Duke1616/etask/internal/agent"
	"github.com/Duke1616/etask/internal/agent/web"
	"github.com/Duke1616/etask/internal/grpc"
	"github.com/Duke1616/etask/internal/repository"
	"github.com/Duke1616/etask/internal/repository/dao"
	taskSvc "github.com/Duke1616/etask/internal/service/task"
	"github.com/Duke1616/etask/internal/web/executor"
	"github.com/Duke1616/etask/internal/web/task"
	"github.com/google/wire"
)

var (
	BaseSet = wire.NewSet(
		InitDB,
		InitRedis,
		InitDistributedLock,
		InitEtcdClient,
		InitMQ,
		InitRegistry,
	)

	WebSetup = wire.NewSet(
		InitECMDBGrpcClient,
		InitEndpointServiceClient,
		policy.NewSDK,
		InitListener,
		InitGinMiddlewares,
		InitGinWebServer,
	)

	TaskSet = wire.NewSet(
		dao.NewGORMTaskDAO,
		repository.NewTaskRepository,
		taskSvc.NewService,
		taskSvc.NewLogService,
		task.NewHandler,
	)

	ExecutorSet = wire.NewSet(
		executor.NewHandler,
	)

	TaskExecutionSet = wire.NewSet(
		dao.NewGORMTaskExecutionDAO,
		dao.NewGORMTaskExecutionLogDAO,
		repository.NewTaskExecutionRepository,
		taskSvc.NewExecutionService,
	)

	SchedulerSet = wire.NewSet(
		InitNodeID,
		InitScheduler,
		InitMySQLTaskAcquirer,
		InitExecutorNodePicker,
		InitExecModeResolver,
	)

	AgentSet = wire.NewSet(
		agentSvc.InitModule,
		wire.FieldsOf(new(*agentSvc.Module), "Hdl"),
	)

	// AgentWebSet 专门用于 Scheduler 等只需要查看 Agent 状态而不运行 Agent 的场景
	AgentWebSet = wire.NewSet(
		web.NewHandler,
	)

	CompensatorSet = wire.NewSet(
		InitRetryCompensator,
		InitRescheduleCompensator,
		InitInterruptCompensator,
	)

	ProducerSet = wire.NewSet(
		InitCompleteProducer,
	)

	GrpcSet = wire.NewSet(
		InitExecutorServiceGRPCClients,
	)

	ConsumerSet = wire.NewSet(
		InitCompleteEventConsumer,
	)

	// AppSet 包含 Scheduler 模式的核心 Provider
	AppSet = wire.NewSet(
		grpc.NewReporterServer,
		grpc.NewTaskServer,
		grpc.NewAgentServer,
		InitTasks,
		InitSchedulerNodeGRPCServer,
	)
)
