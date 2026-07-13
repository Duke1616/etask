package pool

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	agentDomain "github.com/Duke1616/etask/internal/agent/domain"
	"github.com/Duke1616/etask/internal/domain"
	poolSource "github.com/Duke1616/etask/internal/service/pool/source"
	"github.com/Duke1616/etask/pkg/grpc/registry"
	registryEtcd "github.com/Duke1616/etask/pkg/grpc/registry/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const listNodesTimeout = 3 * time.Second

func (s *catalogService) ListNodes(ctx context.Context, pool domain.ExecutionPool) ([]Node, error) {
	if s.etcd == nil {
		return nil, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, listNodesTimeout)
	defer cancel()

	prefix := nodePrefix(pool)
	if prefix == "" {
		return nil, nil
	}

	resp, err := s.etcd.Get(queryCtx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	nodes := make([]Node, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		inst, ok := poolSource.DecodeInstance(kv)
		if !ok || !matchPoolInstance(pool, inst) {
			continue
		}
		id := strings.TrimSpace(inst.ID)
		if id == "" {
			id = strings.TrimSpace(inst.Address)
		}
		nodes = append(nodes, Node{
			ID:      id,
			Address: strings.TrimSpace(inst.Address),
		})
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].ID == nodes[j].ID {
			return nodes[i].Address < nodes[j].Address
		}
		return nodes[i].ID < nodes[j].ID
	})
	return nodes, nil
}

func nodePrefix(pool domain.ExecutionPool) string {
	switch pool.Kind {
	case domain.ExecutionPoolKindExecutor:
		return path.Join(registryEtcd.DefaultPrefix, pool.Name) + "/"
	case domain.ExecutionPoolKindAgent:
		return "/etask/kafka/" + agentDomain.ServiceName
	default:
		return ""
	}
}

func matchPoolInstance(pool domain.ExecutionPool, inst registry.ServiceInstance) bool {
	switch pool.Kind {
	case domain.ExecutionPoolKindExecutor:
		return strings.TrimSpace(inst.Name) == pool.Name && metadataString(inst.Metadata, "role") == "executor"
	case domain.ExecutionPoolKindAgent:
		return agentPoolName(inst.Name, inst.Metadata) == pool.Name
	default:
		return false
	}
}

func agentPoolName(name string, metadata map[string]any) string {
	if val := metadataString(metadata, "name"); val != "" {
		return val
	}
	return strings.TrimSpace(name)
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	if val, ok := metadata[key]; ok {
		return strings.TrimSpace(fmt.Sprintf("%v", val))
	}
	return ""
}
