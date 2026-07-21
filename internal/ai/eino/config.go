package eino

import (
	"fmt"
	"strings"
	"time"
)

const (
	defaultTimeout        = 3 * time.Minute
	maxTimeout            = 9 * time.Minute
	defaultMaxConcurrency = 4
)

// Config 描述 Eino ChatModel 适配层的运行限制。
type Config struct {
	ProviderName   string
	ModelName      string
	Timeout        time.Duration
	MaxConcurrency int
}

func (c *Config) normalize() error {
	c.ProviderName = strings.TrimSpace(c.ProviderName)
	c.ModelName = strings.TrimSpace(c.ModelName)
	if c.ProviderName == "" || c.ModelName == "" {
		return fmt.Errorf("Eino provider name and model are required")
	}
	if c.Timeout <= 0 {
		c.Timeout = defaultTimeout
	}
	if c.Timeout > maxTimeout {
		return fmt.Errorf("Eino timeout cannot exceed %s", maxTimeout)
	}
	if c.MaxConcurrency <= 0 {
		c.MaxConcurrency = defaultMaxConcurrency
	}
	return nil
}
