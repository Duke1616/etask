package ai

import "context"

// EventType 表示模型流中的事件类型。
type EventType string

const (
	EventTypeTextDelta       EventType = "TEXT_DELTA"
	EventTypeToolCallStarted EventType = "TOOL_CALL_STARTED"
	EventTypeToolCall        EventType = "TOOL_CALL"
	EventTypeCompleted       EventType = "COMPLETED"
	EventTypeFailed          EventType = "FAILED"
)

// Tool 描述允许模型调用的受控工具。
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// Request 描述一次模型响应所需的业务无关输入。
type Request struct {
	Instructions string
	Input        string
	Tools        []Tool
	UserKey      string
}

// ToolCall 表示模型生成的工具调用。
type ToolCall struct {
	Name      string
	Arguments string
}

// Usage 记录一次模型响应消耗的 Token。
type Usage struct {
	InputTokens  int64
	OutputTokens int64
}

// Event 是模型供应商输出的统一流事件。
type Event struct {
	Type     EventType
	Text     string
	ToolCall *ToolCall
	Usage    Usage
	Err      error
}

// Stream 是一次模型流式响应。
type Stream interface {
	// Next 推进到下一个事件。
	Next() bool
	// Current 返回当前事件。
	Current() Event
	// Err 返回流结束时的错误。
	Err() error
	// Close 关闭模型响应流。
	Close() error
}

// Provider 定义大模型需要提供的最小流式能力。
type Provider interface {
	// Name 返回供应商名称。
	Name() string
	// Model 返回当前使用的模型名称。
	Model() string
	// Stream 创建一次流式模型响应。
	Stream(ctx context.Context, request Request) (Stream, error)
}
