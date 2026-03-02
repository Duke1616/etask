package executor

import (
	"encoding/json"

	"github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/Duke1616/etask/sdk/executor"
	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	registry registry.Registry
}

func (h *Handler) PublicRoutes(_ *gin.Engine) {
}

func NewHandler(reg registry.Registry) *Handler {
	return &Handler{
		registry: reg,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/api/executor")
	g.GET("/list", ginx.W(h.ListExecutors))
}

func (h *Handler) ListExecutors(ctx *ginx.Context) (ginx.Result, error) {
	// 查询所有服务，通过传入空字符串查询该前缀下所有注册节点
	instances, err := h.registry.ListServices(ctx, "")
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: h.groupExecutors(instances),
		Msg:  "success",
	}, nil
}

func (h *Handler) groupExecutors(instances []registry.ServiceInstance) []Executor {
	groupedNodes := make(map[string][]NodeDetail)
	groupedHandlers := make(map[string][]HandlerDetail)
	groupedDesc := make(map[string]string)
	groupedMode := make(map[string]string)

	for _, inst := range instances {
		if inst.Metadata == nil {
			continue
		}

		// 过滤掉非 executor 节点
		if role, ok := inst.Metadata["role"]; !ok || role != executor.RoleName {
			continue
		}

		// 提取节点运行模式
		if modeRaw, ok := inst.Metadata["mode"]; ok {
			if modeStr, ok := modeRaw.(string); ok {
				groupedMode[inst.Name] = modeStr
			}
		}

		// 提取节点全局描述
		if descRaw, ok := inst.Metadata["desc"]; ok {
			if descStr, ok := descRaw.(string); ok {
				groupedDesc[inst.Name] = descStr
			}
		}

		// 获取 Handlers 一次（每个节点的都应是相同的）
		if _, exists := groupedHandlers[inst.Name]; !exists {
			groupedHandlers[inst.Name] = h.parseHandlers(inst.Metadata["supported_handlers"])
		}

		groupedNodes[inst.Name] = append(groupedNodes[inst.Name], NodeDetail{
			ID:      inst.ID,
			Address: inst.Address,
		})
	}

	executors := make([]Executor, 0, len(groupedNodes))
	for name, nodes := range groupedNodes {
		executors = append(executors, Executor{
			Name:     name,
			Desc:     groupedDesc[name],
			Mode:     groupedMode[name],
			Handlers: groupedHandlers[name],
			Nodes:    nodes,
		})
	}

	return executors
}

func (h *Handler) parseHandlers(data any) []HandlerDetail {
	var handlers []HandlerDetail
	if bytes, ok := data.(string); ok {
		_ = json.Unmarshal([]byte(bytes), &handlers)
	}
	return handlers
}
