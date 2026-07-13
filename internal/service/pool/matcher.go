package pool

import "github.com/Duke1616/etask/internal/domain"

// BindingMatcher 在内存中判断某个资源池 handler 是否被授权。
type BindingMatcher struct {
	pools map[string]bindingPermission
}

type bindingPermission struct {
	wildcard domain.ExecutionPoolBindingStatus
	handlers map[string]domain.ExecutionPoolBindingStatus
}

// NewBindingMatcher 基于绑定列表构建授权匹配器。
func NewBindingMatcher(bindings []domain.ExecutionPoolBinding) BindingMatcher {
	matcher := BindingMatcher{
		pools: make(map[string]bindingPermission, len(bindings)),
	}
	for _, binding := range bindings {
		permission := matcher.pools[binding.PoolName]
		if permission.handlers == nil {
			permission.handlers = make(map[string]domain.ExecutionPoolBindingStatus)
		}
		handlerName := domain.NormalizeExecutionPoolHandlerName(binding.HandlerName)
		if handlerName == "" {
			permission.wildcard = binding.Status
		} else {
			permission.handlers[handlerName] = binding.Status
		}
		matcher.pools[binding.PoolName] = permission
	}
	return matcher
}

// Allow 判断指定资源池 handler 是否被授权；精确绑定优先于整池绑定。
func (m BindingMatcher) Allow(poolName, handlerName string) bool {
	permission, ok := m.pools[poolName]
	if !ok {
		return false
	}
	handlerName = domain.NormalizeExecutionPoolHandlerName(handlerName)
	if status, ok := permission.handlers[handlerName]; ok {
		return status == domain.ExecutionPoolBindingStatusEnabled
	}
	return permission.wildcard == domain.ExecutionPoolBindingStatusEnabled
}
