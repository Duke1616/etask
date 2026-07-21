package openai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	internalai "github.com/Duke1616/etask/internal/ai"
	sdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

var _ internalai.Provider = (*Provider)(nil)

// Provider 使用 OpenAI Responses API 生成代码助手响应。
type Provider struct {
	config          Config
	client          sdk.Client
	semaphore       chan struct{}
	newEventAdapter StreamEventAdapterFactory
}

// NewProvider 创建 OpenAI 模型供应商。
func NewProvider(config Config, apiKey string, options ...Option) (*Provider, error) {
	if err := config.normalize(); err != nil {
		return nil, err
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}
	client := sdk.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(config.Endpoint+"/"),
	)
	provider := &Provider{
		config: config, client: client,
		semaphore: make(chan struct{}, config.MaxConcurrency),
	}
	for _, option := range options {
		option(provider)
	}
	return provider, nil
}

func (p *Provider) Name() string  { return "openai" }
func (p *Provider) Model() string { return p.config.Model }

// Stream 创建受并发和超时限制的 OpenAI 流式响应。
func (p *Provider) Stream(ctx context.Context, request internalai.Request) (internalai.Stream, error) {
	streamCtx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	select {
	case p.semaphore <- struct{}{}:
	case <-streamCtx.Done():
		cancel()
		return nil, streamCtx.Err()
	}

	params := responses.ResponseNewParams{
		Instructions:      sdk.String(request.Instructions),
		Input:             responses.ResponseNewParamsInputUnion{OfString: sdk.String(request.Input)},
		Model:             shared.ResponsesModel(p.config.Model),
		MaxOutputTokens:   sdk.Int(p.config.MaxOutputTokens),
		Store:             sdk.Bool(false),
		ParallelToolCalls: sdk.Bool(false),
		SafetyIdentifier:  sdk.String(safetyIdentifier(request.UserKey)),
	}
	if p.config.ReasoningEffort != "" {
		params.Reasoning = shared.ReasoningParam{
			Effort: shared.ReasoningEffort(p.config.ReasoningEffort),
		}
	}
	for _, tool := range request.Tools {
		params.Tools = append(params.Tools, responses.ToolUnionParam{
			OfFunction: &responses.FunctionToolParam{
				Name: tool.Name, Description: sdk.String(tool.Description),
				Parameters: tool.Parameters, Strict: sdk.Bool(true),
			},
		})
	}
	raw := p.client.Responses.NewStreaming(streamCtx, params)
	stream := &responseStream{
		raw: raw,
		close: func() {
			cancel()
			<-p.semaphore
		},
	}
	if p.newEventAdapter != nil {
		stream.eventAdapter = p.newEventAdapter()
	}
	return stream, nil
}

func safetyIdentifier(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

type responseStream struct {
	raw          *ssestream.Stream[responses.ResponseStreamEventUnion]
	current      internalai.Event
	toolStarted  bool
	eventAdapter StreamEventAdapter
	close        func()
	release      sync.Once
}

func (s *responseStream) Next() bool {
	for s.raw.Next() {
		event := s.raw.Current()
		if s.eventAdapter != nil {
			event = s.eventAdapter.Adapt(event)
		}
		switch event.Type {
		case "response.output_text.delta":
			s.current = internalai.Event{Type: internalai.EventTypeTextDelta, Text: event.Delta}
			return true
		case "response.refusal.delta":
			s.current = internalai.Event{Type: internalai.EventTypeTextDelta, Text: event.Delta}
			return true
		case "response.function_call_arguments.delta":
			if s.startToolCall() {
				return true
			}
		case "response.function_call_arguments.done":
			s.current = internalai.Event{Type: internalai.EventTypeToolCall,
				ToolCall: &internalai.ToolCall{Name: event.Name, Arguments: event.Arguments}}
			return true
		case "response.completed":
			if reason := event.Response.IncompleteDetails.Reason; reason != "" {
				s.current = incompleteEvent(reason)
				return true
			}
			s.current = internalai.Event{
				Type: internalai.EventTypeCompleted,
				Usage: internalai.Usage{
					InputTokens:  event.Response.Usage.InputTokens,
					OutputTokens: event.Response.Usage.OutputTokens,
				},
			}
			return true
		case "response.incomplete":
			s.current = incompleteEvent(event.Response.IncompleteDetails.Reason)
			return true
		case "error":
			s.current = internalai.Event{Type: internalai.EventTypeFailed,
				Err: fmt.Errorf("OpenAI stream error: %s", event.Message)}
			return true
		case "response.failed":
			s.current = internalai.Event{Type: internalai.EventTypeFailed,
				Err: fmt.Errorf("OpenAI response failed: %s", event.Response.Error.Message)}
			return true
		}
	}
	s.releaseResources()
	return false
}

func (s *responseStream) startToolCall() bool {
	if s.toolStarted {
		return false
	}
	s.toolStarted = true
	s.current = internalai.Event{Type: internalai.EventTypeToolCallStarted}
	return true
}

func incompleteEvent(reason string) internalai.Event {
	if reason == "" {
		reason = "unknown reason"
	}
	return internalai.Event{
		Type: internalai.EventTypeFailed,
		Err:  fmt.Errorf("OpenAI response incomplete: %s", reason),
	}
}

func (s *responseStream) Current() internalai.Event { return s.current }
func (s *responseStream) Err() error                { return s.raw.Err() }

func (s *responseStream) Close() error {
	err := s.raw.Close()
	s.releaseResources()
	return err
}

func (s *responseStream) releaseResources() {
	s.release.Do(func() {
		if s.close != nil {
			s.close()
		}
	})
}
