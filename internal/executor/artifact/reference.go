package artifact

// 本文件校验 Executor 下载所需的传输字段。

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	artifactv1 "github.com/Duke1616/etask/api/proto/gen/etask/artifact/v1"
)

func readyArtifact(dir string, ref *artifactv1.ArtifactRef) bool {
	data, err := os.ReadFile(filepath.Join(dir, ".ready"))
	if err != nil {
		return false
	}
	var cached artifactv1.ArtifactRef
	if json.Unmarshal(data, &cached) != nil {
		return false
	}
	return cached.GetDigest() == ref.GetDigest() &&
		strings.EqualFold(cached.GetBlobChecksum(), ref.GetBlobChecksum()) &&
		cached.GetSize() == ref.GetSize() && cached.GetFormat() == ref.GetFormat() &&
		cached.GetFormatVersion() == ref.GetFormatVersion()
}

func validateArtifactRef(ref *artifactv1.ArtifactRef) error {
	if ref == nil {
		return fmt.Errorf("制品引用不能为空")
	}
	if ref.GetReleaseId() <= 0 {
		return fmt.Errorf("制品发布 ID 非法")
	}
	if err := validateDigest(ref.GetDigest()); err != nil {
		return err
	}
	if err := validateDigest(ref.GetBlobChecksum()); err != nil {
		return fmt.Errorf("制品压缩包校验和非法: %w", err)
	}
	if ref.GetFormat() != supportedArtifactFormat || ref.GetFormatVersion() != supportedArtifactVersion {
		return fmt.Errorf("不支持的制品格式: %s/%d", ref.GetFormat(), ref.GetFormatVersion())
	}
	return nil
}

func validateDigest(digest string) error {
	if len(digest) != 64 {
		return fmt.Errorf("制品摘要长度非法")
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return fmt.Errorf("制品摘要格式非法")
	}
	return nil
}
