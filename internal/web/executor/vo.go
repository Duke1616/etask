package executor

import "github.com/Duke1616/etask/sdk/executor"

type Executor struct {
	Name     string          `json:"name"`     // 分组的执行器服务名
	Desc     string          `json:"desc"`     // 执行器的总体功能描述
	Mode     string          `json:"mode"`     // 执行器模式
	Handlers []HandlerDetail `json:"handlers"` // 该分组下所有节点共同支持的处理方法
	Nodes    []NodeDetail    `json:"nodes"`    // 该服务名下的所有在线节点
}

type NodeDetail struct {
	ID      string `json:"id"`      // 节点唯一ID
	Address string `json:"address"` // 节点网络地址
}

type HandlerDetail struct {
	Name     string               `json:"name"`
	Desc     string               `json:"desc"`
	Metadata []executor.Parameter `json:"metadata"`
}
