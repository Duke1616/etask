package ioc

import (
	executorv1 "github.com/Duke1616/etask/api/proto/gen/etask/executor/v1"
	"github.com/Duke1616/etask/internal/service/invoker"
	"github.com/Duke1616/etask/pkg/grpc/pool"
	"github.com/ecodeclub/mq-api"
)

func InitInvoker(clients *pool.Clients[executorv1.ExecutorServiceClient], q mq.MQ) invoker.Invoker {
	return invoker.NewDispatcher(
		invoker.NewHTTPInvoker(),
		invoker.NewGRPCInvoker(clients),
		invoker.NewLocalInvoker(map[string]invoker.LocalExecuteFunc{}),
		invoker.NewMQInvoker(q))
}
