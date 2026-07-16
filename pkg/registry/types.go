package registry

import (
	"context"
	"io"
)

type Registry interface {
	// Register 注册工作实例。
	Register(ctx context.Context, si Instance) error
	// UnRegister 注销工作实例。
	UnRegister(ctx context.Context, si Instance) error

	// ListWorkers 查询指定名称的工作实例。
	ListWorkers(ctx context.Context, name string) ([]Instance, error)
	// Subscribe 订阅指定工作实例的注册变更事件。
	Subscribe(name string) <-chan Event

	io.Closer
}

type Instance struct {
	Name  string `yaml:"name" json:"name"`   // 实例名称
	Desc  string `yaml:"desc" json:"desc"`   // 注解
	Topic string `yaml:"topic" json:"topic"` // 建立 Topic 通道
}

type EventType int

const (
	EventTypeUnknown EventType = iota
	EventTypeAdd
	EventTypeDelete
)

type Event struct {
	Type     EventType
	Key      string
	Instance Instance
}
