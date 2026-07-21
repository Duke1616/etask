package ioc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Duke1616/etask/internal/ai"
	"github.com/stretchr/testify/require"
)

func TestInitRawChatProviderUsesDedicatedAdapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "data: {\"type\":\"response.output_item.added\",\"item\":{\"type\":\"function_call\",\"name\":\"propose_code\"}}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"type\":\"response.function_call_arguments.delta\",\"delta\":\"{}\"}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"type\":\"response.function_call_arguments.done\",\"arguments\":\"{}\"}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"type\":\"response.completed\",\"response\":{\"usage\":{}}}\n\n")
	}))
	defer server.Close()
	t.Setenv("RAWCHAT_API_KEY", "test-key")

	provider := initRawChatProvider(aiConfig{Endpoint: server.URL, Model: "test-model"})
	require.Equal(t, "rawchat", provider.Name())
	stream, err := provider.Stream(t.Context(), ai.Request{})
	require.NoError(t, err)
	defer stream.Close()

	var toolName string
	for stream.Next() {
		if stream.Current().ToolCall != nil {
			toolName = stream.Current().ToolCall.Name
		}
	}
	require.NoError(t, stream.Err())
	require.Equal(t, "propose_code", toolName)
}

func TestInitQwenProviderStreamsThroughEino(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/v1/chat/completions", request.URL.Path)
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"chat-1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"正在分析\"}}]}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"chat-1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":5,\"completion_tokens\":3,\"total_tokens\":8}}\n\n")
		_, _ = fmt.Fprint(writer, "data: [DONE]\n\n")
	}))
	defer server.Close()

	provider := initQwenProvider(aiConfig{
		Endpoint: server.URL + "/v1", Model: "qwen-14b", MaxOutputTokens: 128,
	})
	require.Equal(t, "qwen", provider.Name())
	stream, err := provider.Stream(t.Context(), ai.Request{
		Instructions: "system", Input: "user",
	})
	require.NoError(t, err)
	defer stream.Close()

	events := make([]ai.Event, 0, 2)
	for stream.Next() {
		events = append(events, stream.Current())
	}

	require.NoError(t, stream.Err())
	require.Len(t, events, 2)
	require.Equal(t, ai.EventTypeTextDelta, events[0].Type)
	require.Equal(t, "正在分析", events[0].Text)
	require.Equal(t, ai.EventTypeCompleted, events[1].Type)
}
