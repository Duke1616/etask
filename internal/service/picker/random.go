package picker

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Duke1616/ework-runner/internal/domain"
	"github.com/Duke1616/ework-runner/pkg/grpc/registry"
)

var _ ExecutorNodePicker = &RandomPicker{}

// RandomPicker 基础随机选择器，职责：从可用节点中随机选出一个节点 ID
type RandomPicker struct {
	reg registry.Registry
	rnd *rand.Rand
}

func (b *RandomPicker) Name() string {
	return "RandomPicker"
}

// Pick 从可用的执行节点中随机选择一个，返回节点 ID
func (b *RandomPicker) Pick(ctx context.Context, task domain.Task) (string, error) {
	services, err := b.reg.ListServices(ctx, task.GrpcConfig.ServiceName)
	if err != nil {
		return "", fmt.Errorf("获取执行节点列表失败: %w", err)
	}
	if len(services) == 0 {
		return "", fmt.Errorf("没有可用的执行节点")
	}
	idx := b.rnd.Intn(len(services))
	return services[idx].ID, nil
}

func NewRandomPicker(reg registry.Registry) ExecutorNodePicker {
	return &RandomPicker{
		reg: reg,
		rnd: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}
