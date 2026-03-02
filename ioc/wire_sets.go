package ioc

import (
	agentSvc "github.com/Duke1616/etask/internal/agent"
	"github.com/Duke1616/etask/internal/grpc"
	"github.com/Duke1616/etask/internal/repository"
	"github.com/Duke1616/etask/internal/repository/dao"
	taskSvc "github.com/Duke1616/etask/internal/service/task"
	"github.com/Duke1616/etask/internal/web/agent"
	"github.com/Duke1616/etask/internal/web/executor"
	"github.com/Duke1616/etask/internal/web/task"
	"github.com/Duke1616/etask/pkg/ginx/middleware"
	"github.com/google/wire"
)

var (
	BaseSet = wire.NewSet(
		InitDB,
		InitRedis,
		InitDistributedLock,
		InitEtcdClient,
		InitMQ,
		InitRunner,
		InitInvoker,
		InitRegistry,
	)

	WebSetup = wire.NewSet(
		InitECMDBGrpcClient,
		InitPolicyServiceClient,
		InitEndpointServiceClient,
		middleware.NewCheckPolicyMiddlewareBuilder,
		InitSession,
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
		agent.NewHandler,
		agentSvc.InitModule,
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
