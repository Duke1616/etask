package picker

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/pkg/grpc/registry"
)

func TestRandomPickerPick(t *testing.T) {
	testCases := []struct {
		name     string
		services []registry.ServiceInstance
		listErr  error
		handler  string
		want     string
		wantErr  string
	}{
		{
			name: "选择支持处理器的节点", handler: "shell",
			services: []registry.ServiceInstance{
				executorInstance("python-node", "PUSH", `[{"name":"python"}]`),
				executorInstance("shell-node", "PUSH", `[{"name":"shell"}]`),
			},
			want: "shell-node",
		},
		{
			name: "忽略派发模式不一致的节点", handler: "shell",
			services: []registry.ServiceInstance{
				executorInstance("pull-node", "PULL", `[{"name":"shell"}]`),
				executorInstance("push-node", "PUSH", `[{"name":"shell"}]`),
			},
			want: "push-node",
		},
		{
			name: "没有匹配 Handler 的节点", handler: "shell",
			services: []registry.ServiceInstance{
				executorInstance("python-node", "PUSH", `[{"name":"python"}]`),
			},
			wantErr: "没有支持处理器 shell",
		},
		{name: "注册中心查询失败", handler: "shell", listErr: errors.New("etcd unavailable"), wantErr: "获取执行节点列表失败"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reg := &registryStub{services: testCase.services, err: testCase.listErr}
			picker := NewRandomPicker(reg)
			got, err := picker.Pick(context.Background(), domain.Task{
				ExecMode:   domain.ExecModePush,
				GrpcConfig: &domain.GrpcConfig{ServiceName: "executor", HandlerName: testCase.handler},
			})
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("Pick() 错误 = %v, 期望包含 %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Pick() 返回意外错误: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("Pick() = %#v, 期望 %#v", got, testCase.want)
			}
			if reg.calls != 1 {
				t.Fatalf("ListServices() 调用次数 = %d, 期望 1", reg.calls)
			}
		})
	}
}

func executorInstance(id, mode, handlers string) registry.ServiceInstance {
	return registry.ServiceInstance{ID: id, Metadata: map[string]any{
		"role": "executor", "mode": mode, "supported_handlers": handlers,
	}}
}

type registryStub struct {
	services []registry.ServiceInstance
	err      error
	calls    int
}

func (r *registryStub) Register(context.Context, registry.ServiceInstance) error   { return nil }
func (r *registryStub) UnRegister(context.Context, registry.ServiceInstance) error { return nil }
func (r *registryStub) ListServices(context.Context, string) ([]registry.ServiceInstance, error) {
	r.calls++
	return r.services, r.err
}
func (r *registryStub) Subscribe(string) <-chan registry.Event { return nil }
func (r *registryStub) Close() error                           { return nil }
