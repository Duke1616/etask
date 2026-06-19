package executor

import (
	"encoding/json"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/samber/lo"
)

func toExecutor(group executorGroup) domain.Executor {
	exec := domain.Executor{Name: group.Name}
	for _, inst := range group.Instances {
		mergeMetadata(&exec, inst.Metadata)
	}
	exec.Nodes = lo.Map(group.Instances, func(inst serviceInstance, _ int) domain.ExecutorNode {
		return domain.ExecutorNode{
			ID:      inst.ID,
			Address: inst.Address,
		}
	})
	return exec
}

func mergeMetadata(exec *domain.Executor, metadata map[string]any) {
	if metadata == nil {
		return
	}
	if exec.Desc == "" {
		if desc, ok := metadata["desc"].(string); ok {
			exec.Desc = desc
		}
	}
	if exec.Mode == "" {
		if mode, ok := metadata["mode"].(string); ok {
			exec.Mode = mode
		}
	}
	if len(exec.Handlers) == 0 {
		exec.Handlers = parseHandlers(metadata["supported_handlers"])
	}
}

func parseHandlers(data any) []domain.ExecutorHandler {
	var handlers []domain.ExecutorHandler
	if bytes, ok := data.(string); ok {
		_ = json.Unmarshal([]byte(bytes), &handlers)
	}
	return handlers
}
