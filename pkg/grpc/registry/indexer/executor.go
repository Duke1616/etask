package indexer

import (
	"fmt"

	"github.com/Duke1616/etask/pkg/grpc/registry"
)

const ExecutorPrefix = "/grpc/executor"

const executorIndexName = "executor"

type ExecutorIndexer struct{}

func NewExecutorIndexer() ExecutorIndexer {
	return ExecutorIndexer{}
}

func (ExecutorIndexer) Name() string {
	return executorIndexName
}

func (ExecutorIndexer) Key(si registry.ServiceInstance) (string, bool) {
	if !isExecutor(si) {
		return "", false
	}
	return fmt.Sprintf("%s/%s/%s", ExecutorPrefix, si.Name, si.Address), true
}

func isExecutor(si registry.ServiceInstance) bool {
	if si.Metadata == nil {
		return false
	}
	role, ok := si.Metadata["role"].(string)
	return ok && role == "executor"
}
