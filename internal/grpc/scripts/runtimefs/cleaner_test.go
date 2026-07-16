package runtimefs

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPruneDirectory(t *testing.T) {
	type state struct {
		root   string
		marker string
	}
	testCases := []struct {
		name      string
		before    func(t *testing.T, state *state)
		after     func(t *testing.T, state *state)
		root      func(state *state) string
		maxAge    time.Duration
		maxSize   int64
		wantError bool
	}{
		{
			name: "按时间和容量清理目录但保留普通文件",
			before: func(t *testing.T, state *state) {
				state.root = t.TempDir()
				state.marker = filepath.Join(state.root, "README")
				require.NoError(t, os.WriteFile(state.marker, []byte("keep"), 0o600))
				now := time.Now()
				for name, age := range map[string]time.Duration{"old": 48 * time.Hour, "first": 2 * time.Hour, "second": time.Hour} {
					directory := filepath.Join(state.root, name)
					require.NoError(t, os.MkdirAll(directory, 0o750))
					require.NoError(t, os.WriteFile(filepath.Join(directory, "data"), []byte("123456"), 0o600))
					require.NoError(t, os.Chtimes(directory, now.Add(-age), now.Add(-age)))
				}
			},
			after: func(t *testing.T, state *state) {
				require.FileExists(t, state.marker)
				require.NoDirExists(t, filepath.Join(state.root, "old"))
				require.NoDirExists(t, filepath.Join(state.root, "first"))
				require.DirExists(t, filepath.Join(state.root, "second"))
			},
			root: func(state *state) string { return state.root }, maxAge: 24 * time.Hour, maxSize: 6,
		},
		{
			name:   "拒绝文件系统根目录",
			root:   func(_ *state) string { return string(os.PathSeparator) },
			maxAge: time.Hour, maxSize: 1, wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			current := &state{}
			if tc.before != nil {
				tc.before(t, current)
			}
			if tc.after != nil {
				defer tc.after(t, current)
			}
			err := PruneDirectory(tc.root(current), tc.maxAge, tc.maxSize)
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
