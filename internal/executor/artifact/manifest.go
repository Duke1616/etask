package artifact

// 本文件实现制品清单和文件完整性校验。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	artifactv1 "github.com/Duke1616/etask/api/proto/gen/etask/artifact/v1"
)

const maxManifestSize = int64(4 << 20)

type cachedManifestFile struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

type cachedArtifactManifest struct {
	FormatVersion int32                `json:"formatVersion"`
	Digest        string               `json:"digest,omitempty"`
	Files         []cachedManifestFile `json:"files"`
}

func validateExtractedArtifact(root string, ref *artifactv1.ArtifactRef) error {
	// 先限制 manifest 大小，再读取和解析，避免异常清单占用过多内存。
	manifestPath := filepath.Join(root, ".etask", "manifest.json")
	info, err := os.Stat(manifestPath)
	if err != nil {
		return fmt.Errorf("读取制品清单失败: %w", err)
	}
	if info.Size() <= 0 || info.Size() > maxManifestSize {
		return fmt.Errorf("制品清单大小非法: %d", info.Size())
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("读取制品清单失败: %w", err)
	}
	var manifest cachedArtifactManifest
	if err = json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("解析制品清单失败: %w", err)
	}
	if manifest.FormatVersion != ref.GetFormatVersion() || !strings.EqualFold(manifest.Digest, ref.GetDigest()) {
		return fmt.Errorf("制品清单与制品引用不一致")
	}
	// 复算不含 Digest 字段的语义摘要，确认清单内容未被替换。
	digest := manifest.Digest
	manifest.Digest = ""
	identity, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("计算制品清单摘要失败: %w", err)
	}
	actualDigest := sha256.Sum256(identity)
	if !strings.EqualFold(hex.EncodeToString(actualDigest[:]), digest) {
		return fmt.Errorf("制品清单摘要校验失败")
	}
	// 逐文件验证清单后再反向扫描目录，确保既不缺文件也没有额外文件。
	expected, err := validateManifestFiles(root, manifest.Files)
	if err != nil {
		return err
	}
	return rejectUnexpectedFiles(root, manifestPath, expected)
}

func validateManifestFiles(root string, files []cachedManifestFile) (map[string]struct{}, error) {
	expected := make(map[string]struct{}, len(files))
	for _, file := range files {
		path, err := safeExtractPath(root, file.Path)
		if err != nil || strings.HasPrefix(file.Path, ".etask/") {
			return nil, fmt.Errorf("制品清单包含非法文件路径: %s", file.Path)
		}
		if _, exists := expected[file.Path]; exists {
			return nil, fmt.Errorf("制品清单包含重复文件路径: %s", file.Path)
		}
		if err = verifyExtractedFile(path, file); err != nil {
			return nil, err
		}
		expected[file.Path] = struct{}{}
	}
	return expected, nil
}

func rejectUnexpectedFiles(root, manifestPath string, expected map[string]struct{}) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || path == manifestPath {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if _, exists := expected[relative]; !exists {
			return fmt.Errorf("制品包含清单外文件: %s", relative)
		}
		return nil
	})
}

func verifyExtractedFile(path string, expected cachedManifestFile) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开制品文件 %s 失败: %w", expected.Path, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("读取制品文件 %s 信息失败: %w", expected.Path, err)
	}
	if !info.Mode().IsRegular() || info.Size() != expected.Size {
		return fmt.Errorf("制品文件 %s 大小或类型与清单不一致", expected.Path)
	}
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return fmt.Errorf("校验制品文件 %s 失败: %w", expected.Path, err)
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), expected.Hash) {
		return fmt.Errorf("制品文件 %s 校验和与清单不一致", expected.Path)
	}
	return nil
}
