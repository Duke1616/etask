package artifact

// 本文件将缓存层组合为任务可使用的运行目录。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	artifactv1 "github.com/Duke1616/etask/api/proto/gen/etask/artifact/v1"
	"github.com/Duke1616/etask/sdk/executor"
)

type namedLayer struct {
	namespace string
	root      string
}

// Roots 返回 Handler 可读取的制品目录。
func (p Prepared) Roots() executor.ArtifactRoots {
	return executor.ArtifactRoots{Default: p.DefaultRoot, Dependencies: p.DependenciesRoot}
}

// Close 清理本次准备产生的临时聚合目录。
func (p Prepared) Close() error {
	p.Cleanup()
	return nil
}

type artifactPreparer struct {
	cache  *artifactCache
	client artifactv1.ArtifactServiceClient
}

func (p artifactPreparer) Prepare(ctx context.Context, refs []*artifactv1.ArtifactRef) (Prepared, error) {
	// 下载前先验证默认层唯一性和具名依赖冲突，避免产生无用缓存。
	if err := validateArtifactLayers(refs); err != nil {
		return Prepared{}, err
	}

	var result Prepared
	namedLayers := make([]namedLayer, 0, len(refs))
	// 每个引用先解析为只读缓存层，默认层和依赖层随后分开组合。
	for _, ref := range refs {
		root, err := p.cache.Ensure(ctx, p.client, ref)
		if err != nil {
			return Prepared{}, err
		}
		if ref.GetMountName() == "" {
			result.DefaultRoot = root
		} else {
			namedLayers = append(namedLayers, namedLayer{
				namespace: ref.GetMountName(), root: root,
			})
		}
	}
	// 多个具名依赖只创建一个任务级聚合目录，任务结束时统一删除。
	if len(namedLayers) > 0 {
		root, cleanup, err := p.mountNamedLayers(namedLayers)
		if err != nil {
			return Prepared{}, err
		}
		result.DependenciesRoot = root
		result.cleanup = cleanup
	}
	return result, nil
}

func validateArtifactLayers(refs []*artifactv1.ArtifactRef) error {
	var defaultLayer bool
	namespaces := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if ref == nil {
			return fmt.Errorf("任务包含空制品引用")
		}
		if err := validateArtifactRef(ref); err != nil {
			return err
		}
		if ref.GetMountName() == "" {
			if defaultLayer {
				return fmt.Errorf("任务包含重复的默认制品层")
			}
			defaultLayer = true
			continue
		}
		if err := validateMountName(ref.GetMountName()); err != nil {
			return err
		}
		if _, exists := namespaces[ref.GetMountName()]; exists {
			return fmt.Errorf("任务包含重复的制品挂载名称: %s", ref.GetMountName())
		}
		namespaces[ref.GetMountName()] = struct{}{}
	}
	return nil
}

func (p artifactPreparer) mountNamedLayers(layers []namedLayer) (string, func(), error) {
	tempDir := filepath.Join(p.cache.cfg.Dir, "tmp")
	if err := os.MkdirAll(tempDir, 0o750); err != nil {
		return "", nil, fmt.Errorf("创建制品依赖临时目录失败: %w", err)
	}
	root, err := os.MkdirTemp(tempDir, "dependency-layer-*")
	if err != nil {
		return "", nil, fmt.Errorf("创建制品依赖聚合目录失败: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(root) }
	// Python 依赖单独形成 namespace -> python 目录映射，避免项目名直接污染顶层 import。
	pythonRoot := filepath.Join(root, "python")
	if err = os.MkdirAll(pythonRoot, 0o750); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("创建 Python 制品依赖目录失败: %w", err)
	}
	for _, layer := range layers {
		if err = validateMountName(layer.namespace); err != nil {
			cleanup()
			return "", nil, err
		}
		if err = os.Symlink(layer.root, filepath.Join(root, layer.namespace)); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("挂载制品依赖 %s 失败: %w", layer.namespace, err)
		}
		// 非 Python 制品仍保留普通命名空间挂载，不强制存在 python 子目录。
		pythonDir := filepath.Join(layer.root, "python")
		if info, statErr := os.Stat(pythonDir); statErr == nil && info.IsDir() {
			if err = os.Symlink(pythonDir, filepath.Join(pythonRoot, layer.namespace)); err != nil {
				cleanup()
				return "", nil, fmt.Errorf("挂载制品依赖 %s 的 Python 命名空间失败: %w", layer.namespace, err)
			}
		} else if statErr != nil && !os.IsNotExist(statErr) {
			cleanup()
			return "", nil, fmt.Errorf("检查制品依赖 %s 的 Python 目录失败: %w", layer.namespace, statErr)
		}
	}
	return root, cleanup, nil
}

func validateMountName(name string) error {
	if name == "." || name == ".." || filepath.Base(name) != name || name == "etask" {
		return fmt.Errorf("制品挂载名称非法或使用了运行时保留名: %q", name)
	}
	return nil
}
