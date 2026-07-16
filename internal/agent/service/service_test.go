package service

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	artifactv1 "github.com/Duke1616/etask/api/proto/gen/etask/artifact/v1"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/sdk/executor"
)

func TestServiceReceive(t *testing.T) {
	testCases := []struct {
		name   string
		before func(t *testing.T) (*serviceFixture, func())
		run    func(t *testing.T, fixture *serviceFixture)
		after  func(t *testing.T, fixture *serviceFixture)
	}{
		{
			name: "相同派发并发到达时只执行一次",
			before: func(t *testing.T) (*serviceFixture, func()) {
				fixture := newServiceFixture(t)
				fixture.handler.block = make(chan struct{})
				return fixture, func() {}
			},
			run: func(t *testing.T, fixture *serviceFixture) {
				var wg sync.WaitGroup
				outputs := make([]string, 2)
				errs := make([]error, 2)
				for index := range outputs {
					wg.Add(1)
					go func(index int) {
						defer wg.Done()
						output, err := fixture.service.Receive(context.Background(), "dispatch-1", fixture.execution)
						outputs[index], errs[index] = output.Result, err
					}(index)
				}
				<-fixture.handler.started
				close(fixture.handler.block)
				wg.Wait()
				for index, err := range errs {
					if err != nil {
						t.Fatalf("Receive()[%d] 返回意外错误: %v", index, err)
					}
					if outputs[index] != `{"status":"ok"}` {
						t.Fatalf("Receive()[%d] 结果 = %q", index, outputs[index])
					}
				}
			},
			after: func(t *testing.T, fixture *serviceFixture) {
				if count := fixture.handler.runs.Load(); count != 1 {
					t.Fatalf("Handler 执行次数 = %d, 期望 1", count)
				}
			},
		},
		{
			name: "不同派发可以重试同一个执行记录",
			before: func(t *testing.T) (*serviceFixture, func()) {
				return newServiceFixture(t), func() {}
			},
			run: func(t *testing.T, fixture *serviceFixture) {
				for _, dispatchID := range []string{"dispatch-1", "dispatch-2"} {
					if _, err := fixture.service.Receive(context.Background(), dispatchID, fixture.execution); err != nil {
						t.Fatalf("Receive() 返回意外错误: %v", err)
					}
				}
			},
			after: func(t *testing.T, fixture *serviceFixture) {
				if count := fixture.handler.runs.Load(); count != 2 {
					t.Fatalf("Handler 执行次数 = %d, 期望 2", count)
				}
			},
		},
		{
			name: "制品引用通过共享引擎准备并清理",
			before: func(t *testing.T) (*serviceFixture, func()) {
				fixture := newServiceFixture(t)
				fixture.execution.Artifacts = []domain.ArtifactRef{{
					ReleaseID: 1, Digest: strings.Repeat("a", 64), BlobChecksum: strings.Repeat("b", 64),
					Size: 1, Format: "tar.zst", FormatVersion: 1, Scope: domain.CodebookScopeSystem,
				}}
				return fixture, func() {}
			},
			run: func(t *testing.T, fixture *serviceFixture) {
				if _, err := fixture.service.Receive(context.Background(), "dispatch-1", fixture.execution); err != nil {
					t.Fatalf("Receive() 返回意外错误: %v", err)
				}
			},
			after: func(t *testing.T, fixture *serviceFixture) {
				if fixture.preparer.prepares.Load() != 1 || fixture.preparer.prepared.closes.Load() != 1 {
					t.Fatalf("制品准备/清理次数 = %d/%d, 期望 1/1",
						fixture.preparer.prepares.Load(), fixture.preparer.prepared.closes.Load())
				}
				if roots := fixture.handler.roots.Load(); roots != "/system:/dependencies" {
					t.Fatalf("Handler 收到的制品目录 = %v", roots)
				}
			},
		},
		{
			name: "敏感变量不会进入 Kafka 结果日志",
			before: func(t *testing.T) (*serviceFixture, func()) {
				fixture := newServiceFixture(t)
				fixture.execution.Task.GrpcConfig.Params = map[string]string{
					"variables": `[{"key":"TOKEN","value":"top-secret","secret":true}]`,
				}
				fixture.handler.logMessage = "token=top-secret"
				return fixture, func() {}
			},
			run: func(t *testing.T, fixture *serviceFixture) {
				output, err := fixture.service.Receive(context.Background(), "dispatch-1", fixture.execution)
				if err != nil {
					t.Fatalf("Receive() 返回意外错误: %v", err)
				}
				if len(output.Logs) != 1 || output.Logs[0] != "token=[MASKED]" {
					t.Fatalf("脱敏日志 = %#v", output.Logs)
				}
			},
			after: func(t *testing.T, fixture *serviceFixture) {},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture, cleanup := testCase.before(t)
			defer cleanup()
			testCase.run(t, fixture)
			testCase.after(t, fixture)
		})
	}
}

type serviceFixture struct {
	service   Service
	handler   *serviceHandlerStub
	preparer  *servicePreparerStub
	execution domain.TaskExecution
}

func newServiceFixture(t *testing.T) *serviceFixture {
	t.Helper()
	handler := &serviceHandlerStub{started: make(chan struct{}, 1)}
	preparer := &servicePreparerStub{prepared: &servicePreparedStub{
		roots: executor.ArtifactRoots{Default: "/system", Dependencies: "/dependencies"},
	}}
	return &serviceFixture{
		service:  NewService([]executor.TaskHandler{handler}, preparer, nil),
		handler:  handler,
		preparer: preparer,
		execution: domain.TaskExecution{
			ID: 10, TenantID: 20,
			Task: domain.Task{ID: 30, Name: "测试任务", GrpcConfig: &domain.GrpcConfig{HandlerName: handler.Name()}},
		},
	}
}

type serviceHandlerStub struct {
	runs       atomic.Int32
	started    chan struct{}
	block      chan struct{}
	roots      atomic.Value
	logMessage string
}

func (h *serviceHandlerStub) Name() string                   { return "test" }
func (h *serviceHandlerStub) Desc() string                   { return "测试处理器" }
func (h *serviceHandlerStub) Metadata() []executor.Parameter { return nil }
func (h *serviceHandlerStub) Run(ctx *executor.Context) error {
	h.runs.Add(1)
	select {
	case h.started <- struct{}{}:
	default:
	}
	roots := ctx.ArtifactRoots()
	h.roots.Store(roots.Default + ":" + roots.Dependencies)
	if h.block != nil {
		<-h.block
	}
	if h.logMessage != "" {
		ctx.Log("%s", h.logMessage)
	}
	ctx.SetResult("status", "ok")
	return nil
}

type servicePreparerStub struct {
	prepares atomic.Int32
	prepared *servicePreparedStub
}

func (p *servicePreparerStub) Prune() error { return nil }
func (p *servicePreparerStub) Prepare(context.Context, artifactv1.ArtifactServiceClient,
	[]*artifactv1.ArtifactRef) (executor.PreparedArtifacts, error) {
	p.prepares.Add(1)
	return p.prepared, nil
}

type servicePreparedStub struct {
	roots  executor.ArtifactRoots
	closes atomic.Int32
}

func (p *servicePreparedStub) Roots() executor.ArtifactRoots { return p.roots }
func (p *servicePreparedStub) Close() error {
	p.closes.Add(1)
	return nil
}
