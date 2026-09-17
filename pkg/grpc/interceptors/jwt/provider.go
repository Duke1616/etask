package jwt

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Duke1616/eiam/pkg/ctxutil"
	"github.com/golang-jwt/jwt/v4"
)

// TokenProvider 客户端令牌提供策略接口
type TokenProvider interface {
	// ProvideToken 根据上下文环境签发或生成合法令牌
	ProvideToken(ctx context.Context) (string, error)
}

// ProviderOption 令牌提供者可选配置
type ProviderOption func(*providerOptions)

type providerOptions struct {
	issuer     string
	expiration time.Duration
}

func defaultProviderOptions() providerOptions {
	return providerOptions{
		issuer:     DefaultIssuer,
		expiration: DefaultExpiration,
	}
}

// WithProviderIssuer 自定义签发者
func WithProviderIssuer(issuer string) ProviderOption {
	return func(o *providerOptions) {
		o.issuer = issuer
	}
}

// WithProviderExpiration 自定义令牌有效期
func WithProviderExpiration(exp time.Duration) ProviderOption {
	return func(o *providerOptions) {
		o.expiration = exp
	}
}

// buildBaseClaims 构建标准基础 Claims (自动注入租户、用户以及受众上下文)
func buildBaseClaims(ctx context.Context, opts providerOptions) jwt.MapClaims {
	claims := jwt.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(opts.expiration).Unix(),
		"iss": opts.issuer,
	}
	if tid := ctxutil.GetTenantID(ctx); tid > 0 {
		claims["tenant_id"] = tid
	}
	if uid := ctxutil.GetUserID(ctx); uid > 0 {
		claims["user_id"] = uid
	}
	if aud := GetAudience(ctx); aud != "" {
		claims["aud"] = aud
	}
	return claims
}

// --- HMAC 对称秘钥提供策略 ---

type hmacTokenProvider struct {
	key  string
	opts providerOptions
}

// NewHMACTokenProvider 实例化基于 HMAC 对称密钥的签发策略
func NewHMACTokenProvider(key string, opts ...ProviderOption) TokenProvider {
	options := defaultProviderOptions()
	for _, opt := range opts {
		opt(&options)
	}
	return &hmacTokenProvider{
		key:  key,
		opts: options,
	}
}

func (p *hmacTokenProvider) ProvideToken(ctx context.Context) (string, error) {
	if p.key == "" {
		return "", errors.New("HMAC 密钥不能为空")
	}
	claims := buildBaseClaims(ctx, p.opts)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(p.key))
}

// --- RSA 非对称密钥提供策略 ---

type cachedToken struct {
	token     string
	expiresAt time.Time
}

type rsaTokenProvider struct {
	km    IClusterKeyManager
	opts  providerOptions
	mu    sync.RWMutex
	cache map[string]cachedToken
}

// NewRSATokenProvider 实例化基于集群 RSA 密钥管理器的签发策略
func NewRSATokenProvider(km IClusterKeyManager, opts ...ProviderOption) TokenProvider {
	options := defaultProviderOptions()
	// RSA 任务下发令牌默认建议短时有效 (如 1 分钟)
	options.expiration = time.Minute
	for _, opt := range opts {
		opt(&options)
	}
	return &rsaTokenProvider{
		km:    km,
		opts:  options,
		cache: make(map[string]cachedToken),
	}
}

func (p *rsaTokenProvider) ProvideToken(ctx context.Context) (string, error) {
	if p.km == nil {
		return "", errors.New("集群密钥管理器未就绪")
	}

	tid := ctxutil.GetTenantID(ctx)
	uid := ctxutil.GetUserID(ctx)
	aud := GetAudience(ctx)
	cacheKey := fmt.Sprintf("%d:%d:%s", tid, uid, aud)

	// 1. 尝试从本地短时缓存中复用未过期的合法令牌 (避免每次 RPC 重新做耗时的 RSA 模幂运算)
	p.mu.RLock()
	if entry, ok := p.cache[cacheKey]; ok {
		// 只要离过期还有至少 15 秒，即可安全复用
		if time.Until(entry.expiresAt) > 15*time.Second {
			p.mu.RUnlock()
			return entry.token, nil
		}
	}
	p.mu.RUnlock()

	claims := buildBaseClaims(ctx, p.opts)
	tokenStr, err := p.km.SignJWT(claims)
	if err != nil {
		return "", fmt.Errorf("使用 RSA 集群私钥签发令牌失败: %w", err)
	}

	// 2. 写入短时复用缓存，自动控制缓存字典大小
	p.mu.Lock()
	if len(p.cache) > 1024 {
		now := time.Now()
		for k, v := range p.cache {
			if v.expiresAt.Before(now) {
				delete(p.cache, k)
			}
		}
	}
	p.cache[cacheKey] = cachedToken{
		token:     tokenStr,
		expiresAt: time.Now().Add(p.opts.expiration),
	}
	p.mu.Unlock()

	return tokenStr, nil
}
