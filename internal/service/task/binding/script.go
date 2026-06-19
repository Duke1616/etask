package binding

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Duke1616/etask/internal/domain"
	codebookSvc "github.com/Duke1616/etask/internal/service/codebook"
	runnerSvc "github.com/Duke1616/etask/internal/service/runner"
	"github.com/Duke1616/etask/sdk/executor"
	"github.com/samber/lo"
)

const (
	CodebookBinding = "codebook"
	RunnerBinding   = "runner"
)

func NewScriptBindingResolvers(codebookSvc codebookSvc.Service, runnerSvc runnerSvc.Service) *executor.BindingResolverRegistry {
	return executor.NewBindingResolverRegistry().
		Register(CodebookBinding, CodebookResolver{svc: codebookSvc}).
		Register(RunnerBinding, RunnerResolver{svc: runnerSvc})
}

type CodebookResolver struct {
	svc codebookSvc.Service
}

func (r CodebookResolver) Resolve(ctx context.Context, req executor.BindingResolveRequest) (string, error) {
	id, err := parseID(req.Value, req.ParamKey)
	if err != nil {
		return "", err
	}

	codebook, err := r.svc.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get codebook failed: %w", err)
	}
	return codebook.Code, nil
}

type RunnerResolver struct {
	svc runnerSvc.Service
}

func (r RunnerResolver) Resolve(ctx context.Context, req executor.BindingResolveRequest) (string, error) {
	id, err := parseID(req.Value, req.ParamKey)
	if err != nil {
		return "", err
	}

	runner, err := r.svc.FindByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get runner failed: %w", err)
	}

	variables := lo.Map(runner.Variables, func(v domain.RunnerVariable, _ int) variable {
		return variable{
			Key:    v.Key,
			Value:  v.Value,
			Secret: v.Secret,
		}
	})

	bytes, err := json.Marshal(variables)
	if err != nil {
		return "", fmt.Errorf("marshal runner variables failed: %w", err)
	}
	return string(bytes), nil
}

func parseID(rawID string, param string) (int64, error) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s binding id: %q", param, rawID)
	}
	return id, nil
}

type variable struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

var _ executor.BindingResolver = CodebookResolver{}
var _ executor.BindingResolver = RunnerResolver{}
