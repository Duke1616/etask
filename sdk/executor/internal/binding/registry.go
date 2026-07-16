// Package binding 实现调度侧参数绑定解析。
package binding

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"sync"
)

// ResolveRequest 描述一次业务参数绑定解析请求。
type ResolveRequest struct {
	HandlerName string
	ParamKey    string
	BindingName string
	Value       string
	Params      map[string]string
	Metadata    map[string]string
}

// Resolver 定义单个绑定类型的解析能力。
type Resolver interface {
	// Resolve 解析绑定值并返回任务实际使用的参数内容。
	Resolve(ctx context.Context, req ResolveRequest) (string, error)
}

// ResolverFunc 将函数适配为 Resolver。
type ResolverFunc func(ctx context.Context, req ResolveRequest) (string, error)

// Resolve 调用绑定解析函数。
func (fn ResolverFunc) Resolve(ctx context.Context, req ResolveRequest) (string, error) {
	return fn(ctx, req)
}

// Registry 并发安全地管理业务参数绑定解析器。
type Registry struct {
	mu        sync.RWMutex
	resolvers map[string]Resolver
}

// NewRegistry 创建空的绑定解析器注册中心。
func NewRegistry() *Registry {
	return &Registry{
		resolvers: make(map[string]Resolver),
	}
}

// Register 注册或替换指定名称的绑定解析器。
func (r *Registry) Register(name string, resolver Resolver) *Registry {
	if name == "" || resolver == nil {
		return r
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.resolvers[name] = resolver
	return r
}

// Resolve 按参数名稳定排序并解析已注册的绑定。
func (r *Registry) Resolve(ctx context.Context, handlerName string, params map[string]string,
	metadata map[string]string) (map[string]string, error) {
	if r == nil || len(metadata) == 0 {
		return nil, nil
	}

	resolvers := r.resolverSnapshot()
	if len(resolvers) == 0 {
		return nil, nil
	}

	// 解析过程只使用快照，注册变更或调用方并发修改不会影响本次结果。
	paramsSnapshot := maps.Clone(params)
	metadataSnapshot := maps.Clone(metadata)
	resolved := make(map[string]string, len(metadataSnapshot))
	keys := make([]string, 0, len(metadataSnapshot))
	for paramKey := range metadataSnapshot {
		keys = append(keys, paramKey)
	}
	// 固定解析顺序，使错误定位和测试结果保持稳定。
	sort.Strings(keys)
	for _, paramKey := range keys {
		bindingName := metadataSnapshot[paramKey]
		resolver, ok := resolvers[bindingName]
		if !ok {
			continue
		}

		value, err := resolver.Resolve(ctx, ResolveRequest{
			HandlerName: handlerName,
			ParamKey:    paramKey,
			BindingName: bindingName,
			Value:       paramsSnapshot[paramKey],
			Params:      paramsSnapshot,
			Metadata:    metadataSnapshot,
		})
		if err != nil {
			return nil, fmt.Errorf("解析参数 %s 的 %s 绑定失败: %w", paramKey, bindingName, err)
		}
		resolved[paramKey] = value
	}

	if len(resolved) == 0 {
		return nil, nil
	}
	return resolved, nil
}

func (r *Registry) resolverSnapshot() map[string]Resolver {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return maps.Clone(r.resolvers)
}
