package execution

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/ecodeclub/mq-api"
)

func TestResultConsumerConsume(t *testing.T) {
	testCases := []struct {
		name        string
		message     *mq.Message
		execution   domain.TaskExecution
		findErr     error
		wantErr     string
		wantReports int
	}{
		{name: "非终态结果进入统一状态机", message: resultMessage(t, Result{
			DispatchID: "dispatch-1",
			State:      domain.ExecutionState{ID: 10, TaskID: 20, Status: domain.TaskExecutionStatusSuccess},
			Logs:       []string{"第一行", "第二行"},
		}), execution: domain.TaskExecution{
			ID: 10, Status: domain.TaskExecutionStatusRunning,
			ExecutorNodeID: DispatchNodeID("agent-shell", "dispatch-1"),
		}, wantReports: 1},
		{name: "重复终态结果被忽略", message: resultMessage(t, Result{
			DispatchID: "dispatch-1",
			State:      domain.ExecutionState{ID: 10, Status: domain.TaskExecutionStatusSuccess},
		}), execution: domain.TaskExecution{ID: 10, Status: domain.TaskExecutionStatusSuccess}},
		{name: "非法消息", message: &mq.Message{Value: []byte("{")}, wantErr: "解析 Agent 执行结果失败"},
		{name: "缺少执行 ID", message: resultMessage(t, Result{}), wantErr: "缺少执行 ID"},
		{name: "缺少派发 ID", message: resultMessage(t, Result{
			State: domain.ExecutionState{ID: 10},
		}), wantErr: "缺少派发 ID"},
		{name: "执行记录查询失败", message: resultMessage(t, Result{
			DispatchID: "dispatch-1",
			State:      domain.ExecutionState{ID: 10},
		}), findErr: errors.New("查询失败"), wantErr: "查询失败"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := &resultHandlerStub{execution: testCase.execution, findErr: testCase.findErr}
			err := NewResultConsumer(handler).Consume(context.Background(), testCase.message)
			if testCase.wantErr == "" && err != nil {
				t.Fatalf("Consume() 返回意外错误: %v", err)
			}
			if testCase.wantErr != "" && (err == nil || !strings.Contains(err.Error(), testCase.wantErr)) {
				t.Fatalf("Consume() 错误 = %v, 期望包含 %q", err, testCase.wantErr)
			}
			if len(handler.reports) != testCase.wantReports {
				t.Fatalf("HandleReports() 调用次数 = %d, 期望 %d", len(handler.reports), testCase.wantReports)
			}
			if testCase.wantReports == 1 && len(handler.reports[0].LogChunks) != 2 {
				t.Fatalf("日志没有完整传入统一状态机: %#v", handler.reports[0].LogChunks)
			}
		})
	}
}

func resultMessage(t *testing.T, result Result) *mq.Message {
	t.Helper()
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("构造结果消息失败: %v", err)
	}
	return &mq.Message{Value: data}
}

type resultHandlerStub struct {
	execution domain.TaskExecution
	findErr   error
	reports   []*domain.Report
}

func (r *resultHandlerStub) FindByID(context.Context, int64) (domain.TaskExecution, error) {
	return r.execution, r.findErr
}
func (r *resultHandlerStub) HandleReports(_ context.Context, reports []*domain.Report) error {
	r.reports = append(r.reports, reports...)
	return nil
}
