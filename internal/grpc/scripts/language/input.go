package language

import (
	"fmt"

	"github.com/Duke1616/etask/internal/grpc/scripts/engine"
)

// FileInput 将参数写入权限受控的文件，并返回统一环境变量。
func FileInput(workspace engine.Workspace, input engine.Input) ([]string, error) {
	args := input.Args
	if args == "" {
		args = "{}"
	}
	variables := input.Variables
	if variables == "" {
		variables = "[]"
	}
	argsFile, err := workspace.WriteFile("args.json", []byte(args), 0o600)
	if err != nil {
		return nil, fmt.Errorf("写入脚本参数文件失败: %w", err)
	}
	variablesFile, err := workspace.WriteFile("variables.json", []byte(variables), 0o600)
	if err != nil {
		return nil, fmt.Errorf("写入脚本变量文件失败: %w", err)
	}
	return []string{
		"ETASK_ARGS_FILE=" + argsFile,
		"ETASK_VARIABLES_FILE=" + variablesFile,
	}, nil
}
