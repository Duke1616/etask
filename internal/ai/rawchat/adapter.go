package rawchat

import "github.com/openai/openai-go/v3/responses"

// streamEventAdapter 补全 RawChat 未在参数完成事件中重复返回的工具名称。
// Provider 已禁用并行工具调用，因此一条流只需维护当前工具名称。
type streamEventAdapter struct {
	toolName string
}

func (a *streamEventAdapter) Adapt(event responses.ResponseStreamEventUnion) responses.ResponseStreamEventUnion {
	if event.Type == "response.output_item.added" &&
		event.Item.Type == "function_call" && event.Item.Name != "" {
		a.toolName = event.Item.Name
	}
	if event.Type == "response.function_call_arguments.done" && event.Name == "" {
		event.Name = a.toolName
	}
	return event
}
