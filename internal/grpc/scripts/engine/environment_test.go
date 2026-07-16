package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeEnvironment(t *testing.T) {
	testCases := []struct {
		name      string
		base      []string
		overrides []string
		want      []string
	}{
		{name: "覆盖同名变量", base: []string{"A=old", "B=keep"}, overrides: []string{"A=new"}, want: []string{"B=keep", "A=new"}},
		{name: "追加新变量", base: []string{"A=old"}, overrides: []string{"B=new"}, want: []string{"A=old", "B=new"}},
		{name: "空覆盖保持原值", base: []string{"A=old"}, want: []string{"A=old"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, MergeEnvironment(tc.base, tc.overrides))
		})
	}
}
