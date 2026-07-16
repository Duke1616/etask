package artifact

// 本文件实现受限 tar.zst 解压。

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

func (c *artifactCache) extract(source, target string) error {
	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("打开制品压缩包失败: %w", err)
	}
	defer file.Close()
	reader, err := zstd.NewReader(file)
	if err != nil {
		return fmt.Errorf("打开制品 zstd 数据失败: %w", err)
	}
	defer reader.Close()
	archive := tar.NewReader(reader)
	var totalSize int64
	fileCount := 0
	for {
		header, readErr := archive.Next()
		if errors.Is(readErr, io.EOF) {
			return nil
		}
		if readErr != nil {
			return fmt.Errorf("读取制品 tar 数据失败: %w", readErr)
		}
		fileCount++
		// 文件数和累计解压大小在创建文件前校验，限制压缩炸弹影响。
		if fileCount > c.cfg.MaxFileCount {
			return fmt.Errorf("制品文件数量超出限制")
		}
		if header.Size < 0 || header.Size > c.cfg.MaxUnpackedSize-totalSize {
			return fmt.Errorf("制品解压大小超出限制")
		}
		totalSize += header.Size
		// 每个归档路径都重新约束在目标目录内，拒绝绝对路径和目录穿越。
		path, err := safeExtractPath(target, header.Name)
		if err != nil {
			return err
		}
		// 缓存只接受普通文件和目录，符号链接等类型一律拒绝。
		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(path, 0o750); err != nil {
				return fmt.Errorf("创建制品目录失败: %w", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err = extractFile(archive, path, header.Size); err != nil {
				return err
			}
		default:
			return fmt.Errorf("制品包含不支持的文件类型: %s", header.Name)
		}
	}
}

func extractFile(reader io.Reader, path string, size int64) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("创建制品父目录失败: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o440)
	if err != nil {
		return fmt.Errorf("创建制品文件失败: %w", err)
	}
	_, copyErr := io.CopyN(file, reader, size)
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("写入制品文件失败: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("关闭制品文件失败: %w", closeErr)
	}
	return nil
}

func safeExtractPath(root, name string) (string, error) {
	name = filepath.FromSlash(name)
	if name == "" || filepath.IsAbs(name) {
		return "", fmt.Errorf("制品包含非法路径: %q", name)
	}
	clean := filepath.Clean(name)
	if clean == ".ready" {
		return "", fmt.Errorf("制品路径使用了保留文件: %q", name)
	}
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("制品路径超出缓存目录: %q", name)
	}
	resolved := filepath.Join(root, clean)
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("制品路径超出缓存目录: %q", name)
	}
	return resolved, nil
}
