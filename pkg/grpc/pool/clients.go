package pool

import (
	"time"

	grpcpkg "github.com/Duke1616/etask/pkg/grpc"
	"github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/ecodeclub/ekit/syncx"
	"google.golang.org/grpc"
)

type Clients[T any] struct {
	clientMap syncx.Map[string, T]
	registry  registry.Registry
	timeout   time.Duration
	creator   func(conn *grpc.ClientConn) T
}

func NewClients[T any](
	registry registry.Registry,
	timeout time.Duration,
	creator func(conn *grpc.ClientConn) T,
) *Clients[T] {
	return &Clients[T]{
		registry: registry,
		timeout:  timeout,
		creator:  creator,
	}
}

// Get 获取带有自定义负载均衡器的客户端
func (c *Clients[T]) Get(serviceName, authToken string) T {
	// 尝试加载，如果存在，直接返回
	if client, ok := c.clientMap.Load(serviceName); ok {
		return client
	}

	// 使用封装的函数创建连接
	grpcConn, err := grpcpkg.NewClientConn(
		c.registry,
		grpcpkg.WithServiceName(serviceName),
		grpcpkg.WithTimeout(c.timeout),
		grpcpkg.WithClientJWTAuth(authToken),
	)
	if err != nil {
		panic(err)
	}

	newClient := c.creator(grpcConn)
	// 使用 LoadOrStore 原子地存储
	// 如果在当前 goroutine 创建期间，有其他 goroutine 已经存入了值，
	// actual 会是那个已经存在的值，ok 会是 true。
	// 这样可以保证我们总是使用第一个被成功创建和存储的 client。
	actual, _ := c.clientMap.LoadOrStore(serviceName, newClient)
	return actual
}
