//go:build wireinject

package agent

import (
	"encoding/json"

	"github.com/Duke1616/etask/internal/agent/domain"
	event2 "github.com/Duke1616/etask/internal/agent/event"
	"github.com/Duke1616/etask/internal/agent/service"
	"github.com/Duke1616/etask/internal/agent/web"
	"github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/Duke1616/etask/pkg/grpc/registry/etcd"
	"github.com/ecodeclub/mq-api"
	"github.com/google/uuid"
	"github.com/google/wire"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Instance struct {
	Name        string `yaml:"name" json:"name"`                 // 实例名称
	Desc        string `yaml:"desc" json:"desc"`                 // 注解
	Topic       string `yaml:"topic" json:"topic"`               // 建立 Topic 通道
	WorkerCount int    `yaml:"worker_count" json:"worker_count"` // 并发工作协程数量
}

var ProviderSet = wire.NewSet(
	service.NewService)

func InitModule(q mq.MQ, etcdClient *clientv3.Client) *Module {
	wire.Build(
		ProviderSet,
		InitRegistry,
		web.NewHandler,
		initExecuteConsumer,
		initExecuteProducer,
		wire.Struct(new(Module), "Svc", "C", "Hdl"),
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

func InitRegistry(etcdClient *clientv3.Client) registry.Registry {
	reg, err := etcd.NewRegistryWithPrefix(etcdClient, "/etask/kafka")
	if err != nil {
		panic(err)
	}
	return reg
}

func initExecuteConsumer(q mq.MQ, svc service.Service, producer event2.TaskExecuteResultProducer, reg registry.Registry) *event2.ExecuteConsumer {
	var cfg Instance
	if err := viper.UnmarshalKey("agent", &cfg); err != nil {
		panic(err)
	}

	handlerMetas, _ := json.Marshal(svc.ListHandlers())
	instance := registry.ServiceInstance{
		Name:    domain.ServiceName, // 统一的服务分组名称
		Address: uuid.New().String(),
		Metadata: map[string]any{
			"name":               cfg.Name,
			"desc":               cfg.Desc,
			"topic":              cfg.Topic,
			"supported_handlers": string(handlerMetas),
		},
	}

	// 2. 创建消费者
	consumer, err := event2.NewExecuteConsumer(q, svc, cfg.Topic, producer, reg, instance, cfg.WorkerCount)
	if err != nil {
		panic(err)
	}

	return consumer
}
