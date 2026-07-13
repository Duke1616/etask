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
	nodes, err := s.ListNodesForPools(ctx, []domain.ExecutionPool{pool})
	if err != nil {
		return nil, err
	}
	return nodes[pool.Name], nil
}

// ListNodesForPools 每种注册前缀只查询一次，避免资源目录列表的 N+1 etcd 请求。
func (s *catalogService) ListNodesForPools(ctx context.Context, pools []domain.ExecutionPool) (map[string][]Node, error) {
	result := make(map[string][]Node, len(pools))
	if s.etcd == nil {
		return result, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, listNodesTimeout)
	defer cancel()

	poolsByKindAndName := make(map[domain.ExecutionPoolKind]map[string]domain.ExecutionPool)
	for _, pool := range pools {
		byName := poolsByKindAndName[pool.Kind]
		if byName == nil {
			byName = make(map[string]domain.ExecutionPool)
			poolsByKindAndName[pool.Kind] = byName
		}
		byName[pool.Name] = pool
	}

	for kind, poolsByName := range poolsByKindAndName {
		prefix := nodePrefix(domain.ExecutionPool{Kind: kind})
		if prefix == "" {
			continue
		}
		resp, err := s.etcd.Get(queryCtx, prefix, clientv3.WithPrefix())
		if err != nil {
			return nil, err
		}
		for _, kv := range resp.Kvs {
			inst, ok := poolSource.DecodeInstance(kv)
			if !ok {
				continue
			}
			poolName := instancePoolName(kind, inst)
			pool, exists := poolsByName[poolName]
			if !exists || !matchPoolInstance(pool, inst) {
				continue
			}
			id := strings.TrimSpace(inst.ID)
			if id == "" {
				id = strings.TrimSpace(inst.Address)
			}
			result[poolName] = append(result[poolName], Node{ID: id, Address: strings.TrimSpace(inst.Address)})
		}
	}

	for poolName := range result {
		sort.Slice(result[poolName], func(i, j int) bool {
			if result[poolName][i].ID == result[poolName][j].ID {
				return result[poolName][i].Address < result[poolName][j].Address
			}
			return result[poolName][i].ID < result[poolName][j].ID
		})
	}
	return result, nil
}

func instancePoolName(kind domain.ExecutionPoolKind, inst registry.ServiceInstance) string {
	switch kind {
	case domain.ExecutionPoolKindExecutor:
		if metadataString(inst.Metadata, "role") != "executor" {
			return ""
		}
		return strings.TrimSpace(inst.Name)
	case domain.ExecutionPoolKindAgent:
		return agentPoolName(inst.Name, inst.Metadata)
	default:
		return ""
	}
}

func nodePrefix(pool domain.ExecutionPool) string {
	switch pool.Kind {
	case domain.ExecutionPoolKindExecutor:
		if pool.Name == "" {
			return registryEtcd.DefaultPrefix + "/"
		}
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
