package complete

import (
	"context"
	"errors"
	"testing"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/event"
	"github.com/Duke1616/etask/internal/service/task"
	"github.com/stretchr/testify/require"
)

type completionExecutionServiceStub struct {
	task.ExecutionService
	updated      bool
	targetStatus domain.TaskExecutionStatus
}

func (s *completionExecutionServiceStub) UpdateScheduleResult(_ context.Context, _ int64,
	_ []domain.TaskExecutionStatus, status domain.TaskExecutionStatus, _ int32, _ int64,
	_ map[string]string, _ string, _ string) (bool, error) {
	s.targetStatus = status
	return s.updated, nil
}

func (s *completionExecutionServiceStub) FindByID(_ context.Context, id int64) (domain.TaskExecution, error) {
	return domain.TaskExecution{ID: id}, nil
}

func TestConsumerIgnoresCompletionWithoutStateTransition(t *testing.T) {
	consumer := &Consumer{execSvc: &completionExecutionServiceStub{updated: false}}

	err := consumer.handleTask(t.Context(), event.Event{
		ExecID:     10,
		TaskID:     20,
		ExecStatus: domain.TaskExecutionStatusSuccess,
	})

	require.NoError(t, err)
}

func TestConsumerPersistsCancelledCompletion(t *testing.T) {
	executions := &completionExecutionServiceStub{updated: false}
	consumer := &Consumer{execSvc: executions}

	err := consumer.handleTask(t.Context(), event.Event{
		ExecID: 10, ExecStatus: domain.TaskExecutionStatusCancelled,
	})

	require.NoError(t, err)
	require.Equal(t, domain.TaskExecutionStatusCancelled, executions.targetStatus)
}

type completionTaskServiceStub struct {
	task.Service
	getByIDFunc        func(ctx context.Context, id int64) (domain.Task, error)
	updateNextTimeFunc func(ctx context.Context, id int64) (domain.Task, error)
}

func (s *completionTaskServiceStub) GetByID(ctx context.Context, id int64) (domain.Task, error) {
	if s.getByIDFunc != nil {
		return s.getByIDFunc(ctx, id)
	}
	return domain.Task{}, nil
}

func (s *completionTaskServiceStub) UpdateNextTime(ctx context.Context, id int64) (domain.Task, error) {
	if s.updateNextTimeFunc != nil {
		return s.updateNextTimeFunc(ctx, id)
	}
	return domain.Task{}, nil
}

type mockCompletionNotifier struct {
	notifiedCount         int
	businessDispatchCount int
}

func (m *mockCompletionNotifier) Notify(ctx context.Context, rule domain.ExecutionNotificationRule, execution domain.TaskExecution) error {
	m.notifiedCount++
	return nil
}

func (m *mockCompletionNotifier) DispatchBusinessNotifications(ctx context.Context, config domain.BusinessNotificationConfig, execution domain.TaskExecution) error {
	m.businessDispatchCount++
	return nil
}

func TestConsumerDispatchesNotificationsBeforeUpdateNextTime(t *testing.T) {
	executions := &completionExecutionServiceStub{updated: true}
	notifier := &mockCompletionNotifier{}
	taskSvc := &completionTaskServiceStub{
		getByIDFunc: func(ctx context.Context, id int64) (domain.Task, error) {
			return domain.Task{
				ID: id,
				NotificationRules: []domain.ExecutionNotificationRule{
					{
						TriggerStatus: domain.TaskExecutionStatusSuccess,
						Enabled:       true,
					},
				},
				BusinessNotification: &domain.BusinessNotificationConfig{
					Enabled: true,
				},
			}, nil
		},
		updateNextTimeFunc: func(ctx context.Context, id int64) (domain.Task, error) {
			// 故意模拟下次时间推算失败
			return domain.Task{}, errors.New("模拟 UpdateNextTime 失败")
		},
	}

	consumer := &Consumer{
		execSvc:  executions,
		taskSvc:  taskSvc,
		notifier: notifier,
	}

	// 执行 handleTask
	err := consumer.handleTask(t.Context(), event.Event{
		ExecID:     10,
		TaskID:     20,
		ExecStatus: domain.TaskExecutionStatusSuccess,
	})

	// 尽管 UpdateNextTime 返回了错误
	require.Error(t, err)
	require.Contains(t, err.Error(), "模拟 UpdateNextTime 失败")

	// 但执行终态通知与业务通知必须在此之前已成功投递！
	require.Equal(t, 1, notifier.notifiedCount, "执行终态通知应先于 UpdateNextTime 送达")
	require.Equal(t, 1, notifier.businessDispatchCount, "业务通知应先于 UpdateNextTime 送达")
}
