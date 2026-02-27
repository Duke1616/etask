package grpc

import (
	"fmt"
	"time"

	"github.com/Duke1616/ework-runner/pkg/grpc/balancer"
	jwtinterceptor "github.com/Duke1616/ework-runner/pkg/grpc/interceptors/jwt"
	"github.com/Duke1616/ework-runner/pkg/grpc/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientConfig gRPC Client 配置
type ClientConfig struct {
	Name      string `mapstructure:"name"`       // 目标服务
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
	var dialOpts []grpc.DialOption

	// 添加 JWT 认证拦截器
	if options.authToken != "" {
		jwtInterceptor := jwtinterceptor.NewClientInterceptorBuilder(options.authToken)
		dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(jwtInterceptor.UnaryClientInterceptor()))
	}

	// 默认使用 insecure credentials
	// 如果需要 TLS,用户可以通过 WithDialOption 传递 grpc.WithTransportCredentials
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// 添加自定义的 DialOption
	dialOpts = append(dialOpts, options.dialOptions...)

	return dialOpts
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
