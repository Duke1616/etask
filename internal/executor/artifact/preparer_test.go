package artifact

// 准备器测试覆盖默认层、具名层和安全挂载。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactv1 "github.com/Duke1616/etask/api/proto/gen/etask/artifact/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestValidateArtifactLayers(t *testing.T) {
	defaultLayer := validRef("")
	namedLayer := validRef("ops_common")
	other := proto.Clone(namedLayer).(*artifactv1.ArtifactRef)
	other.ReleaseId = 3
	testCases := []struct {
		name      string
		refs      []*artifactv1.ArtifactRef
		wantError string
	}{
		{name: "允许默认层与多个具名层", refs: []*artifactv1.ArtifactRef{defaultLayer, namedLayer}},
		{name: "拒绝空引用", refs: []*artifactv1.ArtifactRef{nil}, wantError: "空制品引用"},
		{name: "拒绝重复默认层", refs: []*artifactv1.ArtifactRef{defaultLayer, defaultLayer}, wantError: "重复的默认制品层"},
		{name: "拒绝重复挂载名称", refs: []*artifactv1.ArtifactRef{namedLayer, other}, wantError: "重复的制品挂载名称"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateArtifactLayers(tc.refs)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMountNamedLayers(t *testing.T) {
	type state struct {
		cache   *artifactCache
		first   string
		second  string
		root    string
		cleanup func()
	}
	testCases := []struct {
		name       string
		before     func(t *testing.T, state *state)
		after      func(t *testing.T, state *state)
		layers     func(state *state) []namedLayer
		wantError  string
		assertions func(t *testing.T, state *state)
	}{
		{
			name: "使用英文名称挂载多个具名制品",
			before: func(t *testing.T, current *state) {
				current.first = createPythonLayer(t, "A = 1")
				current.second = createPythonLayer(t, "B = 2")
			},
			layers: func(current *state) []namedLayer {
				return []namedLayer{{namespace: "alpha", root: current.first}, {namespace: "beta", root: current.second}}
			},
			assertions: func(t *testing.T, current *state) {
				content, err := os.ReadFile(filepath.Join(current.root, "python", "beta", "private", "util.py"))
				require.NoError(t, err)
				require.Equal(t, "B = 2", string(content))
			},
		},
		{
			name:   "拒绝保留命名空间",
			before: func(t *testing.T, current *state) { current.first = createPythonLayer(t, "A = 1") },
			layers: func(current *state) []namedLayer {
				return []namedLayer{{namespace: "etask", root: current.first}}
			},
			wantError: "运行时保留名",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			current := &state{cache: newArtifactCache(Config{Dir: t.TempDir()})}
			if tc.before != nil {
				tc.before(t, current)
			}
			if tc.after != nil {
				defer tc.after(t, current)
			}
			var err error
			current.root, current.cleanup, err = (artifactPreparer{cache: current.cache}).mountNamedLayers(tc.layers(current))
			if current.cleanup != nil {
				defer current.cleanup()
			}
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
			tc.assertions(t, current)
		})
	}
}

func validRef(namespace string) *artifactv1.ArtifactRef {
	return &artifactv1.ArtifactRef{
		ReleaseId: 1, MountName: namespace,
		Digest: strings.Repeat("a", 64), BlobChecksum: strings.Repeat("b", 64), Size: 1,
		Format: supportedArtifactFormat, FormatVersion: supportedArtifactVersion,
	}
}

func createPythonLayer(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	directory := filepath.Join(root, "python", "private")
	require.NoError(t, os.MkdirAll(directory, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "util.py"), []byte(content), 0o440))
	return root
}
