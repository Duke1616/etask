package agent

import (
	"context"

	"github.com/Duke1616/ework-runner/internal/agent/event"
	"github.com/Duke1616/ework-runner/internal/agent/service"
	"github.com/gotomicro/ego/core/constant"
	"github.com/gotomicro/ego/server"
)

type Consumer interface {
	Start(ctx context.Context)
	Stop(ctx context.Context) error
}

type Module struct {
	Svc    service.Service
	C      *event.ExecuteConsumer
	ctx    context.Context
	cancel context.CancelFunc
}

func (m *Module) GetConsumer() Consumer {
	return m.C
}

// --- 实现 server.Server 接口 ---

func (m *Module) Name() string {
	return "agent_consumer"
}

func (m *Module) PackageName() string {
	return "internal.agent"
}

func (m *Module) Init() error {
	return nil
}

func (m *Module) Start() error {
	// 启动消费者任务
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.C.Start(m.ctx)
	return nil
}

func (m *Module) Stop() error {
	if m.cancel != nil {
		m.cancel()
	}
	return m.C.Stop(context.Background())
}

func (m *Module) GracefulStop(ctx context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	return m.C.Stop(ctx)
}

func (m *Module) Info() *server.ServiceInfo {
	info := server.ApplyOptions(
		server.WithName(m.Name()),
		server.WithKind(constant.ServiceConsumer),
	)
	return &info
}
