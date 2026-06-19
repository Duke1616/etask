package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Duke1616/etask/internal/agent/domain"
	"github.com/Duke1616/etask/internal/grpc/scripts"
	"github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/Duke1616/etask/sdk/executor"
	"github.com/ecodeclub/mq-api"
	"github.com/gotomicro/ego/core/elog"
)

type Service interface {
	// Receive 接收任务并执行
	// 返回值: (结构化结果, 原始日志, 状态, 错误)
	Receive(ctx context.Context, req domain.ExecuteReceive) (string, string, domain.Status, error)

	// ListHandlers 列出支持的任务处理器详情
	ListHandlers() []executor.HandlerMeta

	// ListAgents 列出在线代理节点
	ListAgents(ctx context.Context, limit int64, cursor string, keyword string) (domain.AgentList, error)
}

type service struct {
	mq       mq.MQ
	reg      registry.Registry
	registry *executor.HandlerRegistry
	logger   *elog.Component
}

func (s *service) ListHandlers() []executor.HandlerMeta {
	return s.registry.ListMetas()
}

func NewService(mq mq.MQ, reg registry.Registry) Service {
	s := &service{
		mq:       mq,
		reg:      reg,
		registry: executor.NewHandlerRegistry(),
		logger:   elog.DefaultLogger.With(elog.FieldComponentName("execute.service")),
	}

	s.registerHandler(scripts.GetDefaultHandlers()...)
	return s
}

func (s *service) registerHandler(handlers ...executor.TaskHandler) {
	s.registry.Register(handlers...)
}

func (s *service) ListAgents(ctx context.Context, limit int64, cursor string, keyword string) (domain.AgentList, error) {
	instances, err := s.reg.ListServices(ctx, domain.ServiceName)
	if err != nil {
		return domain.AgentList{}, err
	}
	agents := s.groupAgents(instances)
	agents = s.filterAgents(agents, strings.TrimSpace(keyword))
	return s.pageAgents(agents, normalizeLimit(limit), cursor), nil
}

func (s *service) groupAgents(instances []registry.ServiceInstance) []domain.Agent {
	type aggregate struct {
		desc     string
		topic    string
		handlers []domain.HandlerDetail
		nodes    []domain.NodeDetail
	}

	grouped := make(map[string]*aggregate)
	for _, inst := range instances {
		if inst.Metadata == nil {
			continue
		}

		agentName := s.getStringMetadata(inst.Metadata, "name")
		if agentName == "" {
			agentName = inst.Name
		}

		agg, exists := grouped[agentName]
		if !exists {
			agg = &aggregate{
				desc:     s.getStringMetadata(inst.Metadata, "desc"),
				topic:    s.getStringMetadata(inst.Metadata, "topic"),
				handlers: s.parseHandlers(inst.Metadata["supported_handlers"]),
			}
			grouped[agentName] = agg
		}

		agg.nodes = append(agg.nodes, domain.NodeDetail{
			ID:      inst.ID,
			Address: inst.Address,
		})
	}

	agents := make([]domain.Agent, 0, len(grouped))
	for name, agg := range grouped {
		agents = append(agents, domain.Agent{
			Name:     name,
			Desc:     agg.desc,
			Topic:    agg.topic,
			Handlers: agg.handlers,
			Nodes:    agg.nodes,
		})
	}
	return agents
}

func (s *service) filterAgents(agents []domain.Agent, keyword string) []domain.Agent {
	if keyword == "" {
		return agents
	}
	keyword = strings.ToLower(keyword)
	res := make([]domain.Agent, 0, len(agents))
	for _, agent := range agents {
		if s.matchAgent(agent, keyword) {
			res = append(res, agent)
		}
	}
	return res
}

func (s *service) matchAgent(agent domain.Agent, keyword string) bool {
	if strings.Contains(strings.ToLower(agent.Name), keyword) ||
		strings.Contains(strings.ToLower(agent.Desc), keyword) ||
		strings.Contains(strings.ToLower(agent.Topic), keyword) {
		return true
	}
	for _, handler := range agent.Handlers {
		if strings.Contains(strings.ToLower(handler.Name), keyword) ||
			strings.Contains(strings.ToLower(handler.Desc), keyword) {
			return true
		}
	}
	return false
}

func (s *service) pageAgents(agents []domain.Agent, limit int64, cursor string) domain.AgentList {
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})

	start := 0
	if cursor != "" {
		start = sort.Search(len(agents), func(i int) bool {
			return agents[i].Name > cursor
		})
	}

	end := start + int(limit)
	if end > len(agents) {
		end = len(agents)
	}

	nextCursor := ""
	if end < len(agents) && end > start {
		nextCursor = agents[end-1].Name
	}

	return domain.AgentList{
		Agents:     agents[start:end],
		NextCursor: nextCursor,
	}
}

func normalizeLimit(limit int64) int64 {
	if limit <= 0 {
		return 20
	}
	return limit
}

func (s *service) getStringMetadata(metadata map[string]any, key string) string {
	v, ok := metadata[key]
	if !ok {
		return ""
	}
	res, _ := v.(string)
	return res
}

func (s *service) parseHandlers(data any) []domain.HandlerDetail {
	str, ok := data.(string)
	if !ok || str == "" {
		return nil
	}

	var handlers []domain.HandlerDetail
	if err := json.Unmarshal([]byte(str), &handlers); err != nil {
		return nil
	}
	return handlers
}

func (s *service) Receive(ctx context.Context, req domain.ExecuteReceive) (string, string, domain.Status, error) {
	// 1. 查找处理器 (优先使用 Handler，如果没有则退化到使用 Language)
	handlerName := req.Handler
	if handlerName == "" {
		handlerName = req.Language
	}

	h, ok := s.registry.Get(handlerName)
	if !ok {
		return "", "", domain.FAILED, fmt.Errorf("未找到对应的任务处理器: %s", handlerName)
	}

	// 2. 准备输出捕获器
	capLogger := &captureLogger{}

	// 3. 构建参数 map (将分散的各个属性聚合为 Handler 认识的 Context 参数)
	params := map[string]string{
		"code":      req.Code,
		"args":      req.Args,
		"variables": req.Variables,
	}

	// 4. 创建 Context
	taskCtx := executor.NewContextWithLogger(
		ctx,
		0, // Kafka 模式下暂无 ExecutionID
		req.TaskId,
		"kafka_task", // 任务名称
		handlerName,
		params,
		s.logger,
		capLogger,
	)
	defer taskCtx.Close()

	// 5. 调用 Handler 执行
	err := h.Run(taskCtx)

	// 6. 整理汇总结果
	result := taskCtx.GetResultJson() // 这是来自 FD 3 的纯净数据
	output := capLogger.GetContent()  // 这是来自 Stdout 的执行日志

	if err != nil {
		return result, output, domain.FAILED, err
	}

	return result, output, domain.SUCCESS, nil
}

// captureLogger 内部实现，用于捕获来自 Handler 的 Log 调用并存入 string 汇总
type captureLogger struct {
	buffer strings.Builder
}

func (c *captureLogger) Log(format string, args ...any) {
	c.buffer.WriteString(fmt.Sprintf(format, args...))
	c.buffer.WriteByte('\n')
}

func (c *captureLogger) Close() {}

func (c *captureLogger) GetContent() string {
	return c.buffer.String()
}
