package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	executorv1 "github.com/Duke1616/etask/api/proto/gen/etask/executor/v1"
	reporterv1 "github.com/Duke1616/etask/api/proto/gen/etask/reporter/v1"
	grpcpkg "github.com/Duke1616/etask/pkg/grpc"
	"github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/ecodeclub/ekit/syncx"
	"github.com/gotomicro/ego/core/elog"
)

const RoleName = "executor"

type Config struct {
	Mode   string // 执行模式: "PUSH" (默认) 或 "PULL"
	Desc   string // 执行器的全局描述
	Server grpcpkg.ServerConfig
	Client grpcpkg.ClientConfig
}

// Executor 极简 Executor 实现
type Executor struct {
	executorv1.UnimplementedExecutorServiceServer

	config   Config
	registry registry.Registry
	hr       *HandlerRegistry

	// 内部组件
	server         *grpcpkg.Server
	reporterClient reporterv1.ReporterServiceClient
	agentClient    executorv1.AgentServiceClient
	logger         *elog.Component

	// 状态管理 - 使用 syncx.Map
	states  *syncx.Map[int64, *executorv1.ExecutionState]
	cancels *syncx.Map[int64, context.CancelFunc]
}

// NewExecutor 创建 Executor
func NewExecutor(cfg Config, reg registry.Registry) (*Executor, error) {
	if err := cfg.Server.Validate(); err != nil {
		return nil, err
	}

	if cfg.Server.ServiceId == "" {
		return nil, errors.New("service_id is required")
	}

	return &Executor{
		config:   cfg,
		registry: reg,
		hr:       NewHandlerRegistry(),
		logger:   elog.DefaultLogger.With(elog.FieldComponentName("executor")),
		states:   &syncx.Map[int64, *executorv1.ExecutionState]{},
		cancels:  &syncx.Map[int64, context.CancelFunc]{},
	}, nil
}

// RegisterHandler 注册任务处理函数
// name: 任务名称,需要与调度中心下发的 taskName 匹配
// RegisterHandler 注册任务处理函数
func (e *Executor) RegisterHandler(handlers ...TaskHandler) *Executor {
	e.hr.Register(handlers...)
	return e
}

// InitComponents 初始化组件
func (e *Executor) InitComponents() error {
	// 1. 连接 Reporter - 使用封装的 NewClientConn
	reporterConn, err := grpcpkg.NewClientConn(
		e.registry,
		grpcpkg.WithServiceName(e.config.Client.Name),
		grpcpkg.WithClientJWTAuth(e.config.Client.AuthToken),
		grpcpkg.WithTimeout(10*time.Second),
	)
	if err != nil {
		return fmt.Errorf("连接 reporter 失败: %w", err)
	}

	e.reporterClient = reporterv1.NewReporterServiceClient(reporterConn)

	// 2. 创建 gRPC Server
	e.server = grpcpkg.NewServer(
		e.config.Server,
		e.registry,
		grpcpkg.WithJWTAuth(e.config.Server.AuthToken),
		grpcpkg.WithMetadata(e.buildMetadata()),
	)

	// 3. 初始化 Agent 拉取客户端 (复用连接因为它是给同一调度中心汇报)
	e.agentClient = executorv1.NewAgentServiceClient(reporterConn)

	// 4. 注册 Agent 服务
	executorv1.RegisterExecutorServiceServer(e.server.Server, e)

	// 5. 判断模式开启拉取节点
	if e.config.Mode == "PULL" {
		go e.startPullLoop()
	}

	return nil
}

func (e *Executor) startPullLoop() {
	e.logger.Info("启动在 PULL 模式，开始请求中心调度获取任务...")

	for {
		// 收集目前支持的 handlers
		var supported []string
		for name := range e.hr.Handlers() {
			supported = append(supported, name)
		}

		// 增加客户端长轮询的超时控制（设置稍微长于服务端的 25 秒）
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		resp, err := e.agentClient.PullTask(ctx, &executorv1.PullTaskRequest{
			ServiceName: e.config.Server.ServiceName, // 分组名（比如 ework-executor-node1）
			NodeId:      e.config.Server.ServiceId,   // 这台机器确切的主机/实例UUID
			Handlers:    supported,                   // 我能处理这些：这层能帮助中心更精确发活
		})
		cancel()

		if err != nil {
			// 可能调度中心挂了或网络抖动
			e.logger.Warn("拉取任务失败", elog.FieldErr(err))
			time.Sleep(time.Second)
			continue
		}

		if resp != nil && resp.HasTask && resp.TaskReq != nil {
			e.logger.Info("成功拉取到后台派发的远程指令", elog.Int64("eid", resp.TaskReq.Eid))
			// 直接模拟收到了一次本地 grpc push 调度请求（将两端逻辑完美融合复用！）
			_, _ = e.Execute(context.Background(), resp.TaskReq)
		}
	}
}

func (e *Executor) buildMetadata() map[string]any {
	metas := e.hr.ListMetas()
	bytes, _ := json.Marshal(metas)
	return map[string]any{
		"role":               RoleName,      // 标识此注册节点为调度引擎的执行器
		"desc":               e.config.Desc, // 执行器集群总体功能描述
		"supported_handlers": string(bytes), // 支持的任务处理器列表
		"mode":               e.config.Mode, // 执行模式: PUSH 或 PULL，供调度中心感知
	}
}

// Server 获取内部 gRPC Server (用于 ego 启动)
func (e *Executor) Server() *grpcpkg.Server {
	return e.server
}

// Execute 实现 ExecutorServiceServer.Execute
func (e *Executor) Execute(ctx context.Context, req *executorv1.ExecuteRequest) (*executorv1.ExecuteResponse, error) {
	eid := req.GetEid()

	// 检查是否已经在执行
	if state, ok := e.states.Load(eid); ok {
		e.logger.Warn("任务已在执行中", elog.Int64("eid", eid))
		return &executorv1.ExecuteResponse{ExecutionState: state}, nil
	}

	// 创建初始状态
	state := &executorv1.ExecutionState{
		Id:              eid,
		TaskId:          req.GetTaskId(),
		TaskName:        req.GetTaskName(),
		Status:          executorv1.ExecutionStatus_RUNNING,
		RunningProgress: 0,
		ExecutorNodeId:  e.config.Server.ServiceId,
	}
	e.states.Store(eid, state)

	// 创建任务上下文
	taskCtx := NewContext(eid, req.GetTaskId(), req.GetTaskName(), req.GetTaskHandlerName(),
		req.GetParams(), e.reporterClient, e.logger)

	//创建可取消上下文
	runCtx, cancel := context.WithCancel(context.Background())
	e.cancels.Store(eid, cancel)

	e.logger.Info("启动异步任务执行", elog.Int64("eid", eid))
	// 异步执行任务
	go e.executeTask(runCtx, taskCtx, eid)

	return &executorv1.ExecuteResponse{ExecutionState: state}, nil
}

// executeTask 执行用户任务
func (e *Executor) executeTask(runCtx context.Context, taskCtx *Context, eid int64) {
	defer func() {
		e.cancels.Delete(eid)
	}()

	logger := taskCtx.Logger()

	// 查找处理函数
	defer taskCtx.Close() // 确保日志被发送

	handler, exists := e.hr.Get(taskCtx.HandlerName)

	var err error
	if !exists {
		err = fmt.Errorf("未找到任务处理器: %s", taskCtx.TaskName)
	} else {
		// 调用用户处理函数
		err = handler.Run(taskCtx)
	}

	// 确定最终状态
	var finalStatus executorv1.ExecutionStatus
	if runCtx.Err() != nil {
		finalStatus = executorv1.ExecutionStatus_FAILED_RESCHEDULABLE
		logger.Warn("任务被中断")
	} else if err != nil {
		finalStatus = executorv1.ExecutionStatus_FAILED
		logger.Error("任务执行失败", elog.FieldErr(err))
	} else {
		finalStatus = executorv1.ExecutionStatus_SUCCESS
		logger.Info("任务执行成功")
	}

	// 获取任务结果
	taskResult := taskCtx.GetResultJson()

	// 如果执行失败且没有设置结果，则自动将错误信息作为结果上报
	if err != nil && taskResult == "" {
		taskResult = err.Error()
	}

	// 更新并上报最终状态
	e.reportFinalResult(eid, finalStatus, taskResult)
}

// reportFinalResult 上报最终结果
func (e *Executor) reportFinalResult(eid int64, status executorv1.ExecutionStatus, taskResult string) {
	state, exists := e.states.Load(eid)
	if exists {
		state.Status = status
		if status == executorv1.ExecutionStatus_SUCCESS {
			state.RunningProgress = 100
		}
		// 设置任务结果
		state.TaskResult = taskResult
		e.states.Store(eid, state)

		// 上报给 Reporter
		_, err := e.reporterClient.Report(context.Background(), &reporterv1.ReportRequest{
			ExecutionState: state,
		})
		if err != nil {
			e.logger.Error("上报最终状态失败", elog.FieldErr(err))
		}
	}
}

// Query 实现 ExecutorServiceServer.Query
func (e *Executor) Query(ctx context.Context, req *executorv1.QueryRequest) (*executorv1.QueryResponse, error) {
	eid := req.GetEid()

	if state, ok := e.states.Load(eid); ok {
		return &executorv1.QueryResponse{ExecutionState: state}, nil
	}

	return &executorv1.QueryResponse{
		ExecutionState: &executorv1.ExecutionState{
			Id:     eid,
			Status: executorv1.ExecutionStatus_UNKNOWN,
		},
	}, nil
}

// Interrupt 实现 ExecutorServiceServer.Interrupt
func (e *Executor) Interrupt(ctx context.Context, req *executorv1.InterruptRequest) (*executorv1.InterruptResponse, error) {
	eid := req.GetEid()

	if cancel, ok := e.cancels.Load(eid); ok {
		cancel()

		if state, exist := e.states.Load(eid); exist {
			return &executorv1.InterruptResponse{
				Success:        true,
				ExecutionState: state,
			}, nil
		}
	}

	return &executorv1.InterruptResponse{Success: false}, nil
}

// Prepare 实现 ExecutorServiceServer.Prepare
func (e *Executor) Prepare(ctx context.Context, req *executorv1.PrepareRequest) (*executorv1.PrepareResponse, error) {
	return &executorv1.PrepareResponse{
		Params: make(map[string]string),
	}, nil
}
