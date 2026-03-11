package grpc

import (
	"context"

	taskv1 "github.com/Duke1616/etask/api/proto/gen/etask/task/v1"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/service/task"
	"github.com/gotomicro/ego/core/elog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TaskServer TaskService gRPC服务实现
type TaskServer struct {
	taskv1.UnimplementedTaskServiceServer
	taskSvc task.Service
	logger  *elog.Component
}

// NewTaskServer 创建 TaskServer 实例
func NewTaskServer(taskSvc task.Service) *TaskServer {
	return &TaskServer{
		taskSvc: taskSvc,
		logger:  elog.DefaultLogger.With(elog.FieldComponentName("grpc.TaskServer")),
	}
}

// CreateTask 创建任务
func (s *TaskServer) CreateTask(ctx context.Context, req *taskv1.CreateTaskRequest) (*taskv1.CreateTaskResponse, error) {
	s.logger.Info("收到创建任务请求",
		elog.String("name", req.GetName()),
		elog.String("type", req.GetType().String()),
		elog.String("cronExpr", req.GetCronExpr()))

	// 将 protobuf 请求转换为 domain 对象
	domainTask := s.toDomainTask(req)

	// 调用业务服务创建任务
	createdTask, err := s.taskSvc.Create(ctx, domainTask)
	if err != nil {
		s.logger.Error("创建任务失败",
			elog.String("name", req.GetName()),
			elog.FieldErr(err))
		return nil, status.Error(codes.Internal, "创建任务失败")
	}

	return &taskv1.CreateTaskResponse{
		Id: createdTask.ID,
	}, nil
}

// toDomainTask 将 protobuf CreateTaskRequest 转换为 domain.Task
func (s *TaskServer) toDomainTask(req *taskv1.CreateTaskRequest) domain.Task {
	return domain.Task{
		Name:                req.GetName(),
		Type:                s.toDomainTaskType(req.GetType()),
		CronExpr:            req.GetCronExpr(),
		MaxExecutionSeconds: req.GetMaxExecutionSeconds(),
		ScheduleParams:      req.GetScheduleParams(),
		GrpcConfig: &domain.GrpcConfig{
			ServiceName: req.GrpcConfig.GetServiceName(),
			AuthToken:   req.GrpcConfig.GetAuthToken(),
			HandlerName: req.GrpcConfig.GetHandlerName(),
			Params:      req.GrpcConfig.GetParams(),
		},
		HTTPConfig: &domain.HTTPConfig{
			Endpoint: req.HttpConfig.GetEndpoint(),
			Params:   req.HttpConfig.GetParams(),
		},
		RetryConfig: &domain.RetryConfig{
			MaxRetries:      req.RetryConfig.GetMaxRetries(),
			InitialInterval: req.RetryConfig.GetInitialInterval(),
			MaxInterval:     req.RetryConfig.GetMaxInterval(),
		},
		Status:  domain.TaskStatusActive,
		Version: 1,
	}
}

// toDomainTaskType 将 protobuf TaskType 转换为 domain.TaskType
func (s *TaskServer) toDomainTaskType(t taskv1.TaskType) domain.TaskType {
	switch t {
	case taskv1.TaskType_RECURRING:
		return domain.TaskTypeRecurring
	case taskv1.TaskType_ONE_TIME:
		return domain.TaskTypeOneTime
	default:
		return domain.TaskTypeRecurring
	}
}
func (s *TaskServer) RetryTask(ctx context.Context, req *taskv1.RetryTaskRequest) (*taskv1.RetryTaskResponse, error) {
	s.logger.Info("收到重试任务请求", elog.Int64("id", req.GetId()))

	_, err := s.taskSvc.Retry(ctx, req.GetId())
	if err != nil {
		s.logger.Error("重试任务失败",
			elog.Int64("id", req.GetId()),
			elog.FieldErr(err))
		return nil, status.Error(codes.Internal, "重试任务失败: "+err.Error())
	}

	return &taskv1.RetryTaskResponse{}, nil
}

// GetTask 获取任务
func (s *TaskServer) GetTask(ctx context.Context, req *taskv1.GetTaskRequest) (*taskv1.GetTaskResponse, error) {
	t, err := s.taskSvc.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "未找到该任务")
	}

	return &taskv1.GetTaskResponse{
		Task: s.toProtoTask(t),
	}, nil
}

// toProtoTask 将 domain.Task 转换为 protobuf Task
func (s *TaskServer) toProtoTask(t domain.Task) *taskv1.Task {
	return &taskv1.Task{
		Id:                  t.ID,
		Name:                t.Name,
		Type:                s.toProtoTaskType(t.Type),
		CronExpr:            t.CronExpr,
		MaxExecutionSeconds: t.MaxExecutionSeconds,
		ScheduleNodeId:      t.ScheduleNodeID,
		ScheduleParams:      t.ScheduleParams,
		NextTime:            t.NextTime,
		Status:              s.toProtoTaskStatus(t.Status),
		Version:             t.Version,
		Ctime:               t.CTime,
		Utime:               t.UTime,
		ExecMode:            t.ExecMode.ToProto(),
		GrpcConfig:          s.toProtoGrpcConfig(t.GrpcConfig),
		HttpConfig:          s.toProtoHTTPConfig(t.HTTPConfig),
		RetryConfig:         s.toProtoRetryConfig(t.RetryConfig),
	}
}

func (s *TaskServer) toProtoGrpcConfig(cfg *domain.GrpcConfig) *taskv1.GrpcConfig {
	if cfg == nil {
		return nil
	}
	return &taskv1.GrpcConfig{
		ServiceName: cfg.ServiceName,
		AuthToken:   cfg.AuthToken,
		HandlerName: cfg.HandlerName,
		Params:      cfg.Params,
	}
}

func (s *TaskServer) toProtoHTTPConfig(cfg *domain.HTTPConfig) *taskv1.HTTPConfig {
	if cfg == nil {
		return nil
	}
	return &taskv1.HTTPConfig{
		Endpoint: cfg.Endpoint,
		Params:   cfg.Params,
	}
}

func (s *TaskServer) toProtoRetryConfig(cfg *domain.RetryConfig) *taskv1.RetryConfig {
	if cfg == nil {
		return nil
	}
	return &taskv1.RetryConfig{
		MaxRetries:      cfg.MaxRetries,
		InitialInterval: cfg.InitialInterval,
		MaxInterval:     cfg.MaxInterval,
	}
}

// toProtoTaskType 将 domain.TaskType 转换为 protobuf TaskType
func (s *TaskServer) toProtoTaskType(t domain.TaskType) taskv1.TaskType {
	switch t {
	case domain.TaskTypeRecurring:
		return taskv1.TaskType_RECURRING
	case domain.TaskTypeOneTime:
		return taskv1.TaskType_ONE_TIME
	default:
		return taskv1.TaskType_TASK_TYPE_UNSPECIFIED
	}
}

// toProtoTaskStatus 将 domain.TaskStatus 转换为 protobuf TaskStatus
func (s *TaskServer) toProtoTaskStatus(t domain.TaskStatus) taskv1.TaskStatus {
	switch t {
	case domain.TaskStatusActive:
		return taskv1.TaskStatus_ACTIVE
	case domain.TaskStatusPreempted:
		return taskv1.TaskStatus_PREEMPTED
	case domain.TaskStatusInactive:
		return taskv1.TaskStatus_INACTIVE
	case domain.TaskStatusCompleted:
		return taskv1.TaskStatus_COMPLETED
	default:
		return taskv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}
