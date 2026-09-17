package jwt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	AuthorizationKey  = "Authorization"
	BearerPrefix      = "Bearer "
	DefaultIssuer     = "ework-runner"
	DefaultExpiration = 24 * time.Hour
)

// Authenticator 服务端身份认证接口
type Authenticator interface {
	Authenticate(ctx context.Context) (context.Context, error)
	UnaryInterceptor() grpc.UnaryServerInterceptor
	StreamInterceptor() grpc.StreamServerInterceptor
}

// InterceptorBuilder 服务端身份认证拦截器构建器
// 采用原生标准库 Keyfunc 自适应支持对称 HMAC 与非对称 RSA 双轨验签，内聚受众断言与黑名单拦截。
type InterceptorBuilder struct {
	key              string
	pubKeyProvider   PublicKeyProvider
	expectedAudience string
	strictAudience   bool
	blacklistChecker func(ctx context.Context, nodeID string) (bool, error)
	issuer           string
	exp              time.Duration
}

// Option 配置选项
type Option func(*InterceptorBuilder)

// WithIssuer 设置签发者
func WithIssuer(issuer string) Option {
	return func(b *InterceptorBuilder) { b.issuer = issuer }
}

// WithExpiration 设置过期时间
func WithExpiration(exp time.Duration) Option {
	return func(b *InterceptorBuilder) { b.exp = exp }
}

// WithPublicKeyProvider 注入 RSA 公钥提供者 (注入后默认对受众启用严格校验)
func WithPublicKeyProvider(provider PublicKeyProvider) Option {
	return func(b *InterceptorBuilder) {
		b.pubKeyProvider = provider
		b.strictAudience = true
	}
}

// WithExpectedAudience 设置期望的目标受众
func WithExpectedAudience(aud string) Option {
	return func(b *InterceptorBuilder) { b.expectedAudience = aud }
}

// WithStrictAudience 显式指定是否严格要求令牌携带受众
func WithStrictAudience(strict bool) Option {
	return func(b *InterceptorBuilder) { b.strictAudience = strict }
}

// WithBlacklistChecker 注入节点黑名单检查器
func WithBlacklistChecker(checker func(ctx context.Context, nodeID string) (bool, error)) Option {
	return func(b *InterceptorBuilder) { b.blacklistChecker = checker }
}

// NewJwtAuth 创建服务端 JWT 认证拦截器
func NewJwtAuth(key string, opts ...Option) *InterceptorBuilder {
	b := &InterceptorBuilder{
		key:    key,
		issuer: DefaultIssuer,
		exp:    DefaultExpiration,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// NewAuthenticator 创建统一服务端认证器门面
func NewAuthenticator(opts ...Option) Authenticator {
	return NewJwtAuth("", opts...)
}

// Authenticate 从 incoming context 提取并完成令牌鉴权，输出注入 Claims 的 Context
func (b *InterceptorBuilder) Authenticate(ctx context.Context) (context.Context, error) {
	if b == nil || (b.key == "" && b.pubKeyProvider == nil) {
		return ctx, nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "请求缺少元数据 metadata")
	}
	authHeaders := md.Get(AuthorizationKey)
	if len(authHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "请求缺少授权令牌 Authorization")
	}

	rawToken := strings.TrimPrefix(authHeaders[0], BearerPrefix)
	claims, err := b.verifyToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	return Set(ctx, claims), nil
}

// verifyToken 利用原生 Keyfunc 自适应解析并校验令牌及业务约束
func (b *InterceptorBuilder) verifyToken(ctx context.Context, rawToken string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(rawToken, claims, func(t *jwt.Token) (interface{}, error) {
		switch t.Method.(type) {
		case *jwt.SigningMethodHMAC:
			if b.key != "" {
				return []byte(b.key), nil
			}
		case *jwt.SigningMethodRSA:
			if b.pubKeyProvider != nil {
				return b.pubKeyProvider()
			}
		}
		return nil, fmt.Errorf("不支持或未配置的签名算法: %v", t.Header["alg"])
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, status.Error(codes.Unauthenticated, "授权令牌已过期")
		}
		return nil, status.Error(codes.Unauthenticated, "无效的授权令牌: "+err.Error())
	}
	if !token.Valid {
		return nil, status.Error(codes.Unauthenticated, "授权令牌验证失败")
	}

	// 目标受众断言：防止跨节点滥用
	if b.expectedAudience != "" {
		aud, _ := claims["aud"].(string)
		if aud == "" && b.strictAudience {
			return nil, status.Error(codes.PermissionDenied, "缺少目标受众声明 aud")
		}
		if aud != "" && aud != b.expectedAudience {
			return nil, status.Errorf(codes.PermissionDenied, "目标受众不匹配: 期望 %s, 实际为 %s", b.expectedAudience, aud)
		}
	}

	// 黑名单断言：支持节点秒级拉黑
	if b.blacklistChecker != nil && b.expectedAudience != "" {
		blocked, err := b.blacklistChecker(ctx, b.expectedAudience)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "检查节点黑名单状态失败: %v", err)
		}
		if blocked {
			return nil, status.Errorf(codes.PermissionDenied, "节点 %s 已被加入黑名单，禁止执行任务", b.expectedAudience)
		}
	}

	return claims, nil
}

// UnaryInterceptor 创建一元服务端拦截器
func (b *InterceptorBuilder) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		authCtx, err := b.Authenticate(ctx)
		if err != nil {
			return nil, err
		}
		return handler(authCtx, req)
	}
}

// StreamInterceptor 创建流式服务端拦截器
func (b *InterceptorBuilder) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		authCtx, err := b.Authenticate(stream.Context())
		if err != nil {
			return err
		}
		return handler(srv, &authenticatedServerStream{ServerStream: stream, ctx: authCtx})
	}
}

// JwtAuthInterceptor 保持原有方法名兼容
func (b *InterceptorBuilder) JwtAuthInterceptor() grpc.UnaryServerInterceptor {
	return b.UnaryInterceptor()
}

// JwtAuthStreamInterceptor 保持原有方法名兼容
func (b *InterceptorBuilder) JwtAuthStreamInterceptor() grpc.StreamServerInterceptor {
	return b.StreamInterceptor()
}

// Decode 保持原有解析 Claims 工具方法兼容
func (b *InterceptorBuilder) Decode(tokenString string) (jwt.MapClaims, error) {
	if b == nil {
		return nil, errors.New("校验器未就绪")
	}
	rawToken := strings.TrimPrefix(tokenString, BearerPrefix)
	return b.verifyToken(context.Background(), rawToken)
}

// Encode 保持原有签发 HMAC 令牌工具方法兼容
func (b *InterceptorBuilder) Encode(customClaims jwt.MapClaims) (string, error) {
	claims := jwt.MapClaims{
		"iat": time.Now().Unix(),
		"iss": b.issuer,
	}
	for k, v := range customClaims {
		claims[k] = v
	}
	if _, ok := claims["exp"]; !ok {
		claims["exp"] = time.Now().Add(b.exp).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(b.key))
}

type authenticatedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedServerStream) Context() context.Context { return s.ctx }
