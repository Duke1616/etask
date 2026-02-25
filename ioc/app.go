package ioc

import (
	"context"

	endpointv1 "github.com/Duke1616/ework-runner/api/proto/gen/ecmdb/endpoint/v1"
	"github.com/Duke1616/ework-runner/internal/execute"
	"github.com/Duke1616/ework-runner/internal/service/scheduler"
	grpcpkg "github.com/Duke1616/ework-runner/pkg/grpc"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/server/egin"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type App struct {
	Web        *gin.Engine
	WorkerSvc  execute.Service
	EtcdClient *clientv3.Client
}

// Task 调度平台上的长任务 —— 各种补偿任务、消费者等
type Task interface {
	Start(ctx context.Context)
}

type SchedulerApp struct {
	Web         *egin.Component
	Server      *grpcpkg.Server
	Scheduler   *scheduler.Scheduler
	Tasks       []Task
	EndpointSvc endpointv1.EndpointServiceClient
}

func (a *SchedulerApp) StartTasks(ctx context.Context) {
	for _, t := range a.Tasks {
		go func(t Task) {
			t.Start(ctx)
		}(t)
	}
}
