package dispatcher

import (
	"context"
	"errors"
	"testing"

	"github.com/Duke1616/etask/internal/domain"
)

func TestTaskDispatcherRunPlansAfterAcquire(t *testing.T) {
	acquireErr := errors.New("抢占失败")
	planningErr := errors.New("规划失败")
	testCases := []struct {
		name         string
		acquireErr   error
		planningErr  error
		wantPlanned  bool
		wantReleased bool
	}{
		{name: "路由使用抢占后的任务", planningErr: planningErr, wantPlanned: true, wantReleased: true},
		{name: "抢占失败时不规划路由", acquireErr: acquireErr},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			acquired := domain.Task{ID: 10, Version: 8, ExecMode: domain.ExecModePush}
			acquirer := &taskAcquirerStub{acquired: acquired, err: testCase.acquireErr}
			routes := &routePlannerStub{err: testCase.planningErr}
			dispatcher := NewTaskDispatcher("scheduler-1", nil, acquirer, nil, routes)

			err := dispatcher.Run(context.Background(), domain.Task{ID: 10, Version: 7})
			wantErr := testCase.planningErr
			if wantErr == nil {
				wantErr = testCase.acquireErr
			}
			if !errors.Is(err, wantErr) {
				t.Fatalf("Run() 错误 = %v, 期望包含 %v", err, wantErr)
			}
			if routes.called != testCase.wantPlanned {
				t.Fatalf("路由调用状态 = %t, 期望 %t", routes.called, testCase.wantPlanned)
			}
			if testCase.wantPlanned && routes.task.Version != acquired.Version {
				t.Fatalf("路由任务版本 = %d, 期望抢占后的版本 %d", routes.task.Version, acquired.Version)
			}
			if acquirer.released != testCase.wantReleased {
				t.Fatalf("任务释放状态 = %t, 期望 %t", acquirer.released, testCase.wantReleased)
			}
		})
	}
}

type taskAcquirerStub struct {
	acquired domain.Task
	err      error
	released bool
}

func (s *taskAcquirerStub) Acquire(context.Context, int64, int64, string) (domain.Task, error) {
	return s.acquired, s.err
}

func (s *taskAcquirerStub) Release(context.Context, int64, string) error {
	s.released = true
	return nil
}

func (s *taskAcquirerStub) Renew(context.Context, string) error {
	return nil
}

type routePlannerStub struct {
	task   domain.Task
	err    error
	called bool
}

func (s *routePlannerStub) Plan(_ context.Context, task domain.Task) (Route, error) {
	s.called = true
	s.task = task
	return Route{}, s.err
}
