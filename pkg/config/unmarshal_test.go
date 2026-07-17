package config

import (
	"bytes"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"10GB", 10737418240, false},
		{"512MB", 536870912, false},
		{"4.5MB", 4718592, false},
		{"1024B", 1024, false},
		{"1.2TB", 1319413953331, false}, // 1.2 * 1024^4 = 1319413953331.2 -> 1319413953331
		{" 100 KB ", 102400, false},
		{"100", 100, false},
		{"", 0, false},
		{"invalid", 0, true},
		{"100invalid", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			val, err := ParseBytes(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, val)
			}
		})
	}
}

type dummyConfig struct {
	MaxDownloadSize int64         `mapstructure:"max_download_size"`
	MaxFileCount    int           `mapstructure:"max_file_count"`
	Timeout         time.Duration `mapstructure:"timeout"`
}

func TestUnmarshalKey(t *testing.T) {
	yamlConfig := []byte(`
test:
  max_download_size: 5GB
  max_file_count: 50MB
  timeout: 2h
`)

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBuffer(yamlConfig))
	require.NoError(t, err)

	// 使用全局 viper 辅助，但由于 UnmarshalKey 内部直接调用包级别的 viper
	// 为了使包级别的 UnmarshalKey 测试通过，我们需要将配置写入全局 viper
	viper.SetConfigType("yaml")
	err = viper.ReadConfig(bytes.NewBuffer(yamlConfig))
	require.NoError(t, err)

	var cfg dummyConfig
	err = UnmarshalKey("test", &cfg)
	require.NoError(t, err)

	assert.Equal(t, int64(5368709120), cfg.MaxDownloadSize) // 5GB
	assert.Equal(t, int(52428800), cfg.MaxFileCount)       // 50MB
	assert.Equal(t, 2*time.Hour, cfg.Timeout)
}
