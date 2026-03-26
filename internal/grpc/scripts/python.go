package scripts

import (
	"os/exec"

	"github.com/Duke1616/etask/sdk/executor"
)

// PythonTaskHandler Python 任务处理器
type PythonTaskHandler struct {
	executor *ScriptExecutor
}

func NewPythonTaskHandler() *PythonTaskHandler {
	return &PythonTaskHandler{
		executor: NewScriptExecutor(
			"python",
			"scripts-*.py",
			createPythonCmd,
			passThroughVars,
		),
	}
}

func (h *PythonTaskHandler) Name() string {
	return "python"
}

func (h *PythonTaskHandler) Desc() string {
	return "执行 Python 脚本代码的基础处理器"
}

func (h *PythonTaskHandler) Metadata() []executor.Parameter {
	return []executor.Parameter{
		{
			Key:      "code",
			Desc:     "脚本代码内容",
			Required: true,
			Bindings: map[string]executor.Binding{
				"static": &executor.BindingOption{
					Label:       "手动输入",
					Placeholder: "请输入 Python 脚本代码...",
					Component:   "code-editor",
					Config:      map[string]string{"language": "python"},
				},
				"codebook": &executor.BindingOption{
					Label:       "脚本库引用",
					Placeholder: "请选择脚本库...",
					Component:   "codebook-picker",
					Resolver: func(ctx *executor.Context, value string) (string, error) {
						return value, nil
					},
				},
			},
		},
		{
			Key:      "args",
			Desc:     "脚本执行参数",
			Required: false,
			Bindings: map[string]executor.Binding{
				"static": &executor.BindingOption{
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
			Bindings: map[string]executor.Binding{
				"static": &executor.BindingOption{
					Label:       "手动输入",
					Placeholder: `[{"key": "K1", "value": "V1", "secret": false}]`,
					Component:   "kv-input",
				},
				"runner": &executor.BindingOption{
					Label:       "执行单元引用",
					Placeholder: "请选择执行单元...",
					Component:   "runner-picker",
					Resolver: func(ctx *executor.Context, value string) (string, error) {
						return value, nil
					},
				},
			},
		},
	}
}

func (h *PythonTaskHandler) Run(ctx *executor.Context) error {
	return h.executor.Run(ctx)
}

// ---------------------------
// Python 特定逻辑
// ---------------------------

func createPythonCmd(codeFile string, args string, varsContent string) (*exec.Cmd, error) {
	return exec.Command("python", codeFile, args, varsContent), nil
}

// passThroughVars 直接透传变量字符串 (Python 直接解析 JSON)
func passThroughVars(varsJSON string) (string, error) {
	return varsJSON, nil
}
