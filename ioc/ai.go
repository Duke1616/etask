package ioc

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Duke1616/etask/internal/ai"
	einoProvider "github.com/Duke1616/etask/internal/ai/eino"
	openaiProvider "github.com/Duke1616/etask/internal/ai/openai"
	rawchatProvider "github.com/Duke1616/etask/internal/ai/rawchat"
	"github.com/Duke1616/etask/pkg/config"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/gotomicro/ego/core/elog"
)

type aiConfig struct {
	Provider        string        `mapstructure:"provider"`
	Endpoint        string        `mapstructure:"endpoint"`
	Model           string        `mapstructure:"model"`
	Timeout         time.Duration `mapstructure:"timeout"`
	MaxOutputTokens int64         `mapstructure:"max_output_tokens"`
	MaxConcurrency  int           `mapstructure:"max_concurrency"`
	ReasoningEffort string        `mapstructure:"reasoning_effort"`
	EnableThinking  *bool         `mapstructure:"enable_thinking"`
}

// InitAIProvider 根据调度中心配置创建模型供应商。
// 未配置或配置非法时仍允许 Scheduler 启动，AI 接口会返回稳定的不可用错误。
func InitAIProvider() ai.Provider {
	var cfg aiConfig
	if err := config.UnmarshalKey("ai", &cfg); err != nil {
		return unavailableAIProvider(err)
	}
	providerName := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if providerName == "" {
		return ai.NewUnavailableProvider("AI provider is not configured")
	}
	switch providerName {
	case "openai":
		return initOpenAIProvider(cfg)
	case "rawchat":
		return initRawChatProvider(cfg)
	case "qwen":
		return initQwenProvider(cfg)
	default:
		return unavailableAIProvider(fmt.Errorf("unsupported AI provider: %s", providerName))
	}
}

func initOpenAIProvider(cfg aiConfig) ai.Provider {
	provider, err := openaiProvider.NewProvider(responsesConfig(cfg), os.Getenv("OPENAI_API_KEY"))
	if err != nil {
		return unavailableAIProvider(err)
	}
	return provider
}

func initRawChatProvider(cfg aiConfig) ai.Provider {
	provider, err := rawchatProvider.NewProvider(responsesConfig(cfg), os.Getenv("RAWCHAT_API_KEY"))
	if err != nil {
		return unavailableAIProvider(err)
	}
	return provider
}

func responsesConfig(cfg aiConfig) openaiProvider.Config {
	return openaiProvider.Config{
		Endpoint: cfg.Endpoint, Model: cfg.Model, Timeout: cfg.Timeout,
		MaxOutputTokens: cfg.MaxOutputTokens, MaxConcurrency: cfg.MaxConcurrency,
		ReasoningEffort: cfg.ReasoningEffort,
	}
}

func initQwenProvider(cfg aiConfig) ai.Provider {
	if strings.TrimSpace(cfg.Endpoint) == "" || strings.TrimSpace(cfg.Model) == "" {
		return unavailableAIProvider(fmt.Errorf("Qwen endpoint and model are required"))
	}
	maxTokens := int(cfg.MaxOutputTokens)
	if maxTokens <= 0 {
		maxTokens = 8192
	}
	chatModel, err := qwen.NewChatModel(context.Background(), &qwen.ChatModelConfig{
		APIKey: os.Getenv("QWEN_API_KEY"), BaseURL: cfg.Endpoint,
		Model: cfg.Model, Timeout: cfg.Timeout, MaxTokens: &maxTokens,
		EnableThinking: cfg.EnableThinking,
	})
	if err != nil {
		return unavailableAIProvider(err)
	}
	provider, err := einoProvider.NewProvider(einoProvider.Config{
		ProviderName: "qwen", ModelName: cfg.Model,
		Timeout: cfg.Timeout, MaxConcurrency: cfg.MaxConcurrency,
	}, chatModel)
	if err != nil {
		return unavailableAIProvider(err)
	}
	return provider
}

func unavailableAIProvider(err error) ai.Provider {
	elog.DefaultLogger.Error("AI 模型供应商初始化失败", elog.FieldErr(err))
	return ai.NewUnavailableProvider(err.Error())
}
