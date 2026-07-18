package complete

import (
	"context"
	"testing"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/event"
	"github.com/Duke1616/etask/internal/service/task"
	"github.com/stretchr/testify/require"
)

type completionExecutionServiceStub struct {
	task.ExecutionService
	updated bool
}

func (s completionExecutionServiceStub) UpdateScheduleResult(context.Context, int64,
	[]domain.TaskExecutionStatus, domain.TaskExecutionStatus, int32, int64,
	map[string]string, string, string) (bool, error) {
	return s.updated, nil
}

func TestConsumerIgnoresCompletionWithoutStateTransition(t *testing.T) {
	consumer := &Consumer{execSvc: completionExecutionServiceStub{updated: false}}

	err := consumer.handleTask(t.Context(), event.Event{
		ExecID:     10,
		TaskID:     20,
		ExecStatus: domain.TaskExecutionStatusSuccess,
	})

	require.NoError(t, err)
}
