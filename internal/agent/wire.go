//go:build wireinject

package agent

import (
	"context"
	"time"

	event2 "github.com/Duke1616/ework-runner/internal/agent/event"
	"github.com/Duke1616/ework-runner/internal/agent/service"
	"github.com/Duke1616/ework-runner/pkg/grpc/registry"
	"github.com/ecodeclub/mq-api"
	"github.com/google/wire"
	"github.com/gotomicro/ego/core/elog"
	"github.com/spf13/viper"
)

type Instance struct {
	Name  string `yaml:"name" json:"name"`   // 实例名称
	Desc  string `yaml:"desc" json:"desc"`   // 注解
	Topic string `yaml:"topic" json:"topic"` // 建立 Topic 通道
}

var ProviderSet = wire.NewSet(
	service.NewService)

func InitModule(q mq.MQ, reg registry.Registry) *Module {
	wire.Build(
		ProviderSet,
		initExecuteConsumer,
		initExecuteProducer,
		wire.Struct(new(Module), "*"),
	)
	return new(Module)
}

func initExecuteProducer(q mq.MQ) event2.TaskExecuteResultProducer {
	producer, err := event2.NewExecuteResultEventProducer(q)
	if err != nil {
		panic(err)
	}
	return producer
}

func initExecuteConsumer(q mq.MQ, svc service.Service, producer event2.TaskExecuteResultProducer, reg registry.Registry) *event2.ExecuteConsumer {
	var cfg Instance
	if err := viper.UnmarshalKey("agent", &cfg); err != nil {
		panic(err)
	}

	// 1. 服务注册
	// Agent 模式配置读取 (实例注册)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := reg.Register(ctx, registry.ServiceInstance{
		ID:   cfg.Name,
		Name: "agent", // 统一的服务分组名称
		Metadata: map[string]any{
			"desc":               cfg.Desc,
			"topic":              cfg.Topic,
			"supported_handlers": svc.ListHandlers(),
		},
	})
	if err != nil {
		elog.Error("agent_register_failed", elog.FieldErr(err))
	}

	// 2. 创建消费者
	consumer, err := event2.NewExecuteConsumer(q, svc, cfg.Topic, producer)
	if err != nil {
		panic(err)
	}

	return consumer
}
