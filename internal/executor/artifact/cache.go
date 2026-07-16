package artifact

// 缓存实现属于 etask Executor 基础设施，不属于通用 SDK。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	artifactv1 "github.com/Duke1616/etask/api/proto/gen/etask/artifact/v1"
	"golang.org/x/sync/singleflight"
)

const (
	defaultMaxArtifactSize      = int64(512 << 20)
	defaultMaxUnpackedSize      = int64(2 << 30)
	defaultMaxArtifactFileCount = 10000
	defaultMaxArtifactCacheSize = int64(10 << 30)
	artifactDownloadTimeout     = 10 * time.Minute
	supportedArtifactFormat     = "tar.zst"
	supportedArtifactVersion    = int32(1)
)

// Config 描述制品缓存目录和资源限制。
type Config struct {
	Dir             string `mapstructure:"dir" yaml:"dir"`
	MaxDownloadSize int64  `mapstructure:"max_download_size" yaml:"max_download_size"`
	MaxUnpackedSize int64  `mapstructure:"max_unpacked_size" yaml:"max_unpacked_size"`
	MaxFileCount    int    `mapstructure:"max_file_count" yaml:"max_file_count"`
	MaxCacheSize    int64  `mapstructure:"max_cache_size" yaml:"max_cache_size"`
}

type artifactCache struct {
	cfg   Config
	group singleflight.Group
}

func newArtifactCache(cfg Config) *artifactCache {
	if strings.TrimSpace(cfg.Dir) == "" {
		cfg.Dir = filepath.Join(os.TempDir(), "etask", "artifact-cache")
	}
	if cfg.MaxDownloadSize <= 0 {
		cfg.MaxDownloadSize = defaultMaxArtifactSize
	}
	if cfg.MaxUnpackedSize <= 0 {
		cfg.MaxUnpackedSize = defaultMaxUnpackedSize
	}
	if cfg.MaxFileCount <= 0 {
		cfg.MaxFileCount = defaultMaxArtifactFileCount
	}
	if cfg.MaxCacheSize <= 0 {
		cfg.MaxCacheSize = defaultMaxArtifactCacheSize
	}
	return &artifactCache{cfg: cfg}
}

func (c *artifactCache) Prune() error {
	return pruneCache(c.cfg.Dir, c.cfg.MaxCacheSize)
}

func (c *artifactCache) Ensure(ctx context.Context, client artifactv1.ArtifactServiceClient,
	ref *artifactv1.ArtifactRef) (string, error) {
	if client == nil {
		return "", fmt.Errorf("制品下载客户端尚未初始化")
	}
	if err := validateArtifactRef(ref); err != nil {
		return "", err
	}
	if ref.GetSize() <= 0 || ref.GetSize() > c.cfg.MaxDownloadSize {
		return "", fmt.Errorf("制品大小超出限制: %d", ref.GetSize())
	}

	// 相同内容的并发请求共享一次下载；每个等待者仍可独立响应自身取消。
	cacheKey := artifactLayerKey(ref)
	result := c.group.DoChan(cacheKey, func() (any, error) {
		// 共享下载不能被任一调用方取消，但必须受统一超时约束。
		workCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), artifactDownloadTimeout)
		defer cancel()
		return c.ensureOnce(workCtx, client, ref)
	})
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case value := <-result:
		if value.Err != nil {
			return "", value.Err
		}
		return value.Val.(string), nil
	}
}

func (c *artifactCache) ensureOnce(ctx context.Context, client artifactv1.ArtifactServiceClient,
	ref *artifactv1.ArtifactRef) (string, error) {
	layersDir := filepath.Join(c.cfg.Dir, "layers")
	tempDir := filepath.Join(c.cfg.Dir, "tmp")
	targetDir := filepath.Join(layersDir, artifactLayerKey(ref))
	// .ready 内容与引用完全匹配才视为缓存命中，并刷新最近使用时间。
	if readyArtifact(targetDir, ref) {
		touchArtifact(targetDir)
		return targetDir, nil
	}
	if err := os.MkdirAll(tempDir, 0o750); err != nil {
		return "", fmt.Errorf("创建制品缓存目录失败: %w", err)
	}
	if err := os.MkdirAll(layersDir, 0o750); err != nil {
		return "", fmt.Errorf("创建制品缓存目录失败: %w", err)
	}

	// 下载写入独立 part 文件，任何失败都由 defer 清理。
	part, err := os.CreateTemp(tempDir, ref.GetDigest()+"-*.part")
	if err != nil {
		return "", fmt.Errorf("创建制品临时文件失败: %w", err)
	}
	partPath := part.Name()
	defer func() {
		_ = part.Close()
		_ = os.Remove(partPath)
	}()
	if err = c.download(ctx, client, ref, part); err != nil {
		return "", err
	}
	if err = part.Close(); err != nil {
		return "", fmt.Errorf("关闭制品临时文件失败: %w", err)
	}

	// 解压、manifest 和逐文件哈希校验全部在临时目录完成。
	extractDir, err := os.MkdirTemp(tempDir, ref.GetDigest()+"-*.extract")
	if err != nil {
		return "", fmt.Errorf("创建制品解压目录失败: %w", err)
	}
	defer os.RemoveAll(extractDir)
	if err := c.extract(partPath, extractDir); err != nil {
		return "", err
	}
	if err := validateExtractedArtifact(extractDir, ref); err != nil {
		return "", err
	}
	// .ready 是完整缓存层的提交标记，必须在所有校验通过后写入。
	readyBytes, err := json.Marshal(ref)
	if err != nil {
		return "", fmt.Errorf("序列化制品缓存标记失败: %w", err)
	}
	if err = os.WriteFile(filepath.Join(extractDir, ".ready"), readyBytes, 0o440); err != nil {
		return "", fmt.Errorf("写入制品缓存标记失败: %w", err)
	}
	// Rename 将验证完成的目录原子提交；并发提交时优先复用已就绪结果。
	if err = os.Rename(extractDir, targetDir); err != nil {
		if readyArtifact(targetDir, ref) {
			return targetDir, nil
		}
		if removeErr := os.RemoveAll(targetDir); removeErr != nil {
			return "", fmt.Errorf("清理无效制品缓存失败: %w", removeErr)
		}
		if err = os.Rename(extractDir, targetDir); err != nil {
			if readyArtifact(targetDir, ref) {
				return targetDir, nil
			}
			return "", fmt.Errorf("提交制品缓存失败: %w", err)
		}
	}
	return targetDir, nil
}

func artifactLayerKey(ref *artifactv1.ArtifactRef) string {
	return ref.GetDigest() + "-" + ref.GetBlobChecksum()
}
