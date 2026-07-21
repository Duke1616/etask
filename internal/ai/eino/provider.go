package eino

import (
	"context"
	"encoding/json"
	"fmt"

	internalai "github.com/Duke1616/etask/internal/ai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
)

var _ internalai.Provider = (*Provider)(nil)

// Provider 将 Eino ToolCallingChatModel 适配为 etask 的稳定模型边界。
type Provider struct {
	config    Config
	model     model.ToolCallingChatModel
	semaphore chan struct{}
}

// NewProvider 创建 Eino 模型供应商。
func NewProvider(config Config, chatModel model.ToolCallingChatModel) (*Provider, error) {
	if err := config.normalize(); err != nil {
		return nil, err
	}
	if chatModel == nil {
		return nil, fmt.Errorf("Eino chat model is required")
	}
	return &Provider{
		config: config, model: chatModel,
		semaphore: make(chan struct{}, config.MaxConcurrency),
	}, nil
}

func (p *Provider) Name() string  { return p.config.ProviderName }
func (p *Provider) Model() string { return p.config.ModelName }

// Stream 创建受并发和超时限制的 Eino 流式响应。
func (p *Provider) Stream(ctx context.Context, request internalai.Request) (internalai.Stream, error) {
	streamCtx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	select {
	case p.semaphore <- struct{}{}:
	case <-streamCtx.Done():
		cancel()
		return nil, streamCtx.Err()
	}
	release := func() {
		cancel()
		<-p.semaphore
	}

	chatModel := p.model
	if len(request.Tools) > 0 {
		tools, err := toEinoTools(request.Tools)
		if err != nil {
			release()
			return nil, err
		}
		chatModel, err = chatModel.WithTools(tools)
		if err != nil {
			release()
			return nil, fmt.Errorf("bind Eino tools: %w", err)
		}
	}
	reader, err := chatModel.Stream(streamCtx, []*schema.Message{
		schema.SystemMessage(request.Instructions),
		schema.UserMessage(request.Input),
	})
	if err != nil {
		release()
		return nil, fmt.Errorf("start Eino stream: %w", err)
	}
	if reader == nil {
		release()
		return nil, fmt.Errorf("Eino stream reader is nil")
	}
	return newResponseStream(reader, release), nil
}

func toEinoTools(tools []internalai.Tool) ([]*schema.ToolInfo, error) {
	result := make([]*schema.ToolInfo, 0, len(tools))
	for _, source := range tools {
		encoded, err := json.Marshal(source.Parameters)
		if err != nil {
			return nil, fmt.Errorf("encode Eino tool %s: %w", source.Name, err)
		}
		parameters := &jsonschema.Schema{}
		if err = json.Unmarshal(encoded, parameters); err != nil {
			return nil, fmt.Errorf("decode Eino tool %s: %w", source.Name, err)
		}
		result = append(result, &schema.ToolInfo{
			Name: source.Name, Desc: source.Description,
			ParamsOneOf: schema.NewParamsOneOfByJSONSchema(parameters),
		})
	}
	return result, nil
}
