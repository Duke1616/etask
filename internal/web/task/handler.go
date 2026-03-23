package task

import (
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/service/task"
	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	svc    task.Service
	logSvc task.LogService
}

func (h *Handler) PublicRoutes(_ *gin.Engine) {
}

func NewHandler(svc task.Service, logSvc task.LogService) *Handler {
	return &Handler{
		svc:    svc,
		logSvc: logSvc,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/api/task")
	g.POST("/create", ginx.B[CreateTaskReq](h.Create))
	g.POST("/logs", ginx.B[GetLogsReq](h.GetLogs))
	g.POST("/list", ginx.B[PageReq](h.List))
}

func (h *Handler) GetLogs(ctx *ginx.Context, req GetLogsReq) (ginx.Result, error) {
	logs, err := h.logSvc.GetLogs(ctx, req.ExecutionID, req.MinID, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: logs,
		Msg:  "success",
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
	tasks, total, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: ListTaskResp{
			Total: total,
			Tasks: sliceMap(tasks, func(src domain.Task) TaskVO {
				return TaskVO{
					ID:                  src.ID,
					Name:                src.Name,
					Type:                src.Type.String(),
					CronExpr:            src.CronExpr,
					Status:              src.Status.String(),
					NextTime:            src.NextTime,
					MaxExecutionSeconds: src.MaxExecutionSeconds,
				}
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

func toDomain(req CreateTaskReq) domain.Task {
	return domain.Task{
		Name:                req.Name,
		Type:                domain.TaskType(req.Type),
		CronExpr:            req.CronExpr,
		MaxExecutionSeconds: req.MaxExecutionSeconds,
		ScheduleParams:      req.ScheduleParams,
		GrpcConfig: &domain.GrpcConfig{
			ServiceName: req.GrpcConfig.ServiceName,
			AuthToken:   req.GrpcConfig.AuthToken,
			HandlerName: req.GrpcConfig.HandlerName,
			Params:      req.GrpcConfig.Params,
		},
		HTTPConfig: &domain.HTTPConfig{
			Endpoint: req.HTTPConfig.Endpoint,
			Params:   req.HTTPConfig.Params,
		},
		RetryConfig: &domain.RetryConfig{
			MaxRetries:      req.RetryConfig.MaxRetries,
			MaxInterval:     req.RetryConfig.MaxInterval,
			InitialInterval: req.RetryConfig.InitialInterval,
		},
		Status:  domain.TaskStatusActive,
		Version: 1,
	}
}
