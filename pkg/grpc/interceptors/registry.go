package interceptors

import (
	"github.com/Duke1616/etask/pkg/grpc/interceptors/bizid"
	jwtinterceptor "github.com/Duke1616/etask/pkg/grpc/interceptors/jwt"
	"github.com/Duke1616/etask/pkg/grpc/interceptors/tenant"
	"google.golang.org/grpc"
)

// ServerPipeline 服务端拦截器管道拼装门面
type ServerPipeline struct {
	auth jwtinterceptor.Authenticator
}

// ServerPipelineOption 服务端管道选项
type ServerPipelineOption func(*ServerPipeline)

// WithAuthenticator 为管道注入统一的 Authenticator 认证器
func WithAuthenticator(auth jwtinterceptor.Authenticator) ServerPipelineOption {
	return func(pipe *ServerPipeline) {
		pipe.auth = auth
	}
}

// NewServerPipeline 创建服务端拦截器管道 (保持对 authToken 入参的向下兼容)
func NewServerPipeline(authToken string, opts ...ServerPipelineOption) *ServerPipeline {
	p := &ServerPipeline{}
	for _, opt := range opts {
		opt(p)
	}
	// 若未显式传入 Authenticator 且配置了静态 authToken，自动回退构建基础 HMAC 认证器
	if p.auth == nil && authToken != "" {
		p.auth = jwtinterceptor.NewJwtAuth(authToken)
	}
	return p
}

// Build 拼装并输出有序的服务端拦截器链 (一元链 与 流式链)
func (p *ServerPipeline) Build() ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		bizid.UnaryServerInterceptor(),
	}
	streamInterceptors := []grpc.StreamServerInterceptor{
		bizid.StreamServerInterceptor(),
	}

	var unaryAuth grpc.UnaryServerInterceptor
	var streamAuth grpc.StreamServerInterceptor
	if p.auth != nil {
		unaryAuth = p.auth.UnaryInterceptor()
		streamAuth = p.auth.StreamInterceptor()
	}

	// 仅当认证器产出有效的拦截器时挂载 JWT 防线，否则在内网环境安全降级为明文租户透传
	if unaryAuth != nil && streamAuth != nil {
		unaryInterceptors = append(unaryInterceptors, unaryAuth)
		streamInterceptors = append(streamInterceptors, streamAuth)
	} else {
		unaryInterceptors = append(unaryInterceptors, tenant.UnaryServerInterceptor())
		streamInterceptors = append(streamInterceptors, tenant.StreamServerInterceptor())
	}

	return unaryInterceptors, streamInterceptors
}

// ClientPipeline 客户端拦截器管道拼装门面
type ClientPipeline struct {
	tokenProvider jwtinterceptor.TokenProvider
}

// ClientPipelineOption 客户端管道选项
type ClientPipelineOption func(*ClientPipeline)

// WithTokenProvider 为管道注入客户端令牌提供策略
func WithTokenProvider(provider jwtinterceptor.TokenProvider) ClientPipelineOption {
	return func(p *ClientPipeline) {
		p.tokenProvider = provider
	}
}

// NewClientPipeline 创建客户端拦截器管道 (保持对 authToken 入参的向下兼容)
func NewClientPipeline(authToken string, opts ...ClientPipelineOption) *ClientPipeline {
	p := &ClientPipeline{}
	for _, opt := range opts {
		opt(p)
	}
	// 若未显式传入 TokenProvider 且配置了静态 authToken，自动回退构建基础 HMAC 策略
	if p.tokenProvider == nil && authToken != "" {
		p.tokenProvider = jwtinterceptor.NewHMACTokenProvider(authToken)
	}
	return p
}

// Build 拼装并输出有序的客户端拦截器链 (一元链 与 流式链)
func (p *ClientPipeline) Build() ([]grpc.UnaryClientInterceptor, []grpc.StreamClientInterceptor) {
	unaryInterceptors := []grpc.UnaryClientInterceptor{
		bizid.UnaryClientInterceptor(),
	}
	streamInterceptors := []grpc.StreamClientInterceptor{
		bizid.StreamClientInterceptor(),
	}

	// 启用安全认证或提供者则通过自签发 JWT 承载身份，否则在内网环境下通过 Metadata 头部透传
	if p.tokenProvider != nil {
		jwtInterceptor := jwtinterceptor.NewClientInterceptor(p.tokenProvider)
		unaryInterceptors = append(unaryInterceptors, jwtInterceptor.UnaryClientInterceptor())
		streamInterceptors = append(streamInterceptors, jwtInterceptor.StreamClientInterceptor())
	} else {
		unaryInterceptors = append(unaryInterceptors, tenant.UnaryClientInterceptor())
		streamInterceptors = append(streamInterceptors, tenant.StreamClientInterceptor())
	}

	return unaryInterceptors, streamInterceptors
}
