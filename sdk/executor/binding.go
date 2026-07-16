package executor

import "github.com/Duke1616/etask/sdk/executor/internal/binding"

type (
	// BindingResolveRequest 描述一次业务参数绑定解析请求。
	BindingResolveRequest = binding.ResolveRequest
	// BindingResolver 定义一种业务参数绑定解析能力。
	BindingResolver = binding.Resolver
	// BindingResolverFunc 将函数适配为 BindingResolver。
	BindingResolverFunc = binding.ResolverFunc
	// BindingResolverRegistry 管理业务参数绑定解析器。
	BindingResolverRegistry = binding.Registry
)

// NewBindingResolverRegistry 创建参数绑定解析器注册中心。
func NewBindingResolverRegistry() *BindingResolverRegistry {
	return binding.NewRegistry()
}
