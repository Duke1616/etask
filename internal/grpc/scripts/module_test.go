package scripts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRuntime(t *testing.T) {
	testCases := []struct {
		name   string
		config RuntimeConfig
	}{
		{name: "默认配置创建两个处理器"},
		{name: "允许自定义运行目录", config: RuntimeConfig{WorkspaceDir: t.TempDir()}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runtime, err := NewRuntime(tc.config)
			require.NoError(t, err)
			handlers := runtime.Handlers()
			require.Len(t, handlers, 2)
			require.Equal(t, "shell", handlers[0].Name())
			require.Equal(t, "python", handlers[1].Name())
			handlers[0] = nil
			require.NotNil(t, runtime.Handlers()[0], "Handlers 应返回副本")
		})
	}
}
