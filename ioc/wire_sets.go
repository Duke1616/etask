package ioc

import (
	agentSvc "github.com/Duke1616/etask/internal/agent"
	"github.com/Duke1616/etask/internal/grpc"
	"github.com/Duke1616/etask/internal/repository"
	"github.com/Duke1616/etask/internal/repository/dao"
	codebookSvc "github.com/Duke1616/etask/internal/service/codebook"
	executorSvc "github.com/Duke1616/etask/internal/service/executor"
	runnerSvc "github.com/Duke1616/etask/internal/service/runner"
	taskSvc "github.com/Duke1616/etask/internal/service/task"
	taskBinding "github.com/Duke1616/etask/internal/service/task/binding"
	variableSvc "github.com/Duke1616/etask/internal/service/variable"
	codebookWeb "github.com/Duke1616/etask/internal/web/codebook"
	"github.com/Duke1616/etask/internal/web/executor"
	"github.com/Duke1616/etask/internal/web/manager"
	runnerWeb "github.com/Duke1616/etask/internal/web/runner"
	variableWeb "github.com/Duke1616/etask/internal/web/variable"
	"github.com/google/wire"
)

var (
	BaseSet = wire.NewSet(
		InitEtcdClient,
		InitMQ,
		InitRegistry,
	)

	WebSetup = wire.NewSet(
		InitECMDBGrpcClient,
		InitEndpointServiceClient,
		InitPolicySDK,
		InitPermSyncer,
		InitProviders,
		InitListener,
		InitGinMiddlewares,
		InitGinWebServer,
	)

	TaskSet = wire.NewSet(
		InitDB,
		dao.NewGORMTaskDAO,
		repository.NewTaskRepository,
		repository.NewTaskExecutionLogRepository,
		taskSvc.NewService,
		taskSvc.NewLogService,
		manager.NewHandler,
	)

	CodebookSet = wire.NewSet(
		dao.NewGORMCodebookDAO,
		repository.NewCodebookRepository,
		codebookSvc.NewService,
		codebookWeb.NewHandler,
	)

	RunnerSet = wire.NewSet(
		dao.NewGORMRunnerDAO,
		dao.NewGORMVariableDAO,
		InitCrypto,
		repository.NewRunnerRepository,
		runnerSvc.NewService,
		runnerWeb.NewHandler,
	)

	VariableSet = wire.NewSet(
		repository.NewVariableRepository,
		variableSvc.NewService,
		variableWeb.NewHandler,
	)

	MaterializerCoreSet = wire.NewSet(
		dao.NewGORMCodebookDAO,
		repository.NewCodebookRepository,
		codebookSvc.NewService,
		dao.NewGORMRunnerDAO,
		dao.NewGORMVariableDAO,
		InitCrypto,
		repository.NewRunnerRepository,
		runnerSvc.NewService,
		taskBinding.NewScriptBindingResolvers,
	)

	BindingResolverSet = wire.NewSet(
		taskBinding.NewScriptBindingResolvers,
	)

	ExecutorSet = wire.NewSet(
		executorSvc.NewService,
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
		agentSvc.InitWebHandler,
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
		grpc.NewCodebookServer,
		grpc.NewRunnerServer,
		InitTasks,
		InitSchedulerNodeGRPCServer,
	)
)
