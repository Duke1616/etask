package grpc

import (
	"fmt"
	"time"

	"github.com/Duke1616/etask/pkg/grpc/balancer"
	"github.com/Duke1616/etask/pkg/grpc/interceptors"
	"github.com/Duke1616/etask/pkg/grpc/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientConfig gRPC Client 配置
type ClientConfig struct {
	Name      string `mapstructure:"name"`       // 目标服务
	Address   string `mapstructure:"address"`    // 可选：目标服务直连地址，配置后不经过注册中心
	AuthToken string `mapstructure:"auth_token"` // JWT 认证 token (可选)
}

// ClientOption Client 配置选项
type ClientOption func(*clientOptions)

type clientOptions struct {
	authToken   string
	timeout     time.Duration
	serviceName string
	dialOptions []grpc.DialOption // 原生 gRPC DialOption
}

// WithClientJWTAuth 启用 JWT 认证
func WithClientJWTAuth(authToken string) ClientOption {
	return func(o *clientOptions) {
		o.authToken = authToken
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = timeout
	}
}

// WithServiceName 设置服务名(用于 NewClientConnWithBalancer)
func WithServiceName(serviceName string) ClientOption {
	return func(o *clientOptions) {
		o.serviceName = serviceName
	}
}

// WithDialOption 添加原生 gRPC DialOption
// 可以传递任何 grpc.DialOption,如 grpc.WithDefaultServiceConfig、grpc.WithTransportCredentials 等
func WithDialOption(opts ...grpc.DialOption) ClientOption {
	return func(o *clientOptions) {
		o.dialOptions = append(o.dialOptions, opts...)
	}
}

// buildDialOptions 构建通用的 dial options
func buildDialOptions(options *clientOptions) []grpc.DialOption {
	// 1. 通过统一的拦截器门面 (Facade) 管道构建客户端拦截器链
	pipeline := interceptors.NewClientPipeline(options.authToken)
	unaryInterceptors, streamInterceptors := pipeline.Build()

	// 2. 统一打包组装 DialOption 选项 (默认使用非安全通道)
	dialOpts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(unaryInterceptors...),
		grpc.WithChainStreamInterceptor(streamInterceptors...),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// 3. 追加用户自定义的原生 grpc.DialOption
	return append(dialOpts, options.dialOptions...)
}

// NewClientConn 创建 gRPC Client 连接
// 使用自定义 resolver 进行服务发现,支持负载均衡
// 必须通过 WithServiceName 选项指定服务名
func NewClientConn(reg registry.Registry, opts ...ClientOption) (*grpc.ClientConn, error) {
	options := &clientOptions{
		timeout: 10 * time.Second, // 默认 10 秒
	}
	for _, opt := range opts {
		opt(options)
	}

	// 验证必需参数
	if options.serviceName == "" {
		return nil, fmt.Errorf("serviceName is required, use WithServiceName option")
	}

	// 创建 resolver
	rs := NewResolverBuilder(reg, options.timeout)

	// 构建 dial options
	dialOpts := []grpc.DialOption{
		grpc.WithResolvers(rs),
		grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingPolicy":%q}`, balancer.RoutingRoundRobinName)),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  1 * time.Second,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   3 * time.Second,
			},
			MinConnectTimeout: 5 * time.Second,
		}),
	}
	dialOpts = append(dialOpts, buildDialOptions(options)...)

	// 构建 target: scheme:///serviceName
	// 注意: 不要在 target 中包含 prefix,因为 resolver 会从 registry 中查询时自动添加
	target := fmt.Sprintf("%s:///%s", rs.Scheme(), options.serviceName)
	return grpc.NewClient(target, dialOpts...)
}

// NewDirectClientConn 创建不依赖注册中心的直连客户端。
func NewDirectClientConn(address string, opts ...ClientOption) (*grpc.ClientConn, error) {
	if address == "" {
		return nil, fmt.Errorf("address is required")
	}
	options := &clientOptions{timeout: 10 * time.Second}
	for _, opt := range opts {
		opt(options)
	}
	dialOpts := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay: time.Second, Multiplier: 1.6, Jitter: 0.2, MaxDelay: 3 * time.Second,
			},
			MinConnectTimeout: 5 * time.Second,
		}),
	}
	dialOpts = append(dialOpts, buildDialOptions(options)...)
	return grpc.NewClient(address, dialOpts...)
}
