package executor

import (
	"testing"

	"github.com/Duke1616/etask/pkg/grpc/registry"
)

func TestToExecutor(t *testing.T) {
	group := executorGroup{
		Name: "aliyun",
		Instances: []registry.ServiceInstance{
			{
				ID:      "node-1",
				Address: "127.0.0.1:9000",
				Metadata: map[string]any{
					"desc":               "aliyun executor",
					"mode":               "PUSH",
					"supported_handlers": `[{"name":"shell","desc":"shell handler"}]`,
				},
			},
			{
				ID:       "node-2",
				Address:  "127.0.0.1:9001",
				Metadata: map[string]any{},
			},
		},
	}

	got := toExecutor(group)
	if got.Name != "aliyun" {
		t.Fatalf("Name = %q, want aliyun", got.Name)
	}
	if got.Desc != "aliyun executor" {
		t.Fatalf("Desc = %q, want aliyun executor", got.Desc)
	}
	if got.Mode != "PUSH" {
		t.Fatalf("Mode = %q, want PUSH", got.Mode)
	}
	if len(got.Nodes) != 2 {
		t.Fatalf("len(Nodes) = %d, want 2", len(got.Nodes))
	}
	if len(got.Handlers) != 1 || got.Handlers[0].Name != "shell" {
		t.Fatalf("Handlers = %#v, want shell handler", got.Handlers)
	}
}
