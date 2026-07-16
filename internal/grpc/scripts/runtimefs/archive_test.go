package runtimefs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Duke1616/etask/internal/grpc/scripts/engine"
	"github.com/stretchr/testify/require"
)

func TestArchiverArchive(t *testing.T) {
	type state struct {
		root     string
		codeFile string
		archiver *Archiver
	}
	testCases := []struct {
		name       string
		config     func(state *state) ArchiveConfig
		failed     bool
		before     func(t *testing.T, state *state)
		after      func(t *testing.T, state *state)
		wantCount  int
		assertions func(t *testing.T, state *state, directory string)
	}{
		{name: "关闭归档不写文件", config: func(state *state) ArchiveConfig { return ArchiveConfig{Dir: state.root} }, wantCount: 0},
		{name: "仅失败策略忽略成功任务", config: func(state *state) ArchiveConfig {
			return ArchiveConfig{Enabled: true, FailedOnly: true, Dir: state.root}
		}, wantCount: 0},
		{
			name: "归档脚本并脱敏变量", failed: true,
			config: func(state *state) ArchiveConfig {
				return ArchiveConfig{Enabled: true, FailedOnly: true, Dir: state.root}
			},
			wantCount: 1,
			assertions: func(t *testing.T, _ *state, directory string) {
				require.FileExists(t, filepath.Join(directory, "scripts.sh"))
				variables, err := os.ReadFile(filepath.Join(directory, "scripts.vars.json"))
				require.NoError(t, err)
				require.JSONEq(t, `[{"key":"public","value":"visible","secret":false},{"key":"token","value":"","secret":true}]`, string(variables))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			current := &state{root: t.TempDir()}
			current.codeFile = filepath.Join(t.TempDir(), "task.sh")
			require.NoError(t, os.WriteFile(current.codeFile, []byte("echo ok\n"), 0o700))
			if tc.before != nil {
				tc.before(t, current)
			}
			if tc.after != nil {
				defer tc.after(t, current)
			}
			var err error
			current.archiver, err = NewArchiver(tc.config(current))
			require.NoError(t, err)
			err = current.archiver.Archive(engine.ArchiveRecord{
				ExecutionID: 88, CodeFile: current.codeFile, Args: `{}`,
				Variables: `[{"key":"public","value":"visible"},{"key":"token","value":"secret","secret":true}]`,
				Failed:    tc.failed,
			})
			require.NoError(t, err)
			entries, err := os.ReadDir(current.root)
			require.NoError(t, err)
			require.Len(t, entries, tc.wantCount)
			if tc.wantCount > 0 {
				require.True(t, strings.HasPrefix(entries[0].Name(), "88_"))
				if tc.assertions != nil {
					tc.assertions(t, current, filepath.Join(current.root, entries[0].Name()))
				}
			}
		})
	}
}
