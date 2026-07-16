package invoker

import (
	"context"
	"fmt"
	"sync"

	"github.com/Duke1616/etask/internal/domain"
	executionevent "github.com/Duke1616/etask/internal/event/execution"
	"github.com/Duke1616/etask/pkg/mpx"
	"github.com/ecodeclub/mq-api"
	"github.com/google/uuid"
)

// MQInvoker 将执行快照发布到 Agent 所属 Topic。
type MQInvoker struct {
	mq        mq.MQ
	mu        sync.Mutex
	producers map[string]mqx.Producer[executionevent.Command]
}

// NewMQInvoker 创建 Kafka Agent 调用器。
func NewMQInvoker(q mq.MQ) *MQInvoker {
	return &MQInvoker{mq: q, producers: make(map[string]mqx.Producer[executionevent.Command])}
}

// Name 返回调用器名称。
func (m *MQInvoker) Name() string {
	return "MQ"
}

// Run 发布执行命令，并返回已进入运行态的状态。
func (m *MQInvoker) Run(ctx context.Context, execution domain.TaskExecution, topic string) (domain.ExecutionState, error) {
	if topic == "" {
		return domain.ExecutionState{}, fmt.Errorf("Agent 资源池缺少执行 Topic")
	}
	producer, err := m.producer(topic)
	if err != nil {
		return domain.ExecutionState{}, err
	}
	// 每次派发生成唯一 dispatchID，用于 Kafka 重投时的 Agent 端幂等。
	dispatchID := uuid.NewString()
	command, err := executionevent.NewCommand(execution, dispatchID)
	if err != nil {
		return domain.ExecutionState{}, fmt.Errorf("构造 Agent 执行命令失败: %w", err)
	}
	if err = producer.Produce(ctx, command); err != nil {
		return domain.ExecutionState{}, fmt.Errorf("发布 Agent 执行命令失败: %w", err)
	}
	// 消息成功写入即进入 RUNNING，真实终态由 Agent 结果消费者异步更新。
	return domain.ExecutionState{
		ID: execution.ID, TaskID: execution.Task.ID, TaskName: execution.Task.Name,
		Status:         domain.TaskExecutionStatusRunning,
		ExecutorNodeID: executionevent.DispatchNodeID(topic, dispatchID),
	}, nil
}

func (m *MQInvoker) producer(topic string) (mqx.Producer[executionevent.Command], error) {
	// Producer 按 Topic 复用，避免每次任务重复创建 Kafka 资源。
	m.mu.Lock()
	defer m.mu.Unlock()
	if producer := m.producers[topic]; producer != nil {
		return producer, nil
	}
	producer, err := mqx.NewGeneralProducer[executionevent.Command](m.mq, topic)
	if err != nil {
		return nil, fmt.Errorf("创建 Agent Topic %s 生产者失败: %w", topic, err)
	}
	m.producers[topic] = producer
	return producer, nil
}
