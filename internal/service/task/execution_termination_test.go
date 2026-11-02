package task

import (
	"context"
	"errors"
	"testing"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/event"
	"github.com/Duke1616/etask/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestUpdateStateIgnoresCallbackAfterTerminalState(t *testing.T) {
	repo := &terminationRepositoryStub{states: []domain.TaskExecutionStatus{
		domain.TaskExecutionStatusCancelled,
	}}
	service := &executionService{repo: repo}

	err := service.UpdateState(context.Background(), domain.ExecutionState{
		ID: 1, Status: domain.TaskExecutionStatusSuccess,
	})

	require.NoError(t, err)
}

func TestUpdateWorkflowStatePublishesCancelledCompletion(t *testing.T) {
	producer := &completeProducerStub{}
	repo := &terminationRepositoryStub{
		states: []domain.TaskExecutionStatus{domain.TaskExecutionStatusRunning},
		source: domain.TaskExecutionSourceWorkflow,
	}
	service := &executionService{repo: repo, producer: producer}

	err := service.UpdateState(context.Background(), domain.ExecutionState{
		ID: 1, Status: domain.TaskExecutionStatusCancelled, TaskResult: "管理员强制结束",
	})

	require.NoError(t, err)
	require.Equal(t, domain.TaskExecutionStatusCancelled, producer.event.ExecStatus)
	require.Equal(t, "管理员强制结束", producer.event.TaskResult)
}

type terminationRepositoryStub struct {
	repository.TaskExecutionRepository
	states []domain.TaskExecutionStatus
	reads  int
	source domain.TaskExecutionSource
	execution domain.TaskExecution
}

func (s *terminationRepositoryStub) GetByID(context.Context, int64) (domain.TaskExecution, error) {
	if s.execution.ID != 0 {
		return s.execution, nil
	}
	if s.reads >= len(s.states) {
		return domain.TaskExecution{}, errors.New("unexpected GetByID call")
	}
	status := s.states[s.reads]
	s.reads++
	return domain.TaskExecution{ID: 1, Status: status, Source: s.source}, nil
}

func TestUpdateState_RetryExceeded_PublishesFailedCompletion(t *testing.T) {
	testCases := []struct {
		name           string
		currentRetries int64
		maxRetries     int32
		expectedStatus domain.TaskExecutionStatus
	}{
		{
			name:           "达到最大重试次数时状态被正确推进为Failed终态并发布事件",
			currentRetries: 3,
			maxRetries:     3,
			expectedStatus: domain.TaskExecutionStatusFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			producer := &completeProducerStub{}
			repo := &terminationRepositoryStub{
				execution: domain.TaskExecution{
					ID:         100,
					Status:     domain.TaskExecutionStatusRunning,
					RetryCount: tc.currentRetries,
					Task: domain.Task{
						ID:   10,
						Name: "retry-task",
						RetryConfig: &domain.RetryConfig{
							MaxRetries: tc.maxRetries,
						},
					},
				},
			}
			service := &executionService{repo: repo, producer: producer}

			err := service.UpdateState(context.Background(), domain.ExecutionState{
				ID:         100,
				Status:     domain.TaskExecutionStatusFailedRetryable,
				TaskResult: "process exit code 1",
			})

			require.NoError(t, err)
			require.Equal(t, tc.expectedStatus, producer.event.ExecStatus)
			require.Equal(t, "process exit code 1", producer.event.TaskResult)
			require.Equal(t, int64(100), producer.event.ExecID)
		})
	}
}

type completeProducerStub struct {
	event event.Event
}

func (s *completeProducerStub) Produce(_ context.Context, evt event.Event) error {
	s.event = evt
	return nil
}
