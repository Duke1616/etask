package grpc

import (
	"testing"

	executorv1 "github.com/Duke1616/etask/api/proto/gen/etask/executor/v1"
	"github.com/stretchr/testify/require"
)

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
