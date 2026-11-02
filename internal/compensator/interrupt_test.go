package compensator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Duke1616/eiam/pkg/ctxutil"
	"github.com/Duke1616/etask/internal/domain"
	taskmocks "github.com/Duke1616/etask/internal/service/task/mocks"
	"github.com/Duke1616/etask/internal/service/termination"
	terminationmocks "github.com/Duke1616/etask/internal/service/termination/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestInterruptCompensator_interruptTimeoutTasks(t *testing.T) {
	testCases := []struct {
		name      string
		setup     func(ctrl *gomock.Controller) (*taskmocks.MockExecutionService, *terminationmocks.MockService)
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "没有找到超时任务",
			setup: func(ctrl *gomock.Controller) (*taskmocks.MockExecutionService, *terminationmocks.MockService) {
				execSvc := taskmocks.NewMockExecutionService(ctrl)
				termSvc := terminationmocks.NewMockService(ctrl)

				execSvc.EXPECT().FindTimeoutExecutions(gomock.Any(), 10).
					Return([]domain.TaskExecution{}, nil)

				return execSvc, termSvc
			},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "查询超时任务失败",
			setup: func(ctrl *gomock.Controller) (*taskmocks.MockExecutionService, *terminationmocks.MockService) {
				execSvc := taskmocks.NewMockExecutionService(ctrl)
				termSvc := terminationmocks.NewMockService(ctrl)

				execSvc.EXPECT().FindTimeoutExecutions(gomock.Any(), 10).
					Return(nil, errors.New("db error"))

				return execSvc, termSvc
			},
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "查找可中断任务失败")
			},
		},
		{
			name: "成功中断单个及批量超时任务",
			setup: func(ctrl *gomock.Controller) (*taskmocks.MockExecutionService, *terminationmocks.MockService) {
				execSvc := taskmocks.NewMockExecutionService(ctrl)
				termSvc := terminationmocks.NewMockService(ctrl)

				executions := []domain.TaskExecution{
					{
						ID:       101,
						TenantID: 1,
						Task: domain.Task{
							ID:   11,
							Name: "shell_job",
						},
					},
					{
						ID:       102,
						TenantID: 2,
						Task: domain.Task{
							ID:   12,
							Name: "python_job",
						},
					},
				}

				execSvc.EXPECT().FindTimeoutExecutions(gomock.Any(), 10).
					Return(executions, nil)

				termSvc.EXPECT().RequestExecution(gomock.Any(), int64(101), "任务执行超时").
					DoAndReturn(func(ctx context.Context, _ int64, _ string) error {
						require.Equal(t, int64(1), ctxutil.GetTenantID(ctx).Int64())
						return nil
					})

				termSvc.EXPECT().RequestExecution(gomock.Any(), int64(102), "任务执行超时").
					DoAndReturn(func(ctx context.Context, _ int64, _ string) error {
						require.Equal(t, int64(2), ctxutil.GetTenantID(ctx).Int64())
						return nil
					})

				return execSvc, termSvc
			},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "任务已被终止或已处于终态返回ErrRejected应安全忽略",
			setup: func(ctrl *gomock.Controller) (*taskmocks.MockExecutionService, *terminationmocks.MockService) {
				execSvc := taskmocks.NewMockExecutionService(ctrl)
				termSvc := terminationmocks.NewMockService(ctrl)

				executions := []domain.TaskExecution{
					{
						ID:       201,
						TenantID: 99,
						Task: domain.Task{
							ID:   21,
							Name: "completed_job",
						},
					},
				}

				execSvc.EXPECT().FindTimeoutExecutions(gomock.Any(), 10).
					Return(executions, nil)

				termSvc.EXPECT().RequestExecution(gomock.Any(), int64(201), "任务执行超时").
					Return(termination.ErrRejected)

				return execSvc, termSvc
			},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "单个任务终止失败不阻断后续任务",
			setup: func(ctrl *gomock.Controller) (*taskmocks.MockExecutionService, *terminationmocks.MockService) {
				execSvc := taskmocks.NewMockExecutionService(ctrl)
				termSvc := terminationmocks.NewMockService(ctrl)

				executions := []domain.TaskExecution{
					{
						ID:       301,
						TenantID: 3,
						Task: domain.Task{
							ID:   31,
							Name: "failing_job",
						},
					},
					{
						ID:       302,
						TenantID: 3,
						Task: domain.Task{
							ID:   32,
							Name: "normal_job",
						},
					},
				}

				execSvc.EXPECT().FindTimeoutExecutions(gomock.Any(), 10).
					Return(executions, nil)

				termSvc.EXPECT().RequestExecution(gomock.Any(), int64(301), "任务执行超时").
					Return(errors.New("db connection failure"))

				termSvc.EXPECT().RequestExecution(gomock.Any(), int64(302), "任务执行超时").
					Return(nil)

				return execSvc, termSvc
			},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			execSvc, termSvc := tc.setup(ctrl)

			c := NewInterruptCompensator(termSvc, execSvc, InterruptConfig{
				BatchSize:   10,
				MinDuration: time.Millisecond * 10,
			})

			err := c.interruptTimeoutTasks(context.Background())
			tc.assertErr(t, err)
		})
	}
}

func TestInterruptCompensator_StartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	execSvc := taskmocks.NewMockExecutionService(ctrl)
	termSvc := terminationmocks.NewMockService(ctrl)

	execSvc.EXPECT().FindTimeoutExecutions(gomock.Any(), 10).
		Return([]domain.TaskExecution{}, nil).AnyTimes()

	c := NewInterruptCompensator(termSvc, execSvc, InterruptConfig{
		BatchSize:   10,
		MinDuration: time.Millisecond * 20,
	})

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})

	go func() {
		c.Start(ctx)
		close(stopped)
	}()

	time.Sleep(time.Millisecond * 50)
	cancel()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("补偿器未能在 Context 取消时正常退出")
	}
}
