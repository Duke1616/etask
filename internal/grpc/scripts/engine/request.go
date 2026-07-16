package engine

import (
	"fmt"

	"github.com/Duke1616/etask/sdk/executor"
)

type executionRequest struct {
	code      string
	args      string
	variables string
}

func resolveRequest(task *executor.Context, adapterName string, config Config) (executionRequest, error) {
	code, err := task.GetResolvedParam("code")
	if err != nil {
		return executionRequest{}, fmt.Errorf("解析代码参数失败: %w", err)
	}
	args, err := task.GetResolvedParam("args")
	if err != nil {
		return executionRequest{}, fmt.Errorf("解析命令参数失败: %w", err)
	}
	variables, err := task.GetResolvedParam("variables")
	if err != nil {
		return executionRequest{}, fmt.Errorf("解析变量参数失败: %w", err)
	}
	if code == "" {
		return executionRequest{}, fmt.Errorf("[%s] 代码参数不能为空", adapterName)
	}
	checks := []struct {
		name  string
		value string
		limit int64
	}{
		{name: "代码", value: code, limit: config.MaxCodeSize},
		{name: "运行参数", value: args, limit: config.MaxArgsSize},
		{name: "运行变量", value: variables, limit: config.MaxVariablesSize},
	}
	for _, check := range checks {
		if int64(len(check.value)) > check.limit {
			return executionRequest{}, fmt.Errorf("%s大小超过限制: %d > %d 字节", check.name, len(check.value), check.limit)
		}
	}
	return executionRequest{code: code, args: args, variables: variables}, nil
}
