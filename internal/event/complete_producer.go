package event

import (
	"context"
	"encoding/json"

	"github.com/Duke1616/etask/pkg/mpx"
	"github.com/ecodeclub/mq-api"
)

type CompleteProducer interface {
	// Produce 发布一条任务完成事件。
	Produce(ctx context.Context, evt Event) error
}

type completeProducer struct {
	producer mq.Producer
}

func NewCompleteProducer(producer mq.Producer) CompleteProducer {
	return &completeProducer{
		producer: producer,
	}
}

func (c *completeProducer) Produce(ctx context.Context, evt Event) error {
	val, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	msg := &mq.Message{
		Value: val,
	}
	// 自动注入多租户与业务上下文，完成异步消息的隔离闭环
	mqx.InjectContext(ctx, msg)

	_, err = c.producer.Produce(ctx, msg)
	return err
}
