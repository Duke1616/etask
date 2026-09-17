package jwt

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ClientInterceptor 客户端 JWT 认证拦截器
type ClientInterceptor struct {
	provider TokenProvider
}

// NewClientInterceptor 基于指定令牌签发策略创建客户端拦截器
func NewClientInterceptor(provider TokenProvider) *ClientInterceptor {
	return &ClientInterceptor{
		provider: provider,
	}
}

// UnaryClientInterceptor 创建一元客户端拦截器
func (c *ClientInterceptor) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(c.withJWTContext(ctx), method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor 创建流式客户端拦截器
func (c *ClientInterceptor) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return streamer(c.withJWTContext(ctx), desc, cc, method, opts...)
	}
}

func (c *ClientInterceptor) withJWTContext(ctx context.Context) context.Context {
	if c.hasJWTInContext(ctx) {
		return ctx
	}
	return c.injectJWTContext(ctx)
}

func (c *ClientInterceptor) hasJWTInContext(ctx context.Context) bool {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return false
	}
	authHeaders := md.Get(AuthorizationKey)
	return len(authHeaders) > 0
}

func (c *ClientInterceptor) injectJWTContext(ctx context.Context) context.Context {
	if c.provider == nil {
		return ctx
	}
	tokenString, err := c.provider.ProvideToken(ctx)
	if err != nil || tokenString == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, AuthorizationKey, BearerPrefix+tokenString)
}

// --- 向后兼容实现 ---

// SignerFunc 动态签发函数兼容别名
type SignerFunc func(ctx context.Context, claims map[string]interface{}) (string, error)

type customSignerTokenProvider struct {
	signer SignerFunc
	opts   providerOptions
}

func (p *customSignerTokenProvider) ProvideToken(ctx context.Context) (string, error) {
	claims := buildBaseClaims(ctx, p.opts)
	return p.signer(ctx, claims)
}

// ClientOption 客户端拦截器兼容选项
type ClientOption func(*clientInterceptorOptions)

type clientInterceptorOptions struct {
	signer SignerFunc
}

// WithClientSigner 兼容老版本的动态签名器选项
func WithClientSigner(signer SignerFunc) ClientOption {
	return func(o *clientInterceptorOptions) {
		o.signer = signer
	}
}

// ClientInterceptorBuilder 保持对原有类名的兼容
type ClientInterceptorBuilder struct {
	*ClientInterceptor
}

// NewClientInterceptorBuilder 保持老构造函数的签名兼容
func NewClientInterceptorBuilder(jwtKey string, opts ...ClientOption) *ClientInterceptorBuilder {
	var options clientInterceptorOptions
	for _, opt := range opts {
		opt(&options)
	}

	var provider TokenProvider
	if options.signer != nil {
		provider = &customSignerTokenProvider{
			signer: options.signer,
			opts:   defaultProviderOptions(),
		}
	} else if jwtKey != "" {
		provider = NewHMACTokenProvider(jwtKey)
	}

	return &ClientInterceptorBuilder{
		ClientInterceptor: NewClientInterceptor(provider),
	}
}
