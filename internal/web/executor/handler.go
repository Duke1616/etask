package executor

import (
	"github.com/Duke1616/eiam/pkg/web/capability"
	"github.com/Duke1616/etask/internal/domain"
	executorSvc "github.com/Duke1616/etask/internal/service/executor"
	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	svc executorSvc.Service
	capability.IRegistry
}

func (h *Handler) PublicRoutes(_ *gin.Engine) {
}

func NewHandler(svc executorSvc.Service) *Handler {
	return &Handler{
		svc:       svc,
		IRegistry: capability.NewRegistry("task", "executor", "执行节点"),
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/api/executor")

	// --- 执行器管理 ---
	g.GET("/list", h.Capability("执行节点列表", "view").
		Needs("task:agent:view").
		Handle(ginx.B[ListExecutorsReq](h.ListExecutors)),
	)
}

func (h *Handler) ListExecutors(ctx *ginx.Context, req ListExecutorsReq) (ginx.Result, error) {
	res, err := h.svc.List(ctx, req.Limit, req.Cursor, req.Keyword)
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: h.toListResp(res),
		Msg:  "success",
	}, nil
}

func (h *Handler) toListResp(src domain.ExecutorList) ListExecutorsResp {
	return ListExecutorsResp{
		Executors:  lo.Map(src.Executors, func(exec domain.Executor, _ int) ExecutorVO { return h.toVO(exec) }),
		NextCursor: src.NextCursor,
		HasMore:    src.NextCursor != "",
	}
}

func (h *Handler) toVO(src domain.Executor) ExecutorVO {
	return ExecutorVO{
		Name:     src.Name,
		Desc:     src.Desc,
		Mode:     src.Mode,
		Handlers: lo.Map(src.Handlers, func(handler domain.ExecutorHandler, _ int) HandlerDetail { return h.toHandlerVO(handler) }),
		Nodes: lo.Map(src.Nodes, func(node domain.ExecutorNode, _ int) NodeDetail {
			return NodeDetail{
				ID:      node.ID,
				Address: node.Address,
			}
		}),
	}
}

func (h *Handler) toHandlerVO(src domain.ExecutorHandler) HandlerDetail {
	return HandlerDetail{
		Name:     src.Name,
		Desc:     src.Desc,
		Metadata: lo.Map(src.Metadata, func(param domain.ExecutorParameter, _ int) ParameterVO { return h.toParameterVO(param) }),
	}
}

func (h *Handler) toParameterVO(src domain.ExecutorParameter) ParameterVO {
	return ParameterVO{
		Key:      src.Key,
		Desc:     src.Desc,
		Secret:   src.Secret,
		Required: src.Required,
		Bindings: lo.MapValues(src.Bindings, func(binding domain.ExecutorBinding, _ string) BindingVO {
			return BindingVO{
				Label:       binding.Label,
				Placeholder: binding.Placeholder,
				Component:   binding.Component,
				Config:      binding.Config,
			}
		}),
		Default: src.Default,
	}
}
