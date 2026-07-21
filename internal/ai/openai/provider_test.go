package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internalai "github.com/Duke1616/etask/internal/ai"
	"github.com/stretchr/testify/require"
)

func TestProviderStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/v1/responses", request.URL.Path)
		require.Equal(t, "Bearer test-key", request.Header.Get("Authorization"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		require.Equal(t, "test-model", body["model"])
		require.Equal(t, false, body["store"])
		require.Equal(t, map[string]any{"effort": "low"}, body["reasoning"])

		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"正在分析\"}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"type\":\"response.function_call_arguments.delta\",\"delta\":\"{\"}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"type\":\"response.function_call_arguments.done\",\"name\":\"propose_code\",\"arguments\":\"{\\\"summary\\\":\\\"升级\\\",\\\"code\\\":\\\"print(1)\\\"}\"}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"usage\":{\"input_tokens\":4,\"output_tokens\":3}}}\n\n")
		_, _ = fmt.Fprint(writer, "data: [DONE]\n\n")
	}))
	defer server.Close()

	provider, err := NewProvider(Config{
		Endpoint: server.URL + "/v1", Model: "test-model", MaxConcurrency: 1,
		ReasoningEffort: "low",
	}, "test-key")
	require.NoError(t, err)
	stream, err := provider.Stream(context.Background(), internalai.Request{
		Instructions: "instruction", Input: "input", UserKey: "1:2",
		Tools: []internalai.Tool{{
			Name: "propose_code", Description: "proposal",
			Parameters: map[string]any{
				"type": "object", "properties": map[string]any{},
				"required": []string{}, "additionalProperties": false,
			},
		}},
	})
	require.NoError(t, err)
	defer stream.Close()

	events := make([]internalai.Event, 0, 4)
	for stream.Next() {
		events = append(events, stream.Current())
	}
	require.NoError(t, stream.Err())
	require.Len(t, events, 4)
	require.Equal(t, internalai.EventTypeTextDelta, events[0].Type)
	require.Equal(t, "正在分析", events[0].Text)
	require.Equal(t, internalai.EventTypeToolCallStarted, events[1].Type)
	require.Equal(t, internalai.EventTypeToolCall, events[2].Type)
	require.Equal(t, "propose_code", events[2].ToolCall.Name)
	require.Equal(t, internalai.EventTypeCompleted, events[3].Type)
	require.Equal(t, int64(4), events[3].Usage.InputTokens)
	require.Equal(t, int64(3), events[3].Usage.OutputTokens)
}

func TestProviderStreamReportsIncompleteResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "data: {\"type\":\"response.incomplete\",\"response\":{\"incomplete_details\":{\"reason\":\"max_output_tokens\"}}}\n\n")
		_, _ = fmt.Fprint(writer, "data: [DONE]\n\n")
	}))
	defer server.Close()

	provider, err := NewProvider(Config{
		Endpoint: server.URL, Model: "test-model", MaxConcurrency: 1,
	}, "test-key")
	require.NoError(t, err)
	stream, err := provider.Stream(t.Context(), internalai.Request{})
	require.NoError(t, err)
	defer stream.Close()

	require.True(t, stream.Next())
	event := stream.Current()
	require.Equal(t, internalai.EventTypeFailed, event.Type)
	require.ErrorContains(t, event.Err, "max_output_tokens")
}

func TestProviderStreamTimeoutIncludesConcurrencyQueue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	provider, err := NewProvider(Config{
		Endpoint: server.URL, Model: "test-model", Timeout: 30 * time.Millisecond,
		MaxConcurrency: 1,
	}, "test-key")
	require.NoError(t, err)

	first, err := provider.Stream(t.Context(), internalai.Request{})
	require.NoError(t, err)
	defer first.Close()

	startedAt := time.Now()
	_, err = provider.Stream(t.Context(), internalai.Request{})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.GreaterOrEqual(t, time.Since(startedAt), 20*time.Millisecond)
}
