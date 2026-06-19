package executor

import (
	"context"
	"fmt"
	"maps"
	"sync"
)

type BindingResolveRequest struct {
	HandlerName string
	ParamKey    string
	BindingName string
	Value       string
	Params      map[string]string
	Metadata    map[string]string
}

type BindingResolver interface {
	Resolve(ctx context.Context, req BindingResolveRequest) (string, error)
}

type BindingResolverFunc func(ctx context.Context, req BindingResolveRequest) (string, error)

func (fn BindingResolverFunc) Resolve(ctx context.Context, req BindingResolveRequest) (string, error) {
	return fn(ctx, req)
}

type BindingResolverRegistry struct {
	mu        sync.RWMutex
	resolvers map[string]BindingResolver
}

func NewBindingResolverRegistry() *BindingResolverRegistry {
	return &BindingResolverRegistry{
		resolvers: make(map[string]BindingResolver),
	}
}

func (r *BindingResolverRegistry) Register(name string, resolver BindingResolver) *BindingResolverRegistry {
	if name == "" || resolver == nil {
		return r
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.resolvers[name] = resolver
	return r
}

func (r *BindingResolverRegistry) Resolve(ctx context.Context, handlerName string, params map[string]string, metadata map[string]string) (map[string]string, error) {
	if r == nil || len(metadata) == 0 {
		return nil, nil
	}

	resolvers := r.resolverSnapshot()
	if len(resolvers) == 0 {
		return nil, nil
	}

	paramsSnapshot := maps.Clone(params)
	metadataSnapshot := maps.Clone(metadata)
	resolved := make(map[string]string, len(metadataSnapshot))
	for paramKey, bindingName := range metadataSnapshot {
		resolver, ok := resolvers[bindingName]
		if !ok {
			continue
		}

		value, err := resolver.Resolve(ctx, BindingResolveRequest{
			HandlerName: handlerName,
			ParamKey:    paramKey,
			BindingName: bindingName,
			Value:       paramsSnapshot[paramKey],
			Params:      paramsSnapshot,
			Metadata:    metadataSnapshot,
		})
		if err != nil {
			return nil, fmt.Errorf("resolve binding %s for param %s failed: %w", bindingName, paramKey, err)
		}
		resolved[paramKey] = value
	}

	if len(resolved) == 0 {
		return nil, nil
	}
	return resolved, nil
}

func (r *BindingResolverRegistry) resolverSnapshot() map[string]BindingResolver {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return maps.Clone(r.resolvers)
}
