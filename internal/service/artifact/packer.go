package artifact

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/klauspost/compress/zstd"
)

const (
	artifactFormat        = "tar.zst"
	artifactFormatVersion = int32(1)
)

type manifestFile struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

type artifactManifest struct {
	FormatVersion int32          `json:"formatVersion"`
	Digest        string         `json:"digest,omitempty"`
	Files         []manifestFile `json:"files"`
}

type packedArtifact struct {
	Digest       string
	BlobChecksum string
	Path         string
	Size         int64
}

type packer struct {
	tempDir string
}

func (p packer) Pack(files []domain.ArtifactFile) (packedArtifact, error) {
	if len(files) == 0 {
		return packedArtifact{}, fmt.Errorf("代码资源中没有可发布的文件")
	}
	files = append([]domain.ArtifactFile(nil), files...)
	// 固定文件顺序是生成稳定 manifest 摘要和压缩内容的前提。
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	// 打包前统一校验路径、重复项和源码哈希，拒绝不一致的版本记录。
	manifest := artifactManifest{FormatVersion: artifactFormatVersion, Files: make([]manifestFile, 0, len(files))}
	seen := make(map[string]struct{}, len(files))
	for i := range files {
		clean, err := validateArchivePath(files[i].Path)
		if err != nil {
			return packedArtifact{}, err
		}
		if _, ok := seen[clean]; ok {
			return packedArtifact{}, fmt.Errorf("制品中存在重复路径: %s", clean)
		}
		seen[clean] = struct{}{}
		files[i].Path = clean
		actualHash := sha256.Sum256([]byte(files[i].Code))
		actualHashString := hex.EncodeToString(actualHash[:])
		if files[i].Hash != "" && !strings.EqualFold(files[i].Hash, actualHashString) {
			return packedArtifact{}, fmt.Errorf("制品源文件 %s 的源码校验和与版本记录不一致", clean)
		}
		manifest.Files = append(manifest.Files, manifestFile{
			Path: clean, Hash: actualHashString, Size: int64(len(files[i].Code)),
		})
	}
	// Digest 只覆盖语义清单且计算时不包含自身，因此同一文件树得到同一摘要。
	identityJSON, err := json.Marshal(manifest)
	if err != nil {
		return packedArtifact{}, fmt.Errorf("序列化制品清单失败: %w", err)
	}
	digestBytes := sha256.Sum256(identityJSON)
	manifest.Digest = hex.EncodeToString(digestBytes[:])
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return packedArtifact{}, fmt.Errorf("序列化制品清单失败: %w", err)
	}

	// 临时文件只在完整关闭并同步后交给上传阶段，失败路径统一删除。
	if p.tempDir != "" {
		if err = os.MkdirAll(p.tempDir, 0o750); err != nil {
			return packedArtifact{}, fmt.Errorf("创建制品临时目录失败: %w", err)
		}
	}
	tmp, err := os.CreateTemp(p.tempDir, "etask-artifact-*.tar.zst")
	if err != nil {
		return packedArtifact{}, fmt.Errorf("创建制品临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	success := false
	defer func() {
		tmp.Close()
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// 压缩字节同时写入文件和哈希器，BlobChecksum 校验实际传输对象。
	blobHash := sha256.New()
	zstdWriter, err := zstd.NewWriter(io.MultiWriter(tmp, blobHash), zstd.WithEncoderConcurrency(1))
	if err != nil {
		return packedArtifact{}, fmt.Errorf("创建 zstd 编码器失败: %w", err)
	}
	tarWriter := tar.NewWriter(zstdWriter)
	// 清零时间和用户字段，避免机器环境让同一内容产生不同压缩包。
	writeFile := func(name string, content []byte) error {
		header := &tar.Header{
			Name: name, Mode: 0o444, Size: int64(len(content)), Typeflag: tar.TypeReg,
			ModTime: time.Unix(0, 0), AccessTime: time.Unix(0, 0), ChangeTime: time.Unix(0, 0),
			Uid: 0, Gid: 0, Uname: "", Gname: "", Format: tar.FormatPAX,
		}
		if writeErr := tarWriter.WriteHeader(header); writeErr != nil {
			return writeErr
		}
		_, writeErr := tarWriter.Write(content)
		return writeErr
	}
	// manifest 固定为首项，读取端无需扫描整个压缩包即可完成身份校验。
	if err = writeFile(".etask/manifest.json", manifestJSON); err == nil {
		for _, file := range files {
			if err = writeFile(file.Path, []byte(file.Code)); err != nil {
				break
			}
		}
	}
	if err != nil {
		tarWriter.Close()
		zstdWriter.Close()
		return packedArtifact{}, fmt.Errorf("写入制品失败: %w", err)
	}
	if err = tarWriter.Close(); err != nil {
		zstdWriter.Close()
		return packedArtifact{}, fmt.Errorf("关闭 tar 制品写入器失败: %w", err)
	}
	if err = zstdWriter.Close(); err != nil {
		return packedArtifact{}, fmt.Errorf("关闭 zstd 制品写入器失败: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		return packedArtifact{}, fmt.Errorf("同步制品临时文件失败: %w", err)
	}
	stat, err := tmp.Stat()
	if err != nil {
		return packedArtifact{}, fmt.Errorf("读取制品临时文件信息失败: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return packedArtifact{}, fmt.Errorf("关闭制品临时文件失败: %w", err)
	}
	success = true
	return packedArtifact{
		Digest: manifest.Digest, BlobChecksum: hex.EncodeToString(blobHash.Sum(nil)),
		Path: tmpPath, Size: stat.Size(),
	}, nil
}

func validateArchivePath(name string) (string, error) {
	if name == "" || name != strings.TrimSpace(name) || strings.HasPrefix(name, "/") ||
		strings.ContainsAny(name, "\\\x00") {
		return "", fmt.Errorf("非法的制品路径: %q", name)
	}
	clean := path.Clean(name)
	if clean != name {
		return "", fmt.Errorf("制品路径必须是规范相对路径: %q", name)
	}
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("制品路径超出根目录: %q", name)
	}
	if clean == ".etask" || strings.HasPrefix(clean, ".etask/") {
		return "", fmt.Errorf("制品路径使用了保留目录: %q", name)
	}
	return clean, nil
}
