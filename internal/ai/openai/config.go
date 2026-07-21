package openai

import (
	"fmt"
	"strings"
	"time"
)

const (
	defaultEndpoint        = "https://api.openai.com/v1"
	defaultTimeout         = 3 * time.Minute
	maxTimeout             = 9 * time.Minute
	defaultMaxOutputTokens = int64(8192)
	defaultMaxConcurrency  = 4
)

// Config 描述 OpenAI Responses API 运行配置。
type Config struct {
	Endpoint        string
	Model           string
	Timeout         time.Duration
	MaxOutputTokens int64
	MaxConcurrency  int
	ReasoningEffort string
}

func (c *Config) normalize() error {
	c.Endpoint = strings.TrimRight(strings.TrimSpace(c.Endpoint), "/")
	if c.Endpoint == "" {
		c.Endpoint = defaultEndpoint
	}
	c.Model = strings.TrimSpace(c.Model)
	if c.Model == "" {
		return fmt.Errorf("OpenAI model is required")
	}
	if c.Timeout <= 0 {
		c.Timeout = defaultTimeout
	}
	if c.Timeout > maxTimeout {
		return fmt.Errorf("OpenAI timeout cannot exceed %s", maxTimeout)
	}
	if c.MaxOutputTokens <= 0 {
		c.MaxOutputTokens = defaultMaxOutputTokens
	}
	if c.MaxConcurrency <= 0 {
		c.MaxConcurrency = defaultMaxConcurrency
	}
	c.ReasoningEffort = strings.ToLower(strings.TrimSpace(c.ReasoningEffort))
	if !validReasoningEffort(c.ReasoningEffort) {
		return fmt.Errorf("unsupported OpenAI reasoning effort: %s", c.ReasoningEffort)
	}
	return nil
}

func validReasoningEffort(value string) bool {
	switch value {
	case "", "none", "minimal", "low", "medium", "high", "xhigh", "max":
		return true
	default:
		return false
	}
}
