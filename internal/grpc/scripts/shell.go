package scripts

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/Duke1616/etask/sdk/executor"
)

// ShellTaskHandler Shell 任务处理器
type ShellTaskHandler struct {
	executor *ScriptExecutor
}

func NewShellTaskHandler() *ShellTaskHandler {
	return &ShellTaskHandler{
		executor: NewScriptExecutor(
			"shell",
			"scripts-*.sh",
			createShellCmd,
			prepareShellVars,
		),
	}
}

func (h *ShellTaskHandler) Name() string {
	return "shell"
}

func (h *ShellTaskHandler) Desc() string {
	return "执行 Shell 脚本命令的基础处理器"
}

func (h *ShellTaskHandler) Metadata() []executor.Parameter {
	return []executor.Parameter{
		{
			Key:      "code",
			Desc:     "脚本代码内容",
			Required: true,
			Bindings: map[string]executor.BindingOption{
				"static": {
					Label:       "手动输入",
					Placeholder: "请输入 Shell 脚本代码...",
					Component:   "code-editor",
					Config:      map[string]string{"language": "shell"},
				},
				"codebook": {
					Label:       "脚本库引用",
					Placeholder: "请选择脚本库...",
					Component:   "codebook-picker",
				},
			},
		},
		{
			Key:      "args",
			Desc:     "脚本执行参数",
			Required: false,
			Bindings: map[string]executor.BindingOption{
				"static": {
					Label:       "参数内容 (JSON)",
					Placeholder: `{"name": "zhangsan", "age": 18}`,
					Component:   "code-editor",
					Config:      map[string]string{"language": "json"},
				},
			},
		},
		{
			Key:      "variables",
			Desc:     "环境变量",
			Required: false,
			Bindings: map[string]executor.BindingOption{
				"static": {
					Label:       "手动输入",
					Placeholder: `[{"key": "K1", "value": "V1", "secret": false}]`,
					Component:   "kv-input",
				},
				"runner": {
					Label:       "执行单元引用",
					Placeholder: "请选择执行单元...",
					Component:   "runner-picker",
				},
			},
		},
	}
}

func (h *ShellTaskHandler) Run(ctx *executor.Context) error {
	return h.executor.Run(ctx)
}

// ---------------------------
// Shell 特定逻辑
// ---------------------------

func createShellCmd(codeFile string, args string, varsFile string) (*exec.Cmd, error) {
	shell := "/bin/bash"
	if _, err := exec.LookPath(shell); err != nil {
		shell = "/bin/sh"
	}
	// 使用 -e 开启 fail-fast 模式，确保脚本中任一命令失败即停止并返回非零退出码
	return exec.Command(shell, "-e", codeFile, args, varsFile), nil
}

// prepareShellVars 将 JSON 变量转换为 KEY=VALUE 格式的临时文件
func prepareShellVars(varsJSON string) (string, error) {
	if varsJSON == "" {
		return "", nil
	}

	// 解析 JSON
	var variables []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(varsJSON), &variables); err != nil {
		return "", err
	}

	// 转换为 shell 变量格式 KEY=VALUE
	var content string
	for _, v := range variables {
		content += fmt.Sprintf("%s=%v\n", v.Key, v.Value)
	}

	// 创建变量文件
	return createTempFile("scripts-*.vars", []byte(content))
}
