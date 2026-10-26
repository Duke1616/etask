package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/errs"
	"github.com/Duke1616/etask/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTaskRepoForRetry struct {
	repository.TaskRepository
	getByIDFunc        func(ctx context.Context, id int64) (domain.Task, error)
	updateNextTimeFunc func(ctx context.Context, id, version, nextTime int64) (domain.Task, error)
}

func (m *mockTaskRepoForRetry) GetByID(ctx context.Context, id int64) (domain.Task, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return domain.Task{}, nil
}

func (m *mockTaskRepoForRetry) UpdateNextTime(ctx context.Context, id, version, nextTime int64) (domain.Task, error) {
	if m.updateNextTimeFunc != nil {
		return m.updateNextTimeFunc(ctx, id, version, nextTime)
	}
	return domain.Task{}, nil
}

func TestService_UpdateNextTime_CASRetry(t *testing.T) {
	testCases := []struct {
		name          string
		setupMock     func() repository.TaskRepository
		expectedErr   error
		expectSuccess bool
	}{
		{
			name: "第一次更新命中版本冲突，自旋重试第二次成功",
			setupMock: func() repository.TaskRepository {
				attempts := 0
				return &mockTaskRepoForRetry{
					getByIDFunc: func(ctx context.Context, id int64) (domain.Task, error) {
						attempts++
						return domain.Task{
							ID:       id,
							Type:     domain.TaskTypeRecurring,
							CronExpr: "0 0 10 * * 1-5",
							Version:  int64(attempts), // 每次重新查出最新 version
						}, nil
					},
					updateNextTimeFunc: func(ctx context.Context, id, version, nextTime int64) (domain.Task, error) {
						if version == 1 {
							// 第一次模拟并发续约冲突
							return domain.Task{}, errs.ErrTaskUpdateNextTimeFailed
						}
						return domain.Task{
							ID:       id,
							Version:  version + 1,
							NextTime: nextTime,
						}, nil
					},
				}
			},
			expectSuccess: true,
		},
		{
			name: "非乐观锁版本错误，不予重试直接快速失败",
			setupMock: func() repository.TaskRepository {
				callCount := 0
				return &mockTaskRepoForRetry{
					getByIDFunc: func(ctx context.Context, id int64) (domain.Task, error) {
						callCount++
						return domain.Task{
							ID:       id,
							Type:     domain.TaskTypeRecurring,
							CronExpr: "0 0 10 * * 1-5",
							Version:  1,
						}, nil
					},
					updateNextTimeFunc: func(ctx context.Context, id, version, nextTime int64) (domain.Task, error) {
						return domain.Task{}, errors.New("数据库物理连接中断")
					},
				}
			},
			expectedErr:   errors.New("数据库物理连接中断"),
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &service{repo: tc.setupMock()}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			task, err := svc.UpdateNextTime(ctx, 14)
			if tc.expectSuccess {
				require.NoError(t, err)
				assert.Equal(t, int64(14), task.ID)
				assert.False(t, task.NextTime == 0)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr.Error())
			}
		})
	}
}
