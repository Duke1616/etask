package grpc

import (
	"context"
	"testing"

	executorv1 "github.com/Duke1616/etask/api/proto/gen/etask/executor/v1"
	"github.com/Duke1616/etask/internal/domain"
	taskSvc "github.com/Duke1616/etask/internal/service/task"
	"github.com/stretchr/testify/require"
)

func TestGetTaskExecution(t *testing.T) {
	server := &AgentServer{execSvc: &executionServiceStub{execution: domain.TaskExecution{
		ID: 9, Status: domain.TaskExecutionStatusSuccess, TaskResult: `{"ok":true}`,
	}}}

	response, err := server.GetTaskExecution(context.Background(),
		&executorv1.GetTaskExecutionRequest{ExecutionId: 9})

	require.NoError(t, err)
	require.Equal(t, int64(9), response.GetExecution().GetId())
	require.Equal(t, executorv1.ExecutionStatus_SUCCESS, response.GetExecution().GetStatus())
}

func TestPullTaskRejectsInvalidRequest(t *testing.T) {
	testCases := []struct {
		name      string
		request   *executorv1.PullTaskRequest
		wantError string
	}{
		{name: "缺少服务名称", request: &executorv1.PullTaskRequest{NodeId: "node-1", Handlers: []string{"shell"}}, wantError: "服务名称不能为空"},
		{name: "缺少节点 ID", request: &executorv1.PullTaskRequest{ServiceName: "executor", Handlers: []string{"shell"}}, wantError: "节点 ID 不能为空"},
		{name: "缺少处理器", request: &executorv1.PullTaskRequest{ServiceName: "executor", NodeId: "node-1"}, wantError: "至少需要声明一个处理器"},
	}
	server := &AgentServer{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := server.PullTask(t.Context(), tc.request)
			require.ErrorContains(t, err, tc.wantError)
		})
	}
}

func TestNormalizeHandlerNames(t *testing.T) {
	testCases := []struct {
		name   string
		values []string
		want   []string
	}{
		{name: "去除空白和重复值", values: []string{" shell ", "", "python", "shell"}, want: []string{"shell", "python"}},
		{name: "空输入", values: nil, want: []string{}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, normalizeHandlerNames(tc.values))
		})
	}
}

type executionServiceStub struct {
	taskSvc.ExecutionService
	execution domain.TaskExecution
}

func (s *executionServiceStub) FindByID(context.Context, int64) (domain.TaskExecution, error) {
	return s.execution, nil
}
