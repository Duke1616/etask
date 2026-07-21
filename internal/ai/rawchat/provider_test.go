package rawchat

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Duke1616/etask/internal/ai"
	"github.com/stretchr/testify/require"
)

func TestProviderCompletesMissingToolName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "data: {\"type\":\"response.output_item.added\",\"item\":{\"type\":\"function_call\",\"name\":\"propose_code\"}}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"type\":\"response.function_call_arguments.delta\",\"delta\":\"{\\\"code\\\":\\\"print(1)\\\",\\\"summary\\\":\\\"更新\\\"}\"}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"type\":\"response.function_call_arguments.done\",\"arguments\":\"{\\\"code\\\":\\\"print(1)\\\",\\\"summary\\\":\\\"更新\\\"}\"}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"type\":\"response.completed\",\"response\":{\"usage\":{}}}\n\n")
	}))
	defer server.Close()

	provider, err := NewProvider(Config{Endpoint: server.URL, Model: "test-model"}, "test-key")
	require.NoError(t, err)
	require.Equal(t, "rawchat", provider.Name())
	stream, err := provider.Stream(t.Context(), ai.Request{})
	require.NoError(t, err)
	defer stream.Close()

	var toolCall *ai.ToolCall
	for stream.Next() {
		if stream.Current().Type == ai.EventTypeToolCall {
			toolCall = stream.Current().ToolCall
		}
	}
	require.NoError(t, stream.Err())
	require.NotNil(t, toolCall)
	require.Equal(t, "propose_code", toolCall.Name)
}
