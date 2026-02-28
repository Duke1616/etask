package agent

import (
	"context"

	"github.com/Duke1616/ework-runner/internal/agent/event"
	"github.com/Duke1616/ework-runner/internal/agent/service"
)

type Consumer interface {
	Start(ctx context.Context) <-chan error
}

type Module struct {
	Svc service.Service
	C   *event.ExecuteConsumer
}

func (m *Module) GetConsumer() Consumer {
	return m.C
}
