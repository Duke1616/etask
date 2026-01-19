package grpc

import (
	"context"

	taskv1 "github.com/Duke1616/ework-runner/api/proto/gen/task/v1"
	"github.com/Duke1616/ework-runner/internal/domain"
	"github.com/Duke1616/ework-runner/internal/service/task"
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
