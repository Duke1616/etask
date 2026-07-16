package dispatcher

import (
	"context"
	"errors"
	"testing"

	"github.com/Duke1616/eiam/pkg/ctxutil"
	"github.com/Duke1616/etask/internal/domain"
)

func TestRoutePlannerPlan(t *testing.T) {
	pickErr := errors.New("选择失败")
	testCases := []struct {
		name          string
		task          domain.Task
		pool          domain.ExecutionPool
		poolErr       error
		nodeID        string
		pickErr       error
		wantErr       bool
		wantTransport domain.ExecutionTransport
		wantNodeID    string
		wantMode      domain.ExecMode
		wantPoolCalls int
		wantPickCalls int
	}{
		{
			name: "Agent 资源池生成 MQ 路由",
			task: grpcTask(), pool: domain.ExecutionPool{
				Name: "agent-shell", Kind: domain.ExecutionPoolKindAgent, Transport: domain.ExecutionTransportMQ,
				Metadata: map[string]string{"topic": "agent-shell"},
			},
			wantTransport: domain.ExecutionTransportMQ,
			wantMode:      domain.ExecModePush, wantPoolCalls: 1,
		},
		{
			name: "Agent 资源池缺少 Topic 时拒绝路由",
			task: grpcTask(), pool: domain.ExecutionPool{
				Name: "agent-shell", Kind: domain.ExecutionPoolKindAgent, Transport: domain.ExecutionTransportMQ,
			},
			wantErr: true, wantPoolCalls: 1,
		},
		{
			name: "gRPC PUSH 固定目标节点",
			task: grpcTask(), pool: domain.ExecutionPool{Name: "shell", Kind: domain.ExecutionPoolKindExecutor, Transport: domain.ExecutionTransportGRPC, DispatchMode: domain.ExecModePush},
			nodeID:        "executor-1",
			wantTransport: domain.ExecutionTransportGRPC,
			wantNodeID:    "executor-1", wantMode: domain.ExecModePush,
			wantPoolCalls: 1, wantPickCalls: 1,
		},
		{
			name: "gRPC PULL 不查询和绑定节点",
			task: grpcTask(), pool: domain.ExecutionPool{Name: "shell", Kind: domain.ExecutionPoolKindExecutor, Transport: domain.ExecutionTransportGRPC, DispatchMode: domain.ExecModePull},
			wantTransport: domain.ExecutionTransportGRPC,
			wantMode:      domain.ExecModePull, wantPoolCalls: 1,
		},
		{
			name: "节点选择失败时终止路由",
			task: grpcTask(), pool: domain.ExecutionPool{Name: "shell", Kind: domain.ExecutionPoolKindExecutor, Transport: domain.ExecutionTransportGRPC, DispatchMode: domain.ExecModePush},
			pickErr: pickErr, wantErr: true, wantPoolCalls: 1, wantPickCalls: 1,
		},
		{
			name:          "HTTP 任务生成 HTTP 路由",
			task:          domain.Task{ID: 10, HTTPConfig: &domain.HTTPConfig{}},
			wantTransport: domain.ExecutionTransportHTTP, wantMode: domain.ExecModePush,
		},
		{
			name: "资源池查询失败时路由失败",
			task: grpcTask(), poolErr: errors.New("查询失败"), wantErr: true, wantPoolCalls: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pools := &poolReaderStub{pool: testCase.pool, err: testCase.poolErr}
			targets := &targetPickerStub{nodeID: testCase.nodeID, err: testCase.pickErr}
			planner := NewRoutePlanner(pools, targets)
			ctx := ctxutil.WithTenantID(context.Background(), 20)
			route, err := planner.Plan(ctx, testCase.task)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("Plan() 错误 = %v, wantErr = %v", err, testCase.wantErr)
			}
			if route.Execution.Transport != testCase.wantTransport {
				t.Fatalf("Plan() Transport = %q, 期望 %q", route.Execution.Transport, testCase.wantTransport)
			}
			if route.Execution.TargetNodeID != testCase.wantNodeID {
				t.Fatalf("Plan() NodeID = %q, 期望 %q", route.Execution.TargetNodeID, testCase.wantNodeID)
			}
			if route.Task.ExecMode != testCase.wantMode {
				t.Fatalf("路由任务模式 = %q, 期望 %q", route.Task.ExecMode, testCase.wantMode)
			}
			if pools.calls != testCase.wantPoolCalls || targets.calls != testCase.wantPickCalls {
				t.Fatalf("资源池/节点查询次数 = %d/%d, 期望 %d/%d",
					pools.calls, targets.calls, testCase.wantPoolCalls, testCase.wantPickCalls)
			}
		})
	}
}

func grpcTask() domain.Task {
	return domain.Task{ID: 10, TenantID: 20, GrpcConfig: &domain.GrpcConfig{ServiceName: "shell"}}
}

type poolReaderStub struct {
	pool  domain.ExecutionPool
	err   error
	calls int
}

func (r *poolReaderStub) Find(context.Context, string) (domain.ExecutionPool, error) {
	r.calls++
	return r.pool, r.err
}

type targetPickerStub struct {
	nodeID string
	err    error
	calls  int
}

func (p *targetPickerStub) Pick(context.Context, domain.Task) (string, error) {
	p.calls++
	return p.nodeID, p.err
}
