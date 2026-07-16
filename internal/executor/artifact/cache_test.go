package artifact

// 缓存测试覆盖下载、校验和缓存替换。

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	artifactv1 "github.com/Duke1616/etask/api/proto/gen/etask/artifact/v1"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

func TestArtifactCacheEnsure(t *testing.T) {
	type state struct {
		cache       *artifactCache
		archive     []byte
		ref         *artifactv1.ArtifactRef
		client      artifactv1.ArtifactServiceClient
		closeServer func()
	}
	testCases := []struct {
		name       string
		path       string
		content    string
		before     func(t *testing.T, state *state)
		after      func(t *testing.T, state *state)
		wantError  string
		assertions func(t *testing.T, state *state, root string)
	}{
		{
			name: "替换无效缓存并复用完成层", path: "lib/common.py", content: "VALUE = 1\n",
			before: func(t *testing.T, current *state) {
				target := filepath.Join(current.cache.cfg.Dir, "layers", artifactLayerKey(current.ref))
				require.NoError(t, os.MkdirAll(target, 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(target, ".ready"), []byte("broken"), 0o440))
			},
			assertions: func(t *testing.T, current *state, root string) {
				content, err := os.ReadFile(filepath.Join(root, "lib", "common.py"))
				require.NoError(t, err)
				require.Equal(t, "VALUE = 1\n", string(content))
				current.closeServer()
				cached, err := current.cache.Ensure(t.Context(), current.client, current.ref)
				require.NoError(t, err)
				require.Equal(t, root, cached)
				current.closeServer = nil
			},
		},
		{
			name: "拒绝越界路径", path: "../outside", content: "secret", wantError: "超出缓存目录",
			after: func(t *testing.T, current *state) {
				_, err := os.Stat(filepath.Join(current.cache.cfg.Dir, "outside"))
				require.ErrorIs(t, err, os.ErrNotExist)
			},
		},
		{
			name: "拒绝清单摘要不一致", path: "lib/common.py", content: "VALUE = 1\n", wantError: "清单与制品引用不一致",
			before: func(_ *testing.T, current *state) { current.ref.Digest = strings.Repeat("a", 64) },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			current := &state{cache: newArtifactCache(Config{Dir: t.TempDir()})}
			current.archive, current.ref = buildTestRef(t, tc.path, tc.content)
			current.client, current.closeServer = newArtifactClient(t, current.archive)
			defer func() {
				if current.closeServer != nil {
					current.closeServer()
				}
			}()
			if tc.before != nil {
				tc.before(t, current)
			}
			if tc.after != nil {
				defer tc.after(t, current)
			}
			root, err := current.cache.Ensure(t.Context(), current.client, current.ref)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
			tc.assertions(t, current, root)
		})
	}
}

func TestArtifactCacheEnsureRejectsInvalidDependency(t *testing.T) {
	testCases := []struct {
		name      string
		client    artifactv1.ArtifactServiceClient
		ref       *artifactv1.ArtifactRef
		wantError string
	}{
		{name: "拒绝空引用", client: artifactClientStub{}, wantError: "引用不能为空"},
		{name: "拒绝空客户端", ref: validRef("ops_common"), wantError: "客户端尚未初始化"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newArtifactCache(Config{Dir: t.TempDir()}).Ensure(t.Context(), tc.client, tc.ref)
			require.ErrorContains(t, err, tc.wantError)
		})
	}
}

func TestArtifactCachePrunesOldLayers(t *testing.T) {
	root := t.TempDir()
	cache := newArtifactCache(Config{Dir: root, MaxCacheSize: 9})
	layersDir := filepath.Join(root, "layers")
	oldLayer := filepath.Join(layersDir, "old")
	newLayer := filepath.Join(layersDir, "new")
	for _, layer := range []string{oldLayer, newLayer} {
		require.NoError(t, os.MkdirAll(layer, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(layer, ".ready"), []byte("ready"), 0o440))
		require.NoError(t, os.WriteFile(filepath.Join(layer, "data"), []byte("data"), 0o440))
	}
	oldTime := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(filepath.Join(oldLayer, ".ready"), oldTime, oldTime))

	require.NoError(t, cache.Prune())
	_, err := os.Stat(oldLayer)
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(newLayer)
	require.NoError(t, err)
}

func TestValidateArtifactRef(t *testing.T) {
	valid := &artifactv1.ArtifactRef{
		ReleaseId: 1,
		Digest:    strings.Repeat("a", 64), BlobChecksum: strings.Repeat("b", 64), Size: 1,
		Format: supportedArtifactFormat, FormatVersion: supportedArtifactVersion,
	}
	invalidChecksum := proto.Clone(valid).(*artifactv1.ArtifactRef)
	invalidChecksum.BlobChecksum = "invalid"
	invalidFormat := proto.Clone(valid).(*artifactv1.ArtifactRef)
	invalidFormat.FormatVersion++
	testCases := []struct {
		name      string
		ref       *artifactv1.ArtifactRef
		wantError string
	}{
		{name: "合法引用", ref: valid},
		{name: "非法校验和", ref: invalidChecksum, wantError: "校验和非法"},
		{name: "不支持格式", ref: invalidFormat, wantError: "不支持的制品格式"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateArtifactRef(tc.ref)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestArtifactLayerKeySeparatesDifferentBlobs(t *testing.T) {
	first := &artifactv1.ArtifactRef{Digest: strings.Repeat("a", 64), BlobChecksum: strings.Repeat("b", 64)}
	second := proto.Clone(first).(*artifactv1.ArtifactRef)
	second.BlobChecksum = strings.Repeat("c", 64)

	require.NotEqual(t, artifactLayerKey(first), artifactLayerKey(second))
}

type artifactTestServer struct {
	artifactv1.UnimplementedArtifactServiceServer
	data []byte
}

type artifactClientStub struct {
	artifactv1.ArtifactServiceClient
}

func (s artifactTestServer) DownloadArtifact(_ *artifactv1.DownloadArtifactRequest,
	stream grpc.ServerStreamingServer[artifactv1.ArtifactChunk]) error {
	return stream.Send(&artifactv1.ArtifactChunk{Data: s.data})
}

func newArtifactClient(t *testing.T, data []byte) (artifactv1.ArtifactServiceClient, func()) {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	artifactv1.RegisterArtifactServiceServer(server, artifactTestServer{data: data})
	go func() { _ = server.Serve(listener) }()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	return artifactv1.NewArtifactServiceClient(conn), func() {
		_ = conn.Close()
		server.Stop()
		_ = listener.Close()
	}
}

func buildTestArtifact(t *testing.T, name, content string) ([]byte, string) {
	t.Helper()
	fileSum := sha256.Sum256([]byte(content))
	manifest := cachedArtifactManifest{
		FormatVersion: supportedArtifactVersion,
		Files: []cachedManifestFile{{
			Path: name, Hash: hex.EncodeToString(fileSum[:]), Size: int64(len(content)),
		}},
	}
	identity, err := json.Marshal(manifest)
	require.NoError(t, err)
	digestBytes := sha256.Sum256(identity)
	manifest.Digest = hex.EncodeToString(digestBytes[:])
	manifestData, err := json.Marshal(manifest)
	require.NoError(t, err)

	var buffer bytes.Buffer
	encoder, err := zstd.NewWriter(&buffer)
	require.NoError(t, err)
	w := tar.NewWriter(encoder)
	require.NoError(t, w.WriteHeader(&tar.Header{
		Name: ".etask/manifest.json", Mode: 0o444, Size: int64(len(manifestData)), Typeflag: tar.TypeReg,
	}))
	_, err = w.Write(manifestData)
	require.NoError(t, err)
	require.NoError(t, w.WriteHeader(&tar.Header{Name: name, Mode: 0o444, Size: int64(len(content)), Typeflag: tar.TypeReg}))
	_, err = w.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.NoError(t, encoder.Close())
	return buffer.Bytes(), manifest.Digest
}

func buildTestRef(t *testing.T, name, content string) ([]byte, *artifactv1.ArtifactRef) {
	archive, digest := buildTestArtifact(t, name, content)
	checksum := sha256.Sum256(archive)
	return archive, &artifactv1.ArtifactRef{
		ReleaseId: 1, Digest: digest,
		BlobChecksum: hex.EncodeToString(checksum[:]), Size: int64(len(archive)),
		Format: supportedArtifactFormat, FormatVersion: supportedArtifactVersion,
	}
}
