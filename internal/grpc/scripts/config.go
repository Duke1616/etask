package scripts

import (
	"time"

	"github.com/Duke1616/etask/internal/grpc/scripts/engine"
	"github.com/Duke1616/etask/internal/grpc/scripts/runtimefs"
)

// ArchiveConfig 描述脚本执行现场归档配置。
type ArchiveConfig struct {
	Enabled    *bool         `mapstructure:"enabled" yaml:"enabled"`
	FailedOnly bool          `mapstructure:"failed_only" yaml:"failed_only"`
	Dir        string        `mapstructure:"dir" yaml:"dir"`
	MaxAge     time.Duration `mapstructure:"max_age" yaml:"max_age"`
	MaxSize    int64         `mapstructure:"max_size" yaml:"max_size"`
}

// RuntimeConfig 汇总脚本执行编排、工作区、解释器和归档配置。
type RuntimeConfig struct {
	WorkspaceDir     string        `mapstructure:"workspace_dir" yaml:"workspace_dir"`
	WorkspaceMaxAge  time.Duration `mapstructure:"workspace_max_age" yaml:"workspace_max_age"`
	PythonBinary     string        `mapstructure:"python_binary" yaml:"python_binary"`
	ShellBinary      string        `mapstructure:"shell_binary" yaml:"shell_binary"`
	MaxCodeSize      int64         `mapstructure:"max_code_size" yaml:"max_code_size"`
	MaxArgsSize      int64         `mapstructure:"max_args_size" yaml:"max_args_size"`
	MaxVariablesSize int64         `mapstructure:"max_variables_size" yaml:"max_variables_size"`
	MaxLogLineSize   int           `mapstructure:"max_log_line_size" yaml:"max_log_line_size"`
	MaxResultSize    int64         `mapstructure:"max_result_size" yaml:"max_result_size"`
	Archive          ArchiveConfig `mapstructure:"archive" yaml:"archive"`
}

func (c RuntimeConfig) engineConfig() engine.Config {
	return engine.Config{
		MaxCodeSize:      c.MaxCodeSize,
		MaxArgsSize:      c.MaxArgsSize,
		MaxVariablesSize: c.MaxVariablesSize,
		MaxLogLineSize:   c.MaxLogLineSize,
		MaxResultSize:    c.MaxResultSize,
	}
}

func (c RuntimeConfig) workspaceConfig() runtimefs.WorkspaceConfig {
	return runtimefs.WorkspaceConfig{
		Dir:    c.WorkspaceDir,
		MaxAge: c.WorkspaceMaxAge,
	}
}

func (c RuntimeConfig) archiveConfig() runtimefs.ArchiveConfig {
	enabled := true
	if c.Archive.Enabled != nil {
		enabled = *c.Archive.Enabled
	}
	return runtimefs.ArchiveConfig{
		Enabled:    enabled,
		FailedOnly: c.Archive.FailedOnly,
		Dir:        c.Archive.Dir,
		MaxAge:     c.Archive.MaxAge,
		MaxSize:    c.Archive.MaxSize,
	}
}
