//go:build unit

package jwt

import (
	"context"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestClusterKeyManager_SignAndVerify(t *testing.T) {
	ctx := context.Background()

	// 1. 初始化独立 KeyManager
	km, err := NewClusterKeyManager(ctx, nil, "test-cluster-key", "")
	require.NoError(t, err)
	require.NotNil(t, km)
	assert.Equal(t, "test-cluster-key", km.KeyID())
	assert.NotEmpty(t, km.ExportPublicKeyPEM())

	pubKey := km.GetPublicKey()
	require.NotNil(t, pubKey)

	// 2. 签发 RS256 Token
	claims := jwt.MapClaims{
		"tenant_id": float64(100),
		"user_id":   float64(200),
		"aud":       "executor-node-01",
	}
	tokenStr, err := km.SignJWT(claims)
	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)

	// 3. 构建支持 RSA 的服务端拦截器
	jwtAuth := NewJwtAuth("",
		WithPublicKeyProvider(func() (*rsa.PublicKey, error) {
			return pubKey, nil
		}),
		WithExpectedAudience("executor-node-01"),
	)

	decodedClaims, err := jwtAuth.Decode(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, float64(100), decodedClaims["tenant_id"])
	assert.Equal(t, float64(200), decodedClaims["user_id"])
	assert.Equal(t, "executor-node-01", decodedClaims["aud"])
}

func TestJwtAuth_DualMode(t *testing.T) {
	ctx := context.Background()

	// 生成 RSA 密钥对
	km, err := NewClusterKeyManager(ctx, nil, "dual-key", "")
	require.NoError(t, err)

	rsaToken, err := km.SignJWT(jwt.MapClaims{
		"mode": "rsa",
		"aud":  "node-common",
	})
	require.NoError(t, err)

	// 生成 HMAC Token
	hmacAuth := NewJwtAuth("hmac-secret-123")
	hmacToken, err := hmacAuth.Encode(jwt.MapClaims{
		"mode": "hmac",
		"aud":  "node-common",
	})
	require.NoError(t, err)

	// 创建同时配置了 HMAC 密钥与 RSA 公钥提供器的服务端实例
	dualAuth := NewJwtAuth("hmac-secret-123",
		WithPublicKeyProvider(func() (*rsa.PublicKey, error) {
			return km.GetPublicKey(), nil
		}),
		WithExpectedAudience("node-common"),
	)

	testCases := []struct {
		name       string
		token      string
		expectMode string
		wantErr    bool
	}{
		{
			name:       "RS256 非对称令牌验签成功",
			token:      rsaToken,
			expectMode: "rsa",
			wantErr:    false,
		},
		{
			name:       "HS256 对称令牌验签成功 (向后兼容)",
			token:      hmacToken,
			expectMode: "hmac",
			wantErr:    false,
		},
		{
			name:       "篡改令牌验签失败",
			token:      rsaToken + "tampered",
			expectMode: "",
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			claims, decodeErr := dualAuth.Decode(tc.token)
			if tc.wantErr {
				assert.Error(t, decodeErr)
				return
			}
			require.NoError(t, decodeErr)
			assert.Equal(t, tc.expectMode, claims["mode"])
		})
	}
}

func TestJwtAuth_AudienceAndBlacklist(t *testing.T) {
	ctx := context.Background()
	km, err := NewClusterKeyManager(ctx, nil, "aud-key", "")
	require.NoError(t, err)

	// 签发目标受众为 node-01 的令牌
	tokenForNode01, err := km.SignJWT(jwt.MapClaims{
		"aud": "node-01",
	})
	require.NoError(t, err)

	testCases := []struct {
		name             string
		expectedAudience string
		blacklistChecker func(ctx context.Context, nodeID string) (bool, error)
		expectCode       codes.Code
	}{
		{
			name:             "受众匹配成功放行",
			expectedAudience: "node-01",
			expectCode:       codes.OK,
		},
		{
			name:             "受众不匹配拦截 (防止截获令牌跨节点滥用)",
			expectedAudience: "node-02",
			expectCode:       codes.PermissionDenied,
		},
		{
			name:             "目标节点在黑名单中拒绝服务",
			expectedAudience: "node-01",
			blacklistChecker: func(ctx context.Context, nodeID string) (bool, error) {
				return nodeID == "node-01", nil
			},
			expectCode: codes.PermissionDenied,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			serverAuth := NewJwtAuth("",
				WithPublicKeyProvider(func() (*rsa.PublicKey, error) {
					return km.GetPublicKey(), nil
				}),
				WithExpectedAudience(tc.expectedAudience),
				WithBlacklistChecker(tc.blacklistChecker),
			)

			// 模拟传入 gRPC IncomingContext
			md := metadata.Pairs(AuthorizationKey, BearerPrefix+tokenForNode01)
			inCtx := metadata.NewIncomingContext(context.Background(), md)

			_, authErr := serverAuth.Authenticate(inCtx)
			if tc.expectCode == codes.OK {
				assert.NoError(t, authErr)
			} else {
				require.Error(t, authErr)
				st, ok := status.FromError(authErr)
				assert.True(t, ok)
				assert.Equal(t, tc.expectCode, st.Code())
			}
		})
	}
}

func TestTokenProviderAndValidator_Pattern(t *testing.T) {
	ctx := context.Background()
	km, err := NewClusterKeyManager(ctx, nil, "pattern-key", "")
	require.NoError(t, err)

	// 1. 测试 RSATokenProvider
	rsaProvider := NewRSATokenProvider(km)
	ctxWithAud := WithAudience(ctx, "target-executor")
	token, err := rsaProvider.ProvideToken(ctxWithAud)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// 2. 测试服务端 RSA 校验与目标受众断言
	rsaAuth := NewJwtAuth("",
		WithPublicKeyProvider(func() (*rsa.PublicKey, error) {
			return km.GetPublicKey(), nil
		}),
		WithExpectedAudience("target-executor"),
	)
	claims, err := rsaAuth.Decode(token)
	require.NoError(t, err)
	assert.Equal(t, "target-executor", claims["aud"])

	// 目标受众不匹配时拦截
	wrongAudAuth := NewJwtAuth("",
		WithPublicKeyProvider(func() (*rsa.PublicKey, error) {
			return km.GetPublicKey(), nil
		}),
		WithExpectedAudience("wrong-executor"),
	)
	_, err = wrongAudAuth.Decode(token)
	assert.Error(t, err)

	// 3. 测试 HMACTokenProvider 与 HMAC 服务端鉴权
	hmacProvider := NewHMACTokenProvider("my-secret")
	hmacToken, err := hmacProvider.ProvideToken(ctx)
	require.NoError(t, err)

	hmacAuth := NewJwtAuth("my-secret")
	hmacClaims, err := hmacAuth.Decode(hmacToken)
	require.NoError(t, err)
	assert.NotEmpty(t, hmacClaims["iat"])

	// 4. 验证向后兼容性：老客户端静态 HMAC Token 未携带 aud 声明，非严格模式下正常放行
	nonStrictAuth := NewJwtAuth("my-secret",
		WithExpectedAudience("target-executor"),
		WithStrictAudience(false),
	)
	_, err = nonStrictAuth.Decode(hmacToken)
	assert.NoError(t, err, "老客户端缺少 aud 时在非严格模式下应该放行")

	// 严格模式下缺少 aud 则拦截
	strictAuth := NewJwtAuth("my-secret",
		WithExpectedAudience("target-executor"),
		WithStrictAudience(true),
	)
	_, err = strictAuth.Decode(hmacToken)
	assert.Error(t, err, "严格模式下必须包含 aud")
}

func TestRSATokenProvider_Cache(t *testing.T) {
	ctx := context.Background()
	km, err := NewClusterKeyManager(ctx, nil, "cache-test-key", "")
	require.NoError(t, err)

	provider := NewRSATokenProvider(km)
	ctxWithAud := WithAudience(ctx, "cache-target")

	// 第一次生成
	token1, err := provider.ProvideToken(ctxWithAud)
	require.NoError(t, err)

	// 第二次生成相同目标，命中内存缓存复用
	token2, err := provider.ProvideToken(ctxWithAud)
	require.NoError(t, err)
	assert.Equal(t, token1, token2, "未过期前应该复用已签发 Token")

	// 换目标受众，重新生成不同 Token
	ctxWithOtherAud := WithAudience(ctx, "other-target")
	token3, err := provider.ProvideToken(ctxWithOtherAud)
	require.NoError(t, err)
	assert.NotEqual(t, token1, token3, "不同受众应该签发独立 Token")
}

func TestDualMode_NativeKeyfunc(t *testing.T) {
	ctx := context.Background()
	km, err := NewClusterKeyManager(ctx, nil, "composite-route-key", "")
	require.NoError(t, err)

	rsaProvider := NewRSATokenProvider(km)
	hmacProvider := NewHMACTokenProvider("shared-secret")

	rsaToken, err := rsaProvider.ProvideToken(ctx)
	require.NoError(t, err)
	hmacToken, err := hmacProvider.ProvideToken(ctx)
	require.NoError(t, err)

	// 单一 InterceptorBuilder 同时支持 HMAC 与 RSA，由标准库 Keyfunc 自动安全解析 Header
	dualAuth := NewJwtAuth("shared-secret",
		WithPublicKeyProvider(func() (*rsa.PublicKey, error) { return km.GetPublicKey(), nil }),
	)

	// 验证双轨算法均能成功通过标准库验签
	rsaClaims, err := dualAuth.Decode(rsaToken)
	require.NoError(t, err)
	assert.NotEmpty(t, rsaClaims["iat"])

	hmacClaims, err := dualAuth.Decode(hmacToken)
	require.NoError(t, err)
	assert.NotEmpty(t, hmacClaims["iat"])
}

func TestRedisPublicKeyProvider_NilClient(t *testing.T) {
	provider := NewRedisPublicKeyProvider(nil, "any-key", time.Minute)
	pk, err := provider()
	assert.Nil(t, pk)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis 客户端未就绪")
}

type fakeRegistry struct {
	instances map[string][]registry.ServiceInstance
	err       error
}

func (f *fakeRegistry) Register(ctx context.Context, si registry.ServiceInstance) error   { return nil }
func (f *fakeRegistry) UnRegister(ctx context.Context, si registry.ServiceInstance) error { return nil }
func (f *fakeRegistry) ListServices(ctx context.Context, name string) ([]registry.ServiceInstance, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.instances[name], nil
}
func (f *fakeRegistry) Subscribe(name string) <-chan registry.Event { return nil }
func (f *fakeRegistry) Close() error                                { return nil }

func TestRegistryPublicKeyProvider(t *testing.T) {
	ctx := context.Background()
	km, err := NewClusterKeyManager(ctx, nil, "reg-key", "")
	require.NoError(t, err)

	reg := &fakeRegistry{
		instances: map[string][]registry.ServiceInstance{
			"scheduler": {
				{
					Name: "scheduler",
					ID:   "scheduler-01",
					Metadata: map[string]any{
						"public_key": km.ExportPublicKeyPEM(),
					},
				},
			},
		},
	}

	// 1. 正常从注册中心元数据获取公钥
	provider := NewRegistryPublicKeyProvider(reg, "scheduler", time.Minute)
	pubKey, err := provider()
	require.NoError(t, err)
	require.NotNil(t, pubKey)
	assert.Equal(t, km.GetPublicKey().N, pubKey.N)

	// 2. 二次调用命中缓存
	pubKey2, err := provider()
	require.NoError(t, err)
	assert.Equal(t, pubKey, pubKey2)

	// 3. 容灾降级：注册中心模拟网络错误，降级返回历史公钥
	reg.err = errors.New("etcd connection timeout")
	// 强制等待过期或测试降级逻辑
	staleProvider := &cachedKeyProvider{
		cacheKey:   "scheduler",
		cacheTTL:   time.Millisecond,
		cachedKey:  pubKey,
		cachedTime: time.Now().Add(-time.Hour), // 模拟已过期
		loader: func(ctx context.Context) (string, error) {
			return "", reg.err
		},
	}
	fallbackKey, err := staleProvider.GetPublicKey()
	require.NoError(t, err, "应该降级返回历史公钥")
	assert.Equal(t, pubKey, fallbackKey)

	// 4. 空注册中心报错
	nilRegProvider := NewRegistryPublicKeyProvider(nil, "scheduler", time.Minute)
	_, err = nilRegProvider()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "注册中心未就绪")
}

