package execution

import (
	"strings"
	"testing"

	"github.com/Duke1616/etask/internal/domain"
)

func TestNewCommand(t *testing.T) {
	testCases := []struct {
		name    string
		exec    domain.TaskExecution
		wantErr string
	}{
		{name: "正式任务", exec: commandExecution(domain.TaskExecutionSourceTask, 30)},
		{name: "试运行允许任务 ID 为空", exec: commandExecution(domain.TaskExecutionSourceCodebookPreview, 0)},
		{name: "拒绝未声明来源", exec: commandExecution("", 30), wantErr: "来源非法"},
		{name: "拒绝缺少处理器配置", exec: domain.TaskExecution{ID: 10, TenantID: 20}, wantErr: "缺少处理器配置"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			command, err := NewCommand(testCase.exec, "dispatch-1")
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("NewCommand() 错误 = %v, 期望包含 %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewCommand() 返回意外错误: %v", err)
			}
			if command.Source != testCase.exec.Source || command.TaskID != testCase.exec.Task.ID ||
				command.Handler != "shell" {
				t.Fatalf("NewCommand() = %#v", command)
			}
		})
	}
}

func commandExecution(source domain.TaskExecutionSource, taskID int64) domain.TaskExecution {
	return domain.TaskExecution{
		ID: 10, TenantID: 20, Source: source,
		Task: domain.Task{ID: taskID, Name: "测试任务", GrpcConfig: &domain.GrpcConfig{
			HandlerName: "shell", Params: map[string]string{"code": "echo ok"},
		}},
	}
}
