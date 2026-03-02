//go:build wireinject
// +build wireinject

package ioc

import (
	"context"

	endpointv1 "github.com/Duke1616/etask/api/proto/gen/ecmdb/endpoint/v1"
	"github.com/Duke1616/etask/internal/service/scheduler"
	"github.com/Duke1616/etask/ioc"
	grpcpkg "github.com/Duke1616/etask/pkg/grpc"
	"github.com/google/wire"
	"github.com/gotomicro/ego/server/egin"
)

type SchedulerApp struct {
	Web         *egin.Component
	Server      *grpcpkg.Server
	Scheduler   *scheduler.Scheduler
	Tasks       []ioc.Task
	EndpointSvc endpointv1.EndpointServiceClient
}

func (s *SchedulerApp) StartTasks(ctx context.Context) {
	for _, t := range s.Tasks {
		go func(t ioc.Task) {
			t.Start(ctx)
		}(t)
	}
}

func InitSchedulerApp() *SchedulerApp {
	wire.Build(
		ioc.BaseSet,
		ioc.TaskSet,
		ioc.ExecutorSet,
		ioc.TaskExecutionSet,
		ioc.SchedulerSet,
		ioc.CompensatorSet,
		ioc.ConsumerSet,
		ioc.ProducerSet,
		ioc.GrpcSet,
		ioc.WebSetup,
		ioc.AgentSet,
		ioc.AppSet,
		wire.Struct(new(SchedulerApp), "Web", "Server", "Scheduler", "Tasks", "EndpointSvc"),
	)

	return new(SchedulerApp)
}
