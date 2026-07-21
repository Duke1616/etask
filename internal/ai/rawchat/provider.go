package rawchat

import (
	"fmt"

	"github.com/Duke1616/etask/internal/ai"
	openaiProvider "github.com/Duke1616/etask/internal/ai/openai"
)

var _ ai.Provider = (*Provider)(nil)

// Config 复用 RawChat 所兼容的 OpenAI Responses 运行配置。
type Config = openaiProvider.Config

// Provider 为 RawChat 的 Responses 兼容差异提供独立边界。
type Provider struct {
	*openaiProvider.Provider
}

// NewProvider 创建 RawChat 模型供应商。
func NewProvider(config Config, apiKey string) (*Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("RAWCHAT_API_KEY is required")
	}
	provider, err := openaiProvider.NewProvider(config, apiKey,
		openaiProvider.WithStreamEventAdapter(func() openaiProvider.StreamEventAdapter {
			return &streamEventAdapter{}
		}),
	)
	if err != nil {
		return nil, err
	}
	return &Provider{Provider: provider}, nil
}

func (p *Provider) Name() string { return "rawchat" }
