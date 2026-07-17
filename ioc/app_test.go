package ioc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeModes(t *testing.T) {
	testCases := []struct {
		name    string
		modes   []string
		want    map[string]bool
		wantErr string
	}{
		{
			name: "all 展开全部模式", modes: []string{ModeAll},
			want: map[string]bool{ModeScheduler: true, ModeAgent: true, ModeExecutor: true},
		},
		{
			name: "模式去重并规范大小写", modes: []string{" Executor ", "executor", "AGENT"},
			want: map[string]bool{ModeExecutor: true, ModeAgent: true},
		},
		{name: "拒绝未知模式", modes: []string{"unknown"}, wantErr: "不支持的启动模式"},
		{name: "拒绝空模式", modes: nil, wantErr: "至少需要指定一个启动模式"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := normalizeModes(testCase.modes)
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.want, actual)
		})
	}
}
