package openai

import "github.com/openai/openai-go/v3/responses"

// StreamEventAdapter 在标准事件解析前规范化兼容服务的流事件。
// 每条响应流必须使用独立实例，避免跨请求共享组装状态。
type StreamEventAdapter interface {
	// Adapt 返回标准事件解析器可以直接消费的事件。
	Adapt(event responses.ResponseStreamEventUnion) responses.ResponseStreamEventUnion
}

// StreamEventAdapterFactory 为每条响应流创建独立的事件适配器。
type StreamEventAdapterFactory func() StreamEventAdapter

// Option 配置 OpenAI Responses Provider 的可选扩展。
type Option func(*Provider)

// WithStreamEventAdapter 注入兼容服务的流事件适配器。
func WithStreamEventAdapter(factory StreamEventAdapterFactory) Option {
	return func(provider *Provider) {
		provider.newEventAdapter = factory
	}
}
