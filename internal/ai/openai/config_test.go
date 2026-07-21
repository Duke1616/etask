package openai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigNormalize(t *testing.T) {
	testCases := []struct {
		name      string
		config    Config
		wantError string
		check     func(*testing.T, Config)
	}{
		{
			name: "填充安全默认值", config: Config{Model: " test-model "},
			check: func(t *testing.T, config Config) {
				require.Equal(t, "test-model", config.Model)
				require.Equal(t, defaultEndpoint, config.Endpoint)
				require.Equal(t, defaultTimeout, config.Timeout)
				require.Equal(t, defaultMaxOutputTokens, config.MaxOutputTokens)
				require.Equal(t, defaultMaxConcurrency, config.MaxConcurrency)
			},
		},
		{
			name: "保留显式配置",
			config: Config{
				Endpoint: "http://model.local/v1/", Model: "model", Timeout: time.Minute,
				MaxOutputTokens: 1024, MaxConcurrency: 2, ReasoningEffort: " LOW ",
			},
			check: func(t *testing.T, config Config) {
				require.Equal(t, "http://model.local/v1", config.Endpoint)
				require.Equal(t, time.Minute, config.Timeout)
				require.Equal(t, int64(1024), config.MaxOutputTokens)
				require.Equal(t, 2, config.MaxConcurrency)
				require.Equal(t, "low", config.ReasoningEffort)
			},
		},
		{name: "拒绝空模型", wantError: "model is required"},
		{
			name: "拒绝非法推理强度", config: Config{Model: "model", ReasoningEffort: "extreme"},
			wantError: "unsupported OpenAI reasoning effort",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			config := testCase.config
			err := config.normalize()
			if testCase.wantError != "" {
				require.ErrorContains(t, err, testCase.wantError)
				return
			}
			require.NoError(t, err)
			testCase.check(t, config)
		})
	}
}
