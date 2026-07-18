package runtimefs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Duke1616/etask/internal/grpc/scripts/engine"
)

type WorkspaceFactory struct {
	config WorkspaceConfig
}

// NewWorkspaceFactory 创建文件系统工作区工厂。
func NewWorkspaceFactory(config WorkspaceConfig) (*WorkspaceFactory, error) {
	normalized, err := NormalizeWorkspaceConfig(config)
	if err != nil {
		return nil, err
	}
	return &WorkspaceFactory{config: normalized}, nil
}

// Create 创建代码文件并挂载制品目录。
func (f *WorkspaceFactory) Create(options engine.WorkspaceOptions) (engine.Workspace, error) {
	if err := os.MkdirAll(f.config.Dir, 0o750); err != nil {
		return nil, fmt.Errorf("创建任务工作区根目录失败: %w", err)
	}
	root, err := os.MkdirTemp(f.config.Dir, fmt.Sprintf("%d-*", options.ExecutionID))
	if err != nil {
		return nil, fmt.Errorf("创建任务工作区失败: %w", err)
	}
	workspace := &workspace{root: root}
	// prepare 任一步失败都删除本次临时目录，避免留下不可识别的半成品。
	if err = workspace.prepare(options); err != nil {
		_ = workspace.Close()
		return nil, err
	}
	return workspace, nil
}

// Prune 清理过期工作区。
func (f *WorkspaceFactory) Prune() error {
	return PruneDirectory(f.config.Dir, f.config.MaxAge, 0)
}

// Validate 校验工作区目录可写。
func (f *WorkspaceFactory) Validate() error {
	return ValidateDirectory(f.config.Dir)
}

type workspace struct {
	root        string
	codeFile    string
	artifacts   engine.ArtifactRoots
	environment []string
}

func (w *workspace) prepare(options engine.WorkspaceOptions) error {
	codeName := "task" + options.Extension
	// SYSTEM 层固定挂载，并额外映射为 etask Python 命名空间。
	if options.Artifacts.System != "" {
		mounted, err := w.mount("system", options.Artifacts.System)
		if err != nil {
			return err
		}
		w.artifacts.System = mounted
		modules := filepath.Join(w.root, ".etask_modules")
		if err = os.MkdirAll(modules, 0o750); err != nil {
			return fmt.Errorf("创建 Python 制品命名空间失败: %w", err)
		}
		// 显式 python 目录用于纯 Python 制品；混合语言 SYSTEM 制品则将根目录映射到 etask。
		pythonRoot := mounted
		pythonDir := filepath.Join(mounted, "python")
		if info, statErr := os.Stat(pythonDir); statErr == nil && info.IsDir() {
			pythonRoot = pythonDir
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return fmt.Errorf("检查 SYSTEM Python 目录失败: %w", statErr)
		}
		if err = os.Symlink(pythonRoot, filepath.Join(modules, "etask")); err != nil {
			return fmt.Errorf("挂载 SYSTEM Python 命名空间失败: %w", err)
		}
	}
	// 租户制品依赖已经由 Executor 聚合为具名目录，这里只挂载一次。
	if options.Artifacts.Dependencies != "" {
		mounted, err := w.mount("dependencies", options.Artifacts.Dependencies)
		if err != nil {
			return err
		}
		w.artifacts.Dependencies = mounted
	}
	// 脚本和环境最后生成，确保 Handler 看到的是完整且稳定的工作区。
	w.codeFile = filepath.Join(w.root, codeName)
	if err := os.WriteFile(w.codeFile, options.Code, 0o700); err != nil {
		return fmt.Errorf("写入任务脚本失败: %w", err)
	}
	w.environment = buildEnvironment(w.artifacts, w.root)
	return nil
}

func (w *workspace) mount(name, source string) (string, error) {
	absolute, err := filepath.Abs(source)
	if err != nil {
		return "", fmt.Errorf("解析 %s 制品目录失败: %w", name, err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("访问 %s 制品目录失败: %w", name, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s 制品路径不是目录: %s", name, absolute)
	}
	target := filepath.Join(w.root, name)
	if err = os.Symlink(absolute, target); err != nil {
		return "", fmt.Errorf("挂载 %s 制品失败: %w", name, err)
	}
	return target, nil
}

func (w *workspace) Root() string {
	return w.root
}

func (w *workspace) CodeFile() string {
	return w.codeFile
}

func (w *workspace) Environment() []string {
	return w.environment
}

func (w *workspace) WriteFile(name string, content []byte, mode os.FileMode) (string, error) {
	if filepath.Base(name) != name {
		return "", fmt.Errorf("工作区文件名非法: %s", name)
	}
	path := filepath.Join(w.root, name)
	if err := os.WriteFile(path, content, mode); err != nil {
		return "", err
	}
	return path, nil
}

func (w *workspace) Close() error {
	return os.RemoveAll(w.root)
}

var _ engine.WorkspaceFactory = (*WorkspaceFactory)(nil)
var _ engine.Workspace = (*workspace)(nil)
