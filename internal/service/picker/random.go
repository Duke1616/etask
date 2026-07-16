package picker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/pkg/grpc/registry"
)

var _ ExecutorNodePicker = &RandomPicker{}

// RandomPicker 基础随机选择器，职责：从可用节点中随机选出一个节点 ID
type RandomPicker struct {
	reg registry.Registry
	rnd *rand.Rand
}

// Pick 从支持任务 Handler 的在线 Executor 中随机选择节点。
func (b *RandomPicker) Pick(ctx context.Context, task domain.Task) (string, error) {
	if task.GrpcConfig == nil {
		return "", fmt.Errorf("任务缺少 gRPC 配置")
	}
	services, err := b.reg.ListServices(ctx, task.GrpcConfig.ServiceName)
	if err != nil {
		return "", fmt.Errorf("获取执行节点列表失败: %w", err)
	}
	candidates := make([]string, 0, len(services))
	for _, service := range services {
		if !supportsTask(service, task.GrpcConfig.HandlerName) || !supportsDispatchMode(service, task.ExecMode) {
			continue
		}
		nodeID := strings.TrimSpace(service.ID)
		if nodeID == "" {
			continue
		}
		candidates = append(candidates, nodeID)
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("没有支持处理器 %s 的可用执行节点", task.GrpcConfig.HandlerName)
	}
	return candidates[b.rnd.Intn(len(candidates))], nil
}

// supportsDispatchMode 只做节点过滤，派发模式仍以已持久化的资源池快照为准。
func supportsDispatchMode(service registry.ServiceInstance, expected domain.ExecMode) bool {
	actual := strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%v", service.Metadata["mode"])))
	if actual == "" || actual == "<NIL>" {
		actual = domain.ExecModePush.String()
	}
	return actual == expected.String()
}

func NewRandomPicker(reg registry.Registry) ExecutorNodePicker {
	return &RandomPicker{
		reg: reg,
		rnd: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func supportsTask(service registry.ServiceInstance, handlerName string) bool {
	if strings.TrimSpace(fmt.Sprintf("%v", service.Metadata["role"])) != "executor" {
		return false
	}
	raw := strings.TrimSpace(fmt.Sprintf("%v", service.Metadata["supported_handlers"]))
	if raw == "" || raw == "<nil>" {
		return false
	}
	var handlers []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(raw), &handlers); err != nil {
		return false
	}
	for _, handler := range handlers {
		if handler.Name == handlerName {
			return true
		}
	}
	return false
}
