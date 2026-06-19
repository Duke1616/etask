package web

import (
	"github.com/Duke1616/eiam/pkg/web/capability"
	"github.com/Duke1616/etask/internal/agent/domain"
	agentSvc "github.com/Duke1616/etask/internal/agent/service"
	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	svc agentSvc.Service
	capability.IRegistry
}

func (h *Handler) PublicRoutes(_ *gin.Engine) {
}

func NewHandler(svc agentSvc.Service) *Handler {
	return &Handler{
		svc:       svc,
		IRegistry: capability.NewRegistry("task", "agent", "代理管理"),
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/api/agent")

	// --- 代理管理 ---
	g.GET("/list", h.Capability("代理列表", "view").
		NoSync().
		Handle(ginx.B[ListAgentsReq](h.ListAgents)),
	)
}

func (h *Handler) ListAgents(ctx *ginx.Context, req ListAgentsReq) (ginx.Result, error) {
	res, err := h.svc.ListAgents(ctx, req.Limit, req.Cursor, req.Keyword)
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: h.toListResp(res),
		Msg:  "success",
	}, nil
}

func (h *Handler) toListResp(src domain.AgentList) ListAgentsResp {
	return ListAgentsResp{
		Agents:     lo.Map(src.Agents, func(agent domain.Agent, _ int) Agent { return h.toVO(agent) }),
		NextCursor: src.NextCursor,
		HasMore:    src.NextCursor != "",
	}
}

func (h *Handler) toVO(src domain.Agent) Agent {
	return Agent{
		Name:     src.Name,
		Desc:     src.Desc,
		Topic:    src.Topic,
		Handlers: lo.Map(src.Handlers, func(handler domain.HandlerDetail, _ int) HandlerDetail { return HandlerDetail(handler) }),
		Nodes: lo.Map(src.Nodes, func(node domain.NodeDetail, _ int) NodeDetail {
			return NodeDetail{
				ID:      node.ID,
				Address: node.Address,
			}
		}),
	}
}
