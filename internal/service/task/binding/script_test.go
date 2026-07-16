package binding

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Duke1616/etask/internal/domain"
	codebookmocks "github.com/Duke1616/etask/internal/service/codebook/mocks"
	runnermocks "github.com/Duke1616/etask/internal/service/runner/mocks"
	"go.uber.org/mock/gomock"
)

func TestScriptBindingResolversResolve(t *testing.T) {
	ctrl := gomock.NewController(t)
	codebookSvc := codebookmocks.NewMockService(ctrl)
	runnerSvc := runnermocks.NewMockService(ctrl)

	codebookSvc.EXPECT().
		GetByID(gomock.Any(), int64(12)).
		Return(domain.Codebook{ID: 12, TenantID: 3, Scope: domain.CodebookScopeTenant, ProjectID: 7, Code: "echo hello"}, nil)
	runnerSvc.EXPECT().
		ListMergedVariables(gomock.Any(), int64(34)).
		Return([]domain.RunnerVariable{
			{Key: "HOST", Value: "127.0.0.1"},
			{Key: "TOKEN", Value: "secret", Secret: true},
		}, nil)

	resolvers := NewScriptBindingResolvers(codebookSvc, runnerSvc)

	params, err := resolvers.Resolve(context.Background(), "shell", map[string]string{
		"code":      "12",
		"args":      `{"name":"demo"}`,
		"variables": "34",
	}, map[string]string{
		"code":      "codebook",
		"variables": "runner",
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if params["code"] != "echo hello" {
		t.Fatalf("code = %q, want %q", params["code"], "echo hello")
	}
	if _, ok := params["args"]; ok {
		t.Fatalf("args should not be materialized")
	}

	var variables []variable
	if err = json.Unmarshal([]byte(params["variables"]), &variables); err != nil {
		t.Fatalf("variables should be json: %v", err)
	}
	if len(variables) != 2 || variables[0].Key != "HOST" || variables[1].Secret != true {
		t.Fatalf("variables = %+v", variables)
	}
}

func TestScriptBindingResolversResolveInvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	resolvers := NewScriptBindingResolvers(
		codebookmocks.NewMockService(ctrl),
		runnermocks.NewMockService(ctrl),
	)

	_, err := resolvers.Resolve(context.Background(), "shell", map[string]string{"code": "bad"}, map[string]string{"code": "codebook"})
	if err == nil || !strings.Contains(err.Error(), "绑定 ID 非法") {
		t.Fatalf("err = %v, want invalid binding id", err)
	}
}
