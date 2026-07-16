package runtime

// 本文件定义 Executor 运行配置及其校验规则。

import (
	"fmt"
	"strings"

	grpcpkg "github.com/Duke1616/etask/pkg/grpc"
	"github.com/Duke1616/etask/pkg/grpc/registry"
)

const (
	// RoleName 是 Executor 注册到服务中心时使用的角色名称。
	RoleName = "executor"
	// ModePush 表示由调度中心主动推送任务。
	ModePush = "PUSH"
	// ModePull 表示 Executor 主动拉取任务。
	ModePull = "PULL"
	// IsolationShared 表示执行节点允许共享使用。
	IsolationShared = "SHARED"
	// IsolationDedicated 表示执行节点由资源池独占使用。
	IsolationDedicated = "DEDICATED"
)

// Config 描述 Executor 节点的运行模式和基础组件配置。
type Config struct {
	Mode           string
	Desc           string
	IsolationLevel string
	Server         grpcpkg.ServerConfig
	Client         grpcpkg.ClientConfig
}

func normalizeConfig(config Config, reg registry.Registry) (Config, error) {
	config.Mode = strings.ToUpper(strings.TrimSpace(config.Mode))
	if config.Mode == "" {
		config.Mode = ModePush
	}
	if config.Mode != ModePush && config.Mode != ModePull {
		return Config{}, fmt.Errorf("Executor 执行模式非法: %s", config.Mode)
	}
	config.IsolationLevel = strings.ToUpper(strings.TrimSpace(config.IsolationLevel))
	if config.IsolationLevel == "" {
		config.IsolationLevel = IsolationShared
	}
	if config.IsolationLevel != IsolationShared && config.IsolationLevel != IsolationDedicated {
		return Config{}, fmt.Errorf("Executor 隔离级别非法: %s", config.IsolationLevel)
	}
	if err := config.Server.Validate(); err != nil {
		return Config{}, fmt.Errorf("Executor 服务配置非法: %w", err)
	}
	if config.Server.ServiceId == "" {
		return Config{}, fmt.Errorf("Executor 服务实例 ID 不能为空")
	}
	if reg == nil {
		return Config{}, fmt.Errorf("Executor 服务注册中心不能为空")
	}
	return config, nil
}
