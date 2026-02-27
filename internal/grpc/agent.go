package grpc

import (
	"context"
	"time"

	executorv1 "github.com/Duke1616/ework-runner/api/proto/gen/etask/executor/v1"
	"github.com/Duke1616/ework-runner/internal/repository"
	"github.com/Duke1616/ework-runner/internal/service/task"
	"github.com/gotomicro/ego/core/elog"
)

// AgentServer 实现调度中心的 Agent 拉取服务
type AgentServer struct {
	executorv1.UnimplementedAgentServiceServer
	executorv1.UnimplementedTaskExecutionServiceServer
	execRepo repository.TaskExecutionRepository
	logSvc   task.LogService
	logger   *elog.Component
}

func NewAgentServer(execRepo repository.TaskExecutionRepository, logSvc task.LogService) *AgentServer {
	return &AgentServer{
		execRepo: execRepo,
		logSvc:   logSvc,
		logger:   elog.DefaultLogger.With(elog.FieldComponentName("grpc.AgentServer")),
	}
}

// PullTask 响应执行节点的拉取请求
func (s *AgentServer) PullTask(ctx context.Context, req *executorv1.PullTaskRequest) (*executorv1.PullTaskResponse, error) {
	serviceName := req.GetServiceName()
	nodeId := req.GetNodeId()
	if serviceName == "" {
		return &executorv1.PullTaskResponse{HasTask: false}, nil
	}
	if nodeId == "" {
		nodeId = serviceName // 向后兼容
	}

	// 1. 设置最大长轮询时间
	timeoutCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		// 在数据库中寻找并乐观抢占一条状态为 WAITING_PULL 且 Service(Group) 匹配的执行记录
		// 这里将 nodeId 真实落库记录为 executor_node_id
		exec, err := s.execRepo.ClaimPullTask(timeoutCtx, serviceName, nodeId)
		if err == nil {
			handlerName := ""
			if exec.Task.GrpcConfig != nil {
				handlerName = exec.Task.GrpcConfig.HandlerName
			}

			// 2. 抢占成功，直接将执行指令下发
			return &executorv1.PullTaskResponse{
				HasTask: true,
				TaskReq: &executorv1.ExecuteRequest{
					Eid:             exec.ID,
					TaskId:          exec.Task.ID,
					TaskName:        exec.Task.Name,
					TaskHandlerName: handlerName,
					Params:          exec.GRPCParams(),
				},
			}, nil
		}

		select {
		case <-timeoutCtx.Done():
			// 憋了 25 秒还是没有活，正常返回让客户端进入下一轮拉取
			return &executorv1.PullTaskResponse{HasTask: false}, nil
		case <-ticker.C:
			// 没有活儿，稍微等 2 秒继续尝试抢占
			continue
		}
	}
}

// ListTaskExecutions 列出任务执行记录
func (s *AgentServer) ListTaskExecutions(ctx context.Context, req *executorv1.ListTaskExecutionsRequest) (*executorv1.ListTaskExecutionsResponse, error) {
	executions, err := s.execRepo.FindByTaskID(ctx, req.GetTaskId())
	if err != nil {
		s.logger.Error("获取执行记录失败", elog.Int64("taskID", req.GetTaskId()), elog.FieldErr(err))
		return nil, err
	}

	pbExecutions := make([]*executorv1.TaskExecution, len(executions))
	for i, e := range executions {
		pbExecutions[i] = &executorv1.TaskExecution{
			Id:              e.ID,
			TaskId:          e.Task.ID,
			TaskName:        e.Task.Name,
			StartTime:       e.StartTime,
			EndTime:         e.EndTime,
			Status:          executorv1.ExecutionStatus(executorv1.ExecutionStatus_value[e.Status.String()]),
			RunningProgress: e.RunningProgress,
			ExecutorNodeId:  e.ExecutorNodeID,
		}
	}

	return &executorv1.ListTaskExecutionsResponse{
		Executions: pbExecutions,
	}, nil
}

// GetExecutionLogs 获取执行日志
func (s *AgentServer) GetExecutionLogs(ctx context.Context, req *executorv1.GetExecutionLogsRequest) (*executorv1.GetExecutionLogsResponse, error) {
	logs, err := s.logSvc.GetLogs(ctx, req.GetExecutionId(), req.GetMinId(), int(req.GetLimit()))
	if err != nil {
		s.logger.Error("获取日志失败", elog.Int64("executionID", req.GetExecutionId()), elog.FieldErr(err))
		return nil, err
	}

	pbLogs := make([]*executorv1.ExecutionLog, len(logs))
	var maxID int64
	for i, l := range logs {
		pbLogs[i] = &executorv1.ExecutionLog{
			Id:      l.ID,
			Time:    l.CTime,
			Content: l.Content,
		}
		if l.ID > maxID {
			maxID = l.ID
		}
	}

	return &executorv1.GetExecutionLogsResponse{
		Logs:  pbLogs,
		MaxId: maxID,
	}, nil
}
