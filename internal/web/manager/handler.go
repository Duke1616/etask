package manager

import (
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/service/task"
	"github.com/Duke1616/etask/pkg/grpc/interceptors/bizid"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	svc     task.Service
	logSvc  task.LogService
	execSvc task.ExecutionService
}

func (h *Handler) PublicRoutes(_ *gin.Engine) {
}

func NewHandler(svc task.Service, logSvc task.LogService, execSvc task.ExecutionService) *Handler {
	return &Handler{
		svc:     svc,
		logSvc:  logSvc,
		execSvc: execSvc,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/api/manager")
	g.POST("/create", ginx.B[CreateTaskReq](h.Create))
	g.POST("/logs", ginx.B[GetLogsReq](h.GetLogs))
	g.POST("/executions", ginx.B[ListExecutionsReq](h.ListExecutions))
	g.POST("/list", ginx.B[PageReq](h.List))
	g.GET("/detail/:id", ginx.W(h.Detail))
	g.DELETE("/delete/:id", ginx.W(h.Delete))
	g.POST("/update", ginx.B[UpdateTaskReq](h.Update))
	g.POST("/stop/:id", ginx.W(h.Stop))
	g.POST("/run/:id", ginx.W(h.Run))
}

func (h *Handler) Detail(ctx *ginx.Context) (ginx.Result, error) {
	id, err := ctx.Param("id").AsInt64()
	if err != nil {
		return systemErrorResult, err
	}

	t, err := h.svc.GetByID(ctx, id)
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: toVO(t),
		Msg:  "success",
	}, nil
}

func (h *Handler) Delete(ctx *ginx.Context) (ginx.Result, error) {
	id, err := ctx.Param("id").AsInt64()
	if err != nil {
		return systemErrorResult, err
	}

	err = h.svc.Delete(ctx, id)
	if err != nil {
		return ginx.Result{
			Code: SystemErrorCode,
			Msg:  err.Error(),
		}, err
	}

	return ginx.Result{
		Msg: "success",
	}, nil
}

func (h *Handler) Stop(ctx *ginx.Context) (ginx.Result, error) {
	id, err := ctx.Param("id").AsInt64()
	if err != nil {
		return systemErrorResult, err
	}

	err = h.svc.Stop(ctx, id)
	if err != nil {
		return ginx.Result{
			Code: SystemErrorCode,
			Msg:  err.Error(),
		}, err
	}

	return ginx.Result{
		Msg: "success",
	}, nil
}

func (h *Handler) Run(ctx *ginx.Context) (ginx.Result, error) {
	id, err := ctx.Param("id").AsInt64()
	if err != nil {
		return systemErrorResult, err
	}

	err = h.svc.Run(ctx, id)
	if err != nil {
		return ginx.Result{
			Code: SystemErrorCode,
			Msg:  err.Error(),
		}, err
	}

	return ginx.Result{
		Msg: "success",
	}, nil
}

func (h *Handler) Update(ctx *ginx.Context, req UpdateTaskReq) (ginx.Result, error) {
	err := h.svc.Update(ctx, toUpdateDomain(req))
	if err != nil {
		return ginx.Result{
			Code: SystemErrorCode,
			Msg:  err.Error(),
		}, err
	}

	return ginx.Result{
		Msg: "success",
	}, nil
}

func (h *Handler) GetLogs(ctx *ginx.Context, req GetLogsReq) (ginx.Result, error) {
	logs, total, err := h.logSvc.GetLogs(ctx, req.ExecutionID, req.MinID, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: ListLogResp{
			Total: total,
			Logs: slice.Map(logs, func(_ int, src domain.TaskExecutionLog) TaskLogVO {
				return TaskLogVO{
					ID:          src.ID,
					TaskID:      src.TaskID,
					ExecutionID: src.ExecutionID,
					Content:     src.Content,
					CTime:       src.CTime,
				}
			}),
		},
		Msg: "success",
	}, nil
}

func (h *Handler) ListExecutions(ctx *ginx.Context, req ListExecutionsReq) (ginx.Result, error) {
	executions, total, err := h.execSvc.ListByTaskID(ctx, req.TaskID, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: ListExecutionResp{
			Total: total,
			Executions: slice.Map(executions, func(_ int, src domain.TaskExecution) TaskExecutionVO {
				return TaskExecutionVO{
					ID:              src.ID,
					TaskID:          src.Task.ID,
					TaskName:        src.Task.Name,
					StartTime:       src.StartTime,
					EndTime:         src.EndTime,
					Status:          src.Status.String(),
					RunningProgress: src.RunningProgress,
					ExecutorNodeId:  src.ExecutorNodeID,
					TaskResult:      src.TaskResult,
					CTime:           src.CTime,
				}
			}),
		},
		Msg: "success",
	}, nil
}

func (h *Handler) Create(ctx *ginx.Context, req CreateTaskReq) (ginx.Result, error) {
	create, err := h.svc.Create(ctx, toDomain(req))
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: create,
		Msg:  "success",
	}, nil
}

func (h *Handler) List(ctx *ginx.Context, req PageReq) (ginx.Result, error) {
	tasks, total, err := h.svc.List(ctx, bizid.Task, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: ListTaskResp{
			Total: total,
			Tasks: sliceMap(tasks, func(src domain.Task) TaskVO {
				return toVO(src)
			}),
		},
		Msg: "success",
	}, nil
}

func sliceMap[T, R any](data []T, f func(src T) R) []R {
	res := make([]R, 0, len(data))
	for _, v := range data {
		res = append(res, f(v))
	}
	return res
}

func toVO(src domain.Task) TaskVO {
	vo := TaskVO{
		ID:                  src.ID,
		Version:             src.Version,
		Name:                src.Name,
		Type:                src.Type.String(),
		CronExpr:            src.CronExpr,
		Status:              src.Status.String(),
		NextTime:            src.NextTime,
		MaxExecutionSeconds: src.MaxExecutionSeconds,
		ScheduleParams:      src.ScheduleParams,
		CTime:               src.CTime,
		UTime:               src.UTime,
	}

	if src.GrpcConfig != nil {
		vo.GrpcConfig = &GrpcConfig{
			ServiceName: src.GrpcConfig.ServiceName,
			HandlerName: src.GrpcConfig.HandlerName,
			Params:      src.GrpcConfig.Params,
		}
	}

	if src.HTTPConfig != nil {
		vo.HTTPConfig = &HTTPConfig{
			Endpoint: src.HTTPConfig.Endpoint,
			Params:   src.HTTPConfig.Params,
		}
	}

	if src.RetryConfig != nil {
		vo.RetryConfig = &RetryConfig{
			MaxRetries:      src.RetryConfig.MaxRetries,
			MaxInterval:     src.RetryConfig.MaxInterval,
			InitialInterval: src.RetryConfig.InitialInterval,
		}
	}

	return vo
}

func toDomain(req CreateTaskReq) domain.Task {
	t := domain.Task{
		Name:                req.Name,
		Type:                domain.TaskType(req.Type),
		CronExpr:            req.CronExpr,
		MaxExecutionSeconds: req.MaxExecutionSeconds,
		ScheduleParams:      req.ScheduleParams,
		Status:              domain.TaskStatusActive,
		Version:             1,
		BizID:               bizid.Task,
	}

	if req.GrpcConfig != nil {
		t.GrpcConfig = &domain.GrpcConfig{
			ServiceName: req.GrpcConfig.ServiceName,
			HandlerName: req.GrpcConfig.HandlerName,
			Params:      req.GrpcConfig.Params,
		}
	}

	if req.HTTPConfig != nil {
		t.HTTPConfig = &domain.HTTPConfig{
			Endpoint: req.HTTPConfig.Endpoint,
			Params:   req.HTTPConfig.Params,
		}
	}

	if req.RetryConfig != nil {
		t.RetryConfig = &domain.RetryConfig{
			MaxRetries:      req.RetryConfig.MaxRetries,
			MaxInterval:     req.RetryConfig.MaxInterval,
			InitialInterval: req.RetryConfig.InitialInterval,
		}
	}

	return t
}

func toUpdateDomain(req UpdateTaskReq) domain.Task {
	t := domain.Task{
		ID:                  req.ID,
		Version:             req.Version,
		Name:                req.Name,
		Type:                domain.TaskType(req.Type),
		CronExpr:            req.CronExpr,
		MaxExecutionSeconds: req.MaxExecutionSeconds,
		ScheduleParams:      req.ScheduleParams,
		BizID:               bizid.Task,
	}

	if req.GrpcConfig != nil {
		t.GrpcConfig = &domain.GrpcConfig{
			ServiceName: req.GrpcConfig.ServiceName,
			HandlerName: req.GrpcConfig.HandlerName,
			Params:      req.GrpcConfig.Params,
		}
	}

	if req.HTTPConfig != nil {
		t.HTTPConfig = &domain.HTTPConfig{
			Endpoint: req.HTTPConfig.Endpoint,
			Params:   req.HTTPConfig.Params,
		}
	}

	if req.RetryConfig != nil {
		t.RetryConfig = &domain.RetryConfig{
			MaxRetries:      req.RetryConfig.MaxRetries,
			MaxInterval:     req.RetryConfig.MaxInterval,
			InitialInterval: req.RetryConfig.InitialInterval,
		}
	}

	return t
}
