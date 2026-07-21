package ai

import (
	"context"
	"fmt"
)

// UnavailableProvider 在模型尚未配置时提供稳定的错误边界。
type UnavailableProvider struct {
	reason string
}

// NewUnavailableProvider 创建不可用的模型供应商。
func NewUnavailableProvider(reason string) Provider {
	return &UnavailableProvider{reason: reason}
}

func (p *UnavailableProvider) Name() string  { return "unavailable" }
func (p *UnavailableProvider) Model() string { return "" }

func (p *UnavailableProvider) Stream(context.Context, Request) (Stream, error) {
	return nil, p.availabilityError()
}

func (p *UnavailableProvider) availabilityError() error {
	return fmt.Errorf("AI provider is unavailable: %s", p.reason)
}

// EnsureAvailable 检查模型供应商是否已经完成有效配置。
func EnsureAvailable(provider Provider) error {
	if provider == nil {
		return fmt.Errorf("AI provider is unavailable: provider is nil")
	}
	unavailable, ok := provider.(*UnavailableProvider)
	if !ok {
		return nil
	}
	return unavailable.availabilityError()
}
