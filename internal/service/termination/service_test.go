package termination_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Duke1616/eiam/pkg/ctxutil"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/repository"
	repositorymocks "github.com/Duke1616/etask/internal/repository/mocks"
	invokermocks "github.com/Duke1616/etask/internal/service/invoker/mocks"
	"github.com/Duke1616/etask/internal/service/termination"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRequestPersistsIntentBeforeExecutionExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	cancellations := repositorymocks.NewMockExecutionCancellationRepository(ctrl)
	executions := repositorymocks.NewMockTaskExecutionRepository(ctrl)
	physical := invokermocks.NewMockInvoker(ctrl)
	ctx := ctxutil.WithTenantID(context.Background(), 7)
	cancellations.EXPECT().Request(gomock.Any(), int64(0), "eflow:1:1", "管理员终止").Return(nil)
	service := termination.NewService(cancellations, executions, physical)

	err := service.Request(ctx, termination.Request{RequestID: "eflow:1:1", Reason: "管理员终止"})

	require.NoError(t, err)
}

func TestRequestExecutionPersistsCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	cancellations := repositorymocks.NewMockExecutionCancellationRepository(ctrl)
	executions := repositorymocks.NewMockTaskExecutionRepository(ctrl)
	physical := invokermocks.NewMockInvoker(ctrl)
	ctx := ctxutil.WithTenantID(context.Background(), 7)
	cancellations.EXPECT().Request(gomock.Any(), int64(9), "", "参数配置错误").Return(nil)
	service := termination.NewService(cancellations, executions, physical)

	err := service.RequestExecution(ctx, 9, " 参数配置错误 ")

	require.NoError(t, err)
}

func TestRequestExecutionRejectsBlankReason(t *testing.T) {
	ctrl := gomock.NewController(t)
	service := termination.NewService(
		repositorymocks.NewMockExecutionCancellationRepository(ctrl),
		repositorymocks.NewMockTaskExecutionRepository(ctrl),
		invokermocks.NewMockInvoker(ctrl),
	)

	err := service.RequestExecution(ctxutil.WithTenantID(context.Background(), 7), 9, " ")

	require.ErrorIs(t, err, termination.ErrInvalidCommand)
}

func TestDeliverPendingMarksSuccessfulSignalSent(t *testing.T) {
	ctrl := gomock.NewController(t)
	cancellations := repositorymocks.NewMockExecutionCancellationRepository(ctrl)
	executions := repositorymocks.NewMockTaskExecutionRepository(ctrl)
	physical := invokermocks.NewMockInvoker(ctrl)
	pending := []domain.ExecutionCancellation{{
		ID: 1, TenantID: 7, ExecutionID: 9, Reason: "管理员终止",
	}}
	execution := domain.TaskExecution{ID: 9}
	gomock.InOrder(
		cancellations.EXPECT().ListPending(gomock.Any(), 10).Return(pending, nil),
		executions.EXPECT().GetByID(gomock.Any(), int64(9)).Return(execution, nil),
		physical.EXPECT().Terminate(gomock.Any(), execution, "管理员终止").
			DoAndReturn(func(ctx context.Context, _ domain.TaskExecution, _ string) error {
				require.Equal(t, int64(7), ctxutil.GetTenantID(ctx).Int64())
				return nil
			}),
		cancellations.EXPECT().MarkSent(gomock.Any(), int64(1)).Return(nil),
	)
	service := termination.NewService(cancellations, executions, physical)

	err := service.DeliverPending(context.Background(), 10)

	require.NoError(t, err)
}

func TestDeliverPendingSchedulesRetryAfterSignalFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	cancellations := repositorymocks.NewMockExecutionCancellationRepository(ctrl)
	executions := repositorymocks.NewMockTaskExecutionRepository(ctrl)
	physical := invokermocks.NewMockInvoker(ctrl)
	pending := []domain.ExecutionCancellation{{
		ID: 1, TenantID: 7, ExecutionID: 9, Reason: "管理员终止",
	}}
	execution := domain.TaskExecution{ID: 9}
	deliveryErr := errors.New("executor unavailable")
	gomock.InOrder(
		cancellations.EXPECT().ListPending(gomock.Any(), 10).Return(pending, nil),
		executions.EXPECT().GetByID(gomock.Any(), int64(9)).Return(execution, nil),
		physical.EXPECT().Terminate(gomock.Any(), execution, "管理员终止").Return(deliveryErr),
		cancellations.EXPECT().MarkFailed(gomock.Any(), int64(1), deliveryErr.Error(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, _ string, nextAttemptAt int64) error {
				require.NotZero(t, nextAttemptAt)
				return nil
			}),
	)
	service := termination.NewService(cancellations, executions, physical)

	err := service.DeliverPending(context.Background(), 10)

	require.ErrorContains(t, err, "executor unavailable")
}

func TestDeliverPendingSkipsPhysicalSignalWhenAlreadyTerminal(t *testing.T) {
	terminalStatuses := []domain.TaskExecutionStatus{
		domain.TaskExecutionStatusSuccess,
		domain.TaskExecutionStatusFailed,
		domain.TaskExecutionStatusCancelled,
	}

	for _, status := range terminalStatuses {
		t.Run(string(status), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			cancellations := repositorymocks.NewMockExecutionCancellationRepository(ctrl)
			executions := repositorymocks.NewMockTaskExecutionRepository(ctrl)
			physical := invokermocks.NewMockInvoker(ctrl)

			pending := []domain.ExecutionCancellation{{
				ID: 1, TenantID: 7, ExecutionID: 9, Reason: "管理员终止",
			}}
			execution := domain.TaskExecution{ID: 9, Status: status}

			gomock.InOrder(
				cancellations.EXPECT().ListPending(gomock.Any(), 10).Return(pending, nil),
				executions.EXPECT().GetByID(gomock.Any(), int64(9)).Return(execution, nil),
				// 关键断言：已终态执行严禁调用 physical.Terminate，直接标记 MarkSent
				cancellations.EXPECT().MarkSent(gomock.Any(), int64(1)).Return(nil),
			)
			service := termination.NewService(cancellations, executions, physical)

			err := service.DeliverPending(context.Background(), 10)
			require.NoError(t, err)
		})
	}
}

func TestTerminateActiveByTaskID(t *testing.T) {
	testCases := []struct {
		name      string
		taskID    int64
		mock      func(ctrl *gomock.Controller, cancellations *repositorymocks.MockExecutionCancellationRepository, executions *repositorymocks.MockTaskExecutionRepository)
		wantErr   error
		assertErr func(t *testing.T, err error)
	}{
		{
			name:   "非法 taskID",
			taskID: 0,
			mock:   func(ctrl *gomock.Controller, cancellations *repositorymocks.MockExecutionCancellationRepository, executions *repositorymocks.MockTaskExecutionRepository) {},
			wantErr: termination.ErrInvalidCommand,
		},
		{
			name:   "查询活跃执行记录失败",
			taskID: 100,
			mock: func(ctrl *gomock.Controller, cancellations *repositorymocks.MockExecutionCancellationRepository, executions *repositorymocks.MockTaskExecutionRepository) {
				executions.EXPECT().FindNonTerminalByTaskID(gomock.Any(), int64(100)).
					Return(nil, errors.New("db error"))
			},
			assertErr: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "查询任务 100 活跃执行记录失败")
			},
		},
		{
			name:   "批量终止活跃执行成功，且容忍部分执行已进入终态被拒绝",
			taskID: 100,
			mock: func(ctrl *gomock.Controller, cancellations *repositorymocks.MockExecutionCancellationRepository, executions *repositorymocks.MockTaskExecutionRepository) {
				executions.EXPECT().FindNonTerminalByTaskID(gomock.Any(), int64(100)).
					Return([]domain.TaskExecution{
						{ID: 101, Status: domain.TaskExecutionStatusRunning},
						{ID: 102, Status: domain.TaskExecutionStatusPrepare},
					}, nil)

				// 101 取消成功
				cancellations.EXPECT().Request(gomock.Any(), int64(101), "", "任务停用联动终止").
					Return(nil)
				// 102 刚好完成，返回 ErrCancellationRejected，应被吞掉不影响整体
				cancellations.EXPECT().Request(gomock.Any(), int64(102), "", "任务停用联动终止").
					Return(repository.ErrCancellationRejected)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			cancellations := repositorymocks.NewMockExecutionCancellationRepository(ctrl)
			executions := repositorymocks.NewMockTaskExecutionRepository(ctrl)
			physical := invokermocks.NewMockInvoker(ctrl)
			ctx := ctxutil.WithTenantID(context.Background(), 7)

			tc.mock(ctrl, cancellations, executions)

			service := termination.NewService(cancellations, executions, physical)
			err := service.TerminateActiveByTaskID(ctx, tc.taskID, "任务停用联动终止")

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else if tc.assertErr != nil {
				tc.assertErr(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
