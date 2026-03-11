package web

import (
	"encoding/json"
	"fmt"

	"github.com/Duke1616/etask/internal/agent/domain"
	"github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/Duke1616/etask/pkg/grpc/registry/etcd"
	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	registry registry.Registry
}

func (h *Handler) PublicRoutes(_ *gin.Engine) {
}

func NewHandler(etcdClient *clientv3.Client) *Handler {
	reg, err := etcd.NewRegistryWithPrefix(etcdClient, "/etask/kafka")
	if err != nil {
		panic(err)
	}

	return &Handler{
		registry: reg,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/api/agent")
	g.GET("/list", ginx.W(h.ListAgents))
}

func (h *Handler) ListAgents(ctx *ginx.Context) (ginx.Result, error) {
	// 查询所有服务，通过传入空字符串查询该前缀下所有注册节点
	instances, err := h.registry.ListServices(ctx, domain.ServiceName)
	if err != nil {
		return systemErrorResult, err
	}

	fmt.Println("instance", instances)

	return ginx.Result{
		Data: h.groupExecutors(instances),
		Msg:  "success",
	}, nil
}

func (h *Handler) groupExecutors(instances []registry.ServiceInstance) []Agent {
	type aggregate struct {
		desc     string
		topic    string
		handlers []HandlerDetail
		nodes    []NodeDetail
	}

	grouped := make(map[string]*aggregate)

	for _, inst := range instances {
		if inst.Metadata == nil {
			continue
		}

		agentName := h.getStringMetadata(inst.Metadata, "name")
		if agentName == "" {
			// 如果没有 name，则 fallback 到 inst.Name
			agentName = inst.Name
		}

		agg, exists := grouped[agentName]
		if !exists {
			agg = &aggregate{}
			grouped[agentName] = agg

			// 只在第一次初始化时解析全局字段
			agg.desc = h.getStringMetadata(inst.Metadata, "desc")
			agg.topic = h.getStringMetadata(inst.Metadata, "topic")
			agg.handlers = h.parseHandlers(inst.Metadata["supported_handlers"])
		}

		// 每个实例都要加入 node
		agg.nodes = append(agg.nodes, NodeDetail{
			ID:      inst.ID,
			Address: inst.Address,
		})
	}

	// 转换为最终结构
	result := make([]Agent, 0, len(grouped))
	for name, agg := range grouped {
		result = append(result, Agent{
			Name:     name,
			Desc:     agg.desc,
			Topic:    agg.topic,
			Handlers: agg.handlers,
			Nodes:    agg.nodes,
		})
	}

	return result
}

func (h *Handler) getStringMetadata(metadata map[string]any, key string) string {
	if v, ok := metadata[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (h *Handler) parseHandlers(data any) []HandlerDetail {
	str, ok := data.(string)
	if !ok || str == "" {
		return nil
	}

	var handlers []HandlerDetail
	if err := json.Unmarshal([]byte(str), &handlers); err != nil {
		return nil
	}
	return handlers
}
