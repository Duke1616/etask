package eino

import (
	"context"
	"testing"

	internalai "github.com/Duke1616/etask/internal/ai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"
)

type chatModelStub struct {
	tools    []*schema.ToolInfo
	messages []*schema.Message
}

func (s *chatModelStub) Generate(context.Context, []*schema.Message,
	...model.Option) (*schema.Message, error) {
	return nil, nil
}

func (s *chatModelStub) Stream(_ context.Context, messages []*schema.Message,
	_ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	s.messages = messages
	index := 0
	return schema.StreamReaderFromArray([]*schema.Message{
		{Role: schema.Assistant, Content: "正在修改"},
		{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{
			Index: &index, Type: "function",
			Function: schema.FunctionCall{Name: "propose_code", Arguments: `{"code":"print(1)"}`},
		}}, ResponseMeta: &schema.ResponseMeta{Usage: &schema.TokenUsage{
			PromptTokens: 12, CompletionTokens: 8,
		}}},
	}), nil
}

func (s *chatModelStub) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	copy := *s
	copy.tools = tools
	return &copy, nil
}

func TestProviderStream(t *testing.T) {
	modelStub := &chatModelStub{}
	provider, err := NewProvider(Config{
		ProviderName: "qwen", ModelName: "qwen-14b", MaxConcurrency: 1,
	}, modelStub)
	require.NoError(t, err)
	stream, err := provider.Stream(t.Context(), internalai.Request{
		Instructions: "system", Input: "user",
		Tools: []internalai.Tool{{
			Name: "propose_code", Description: "proposal",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"code": map[string]any{"type": "string"}},
				"required":   []string{"code"},
			},
		}},
	})
	require.NoError(t, err)

	events := make([]internalai.Event, 0, 3)
	for stream.Next() {
		events = append(events, stream.Current())
	}

	require.NoError(t, stream.Err())
	require.Len(t, events, 3)
	require.Equal(t, internalai.EventTypeTextDelta, events[0].Type)
	require.Equal(t, internalai.EventTypeToolCall, events[1].Type)
	require.Equal(t, "propose_code", events[1].ToolCall.Name)
	require.Equal(t, internalai.EventTypeCompleted, events[2].Type)
	require.Equal(t, int64(12), events[2].Usage.InputTokens)
	require.Equal(t, int64(8), events[2].Usage.OutputTokens)
	require.NoError(t, stream.Close())
	require.NoError(t, stream.Close())
}
