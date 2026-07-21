package codeassist

import "github.com/Duke1616/etask/internal/ai"

const proposeCodeToolName = "propose_code"

func proposalTool() ai.Tool {
	return ai.Tool{
		Name:        proposeCodeToolName,
		Description: "提交完整的候选脚本代码。只有用户明确要求生成、修改或修复代码时才调用。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"summary": map[string]any{"type": "string"},
				"code":    map[string]any{"type": "string"},
			},
			"required":             []string{"summary", "code"},
			"additionalProperties": false,
		},
	}
}
