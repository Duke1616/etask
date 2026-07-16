package blobstore

import (
	"fmt"
	"strings"
)

type Config struct {
	Driver string      `mapstructure:"driver" yaml:"driver"`
	Local  LocalConfig `mapstructure:"local" yaml:"local"`
	S3     S3Config    `mapstructure:"s3" yaml:"s3"`
}

type LocalConfig struct {
	Root string `mapstructure:"root" yaml:"root"`
}

// New 根据配置创建 Local 或 S3 存储实现。
func New(cfg Config) (Store, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Driver)) {
	case "local":
		return NewLocal(cfg.Local.Root)
	case "s3":
		return NewS3(cfg.S3)
	default:
		return nil, fmt.Errorf("不支持的制品存储类型: %s", cfg.Driver)
	}
}
