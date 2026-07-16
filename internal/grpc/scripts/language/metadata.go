package language

import "github.com/Duke1616/etask/sdk/executor"

// Metadata 返回脚本处理器共用的参数定义。
func Metadata(language string) []executor.Parameter {
	return []executor.Parameter{
		{
			Key: "code", Desc: "脚本代码内容", Required: true,
			Bindings: map[string]executor.Binding{
				"static": &executor.BindingOption{
					Label: "手动输入", Placeholder: "请输入脚本代码...", Component: "code-editor",
					Config: map[string]string{"language": language},
				},
				"codebook": &executor.BindingOption{
					Label: "脚本库引用", Placeholder: "请选择脚本库...", Component: "codebook-picker",
				},
			},
		},
		{
			Key: "args", Desc: "脚本执行参数",
			Bindings: map[string]executor.Binding{
				"static": &executor.BindingOption{
					Label: "参数内容 (JSON)", Placeholder: `{"name": "zhangsan", "age": 18}`,
					Component: "code-editor", Config: map[string]string{"language": "json"},
				},
			},
		},
		{
			Key: "variables", Desc: "环境变量",
			Bindings: map[string]executor.Binding{
				"static": &executor.BindingOption{
					Label: "手动输入", Placeholder: `[{"key": "K1", "value": "V1", "secret": false}]`,
					Component: "kv-input",
				},
				"runner": &executor.BindingOption{
					Label: "执行单元引用", Placeholder: "请选择执行单元...", Component: "runner-picker",
				},
			},
		},
	}
}
