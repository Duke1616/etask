package jwt

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	cryptorsa "github.com/Duke1616/etask/pkg/crypto/rsa"
	"github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/golang-jwt/jwt/v4"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	DefaultClusterKeyID          = "default"
	ClusterSigningKeyPrefix      = "etask:auth:scheduler:signing_key:"
	ClusterPublicKeyPrefix       = "etask:auth:scheduler:public_key:"
	ClusterBlacklistedNodePrefix = "etask:auth:blacklist:nodes:"
)

// PublicKeyProvider 公钥动态提供函数契约
type PublicKeyProvider func() (*rsa.PublicKey, error)

// IClusterKeyManager 集群 RSA 签名与密钥管理契约
type IClusterKeyManager interface {
	KeyID() string
	SignJWT(claims jwt.MapClaims) (string, error)
	GetPublicKey() *rsa.PublicKey
	ExportPublicKeyPEM() string
	BlockNode(ctx context.Context, nodeID string, ttl time.Duration) error
	IsNodeBlocked(ctx context.Context, nodeID string) (bool, error)
}

// clusterKeyManager 集群私钥与公钥管理器（初始化后完全不可变，天然并发安全，无需加锁）
type clusterKeyManager struct {
	keyID      string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	pubPEM     string
	client     redis.Cmdable
}

// NewClusterKeyManager 创建或从 Redis 集群原子共享获取 RSA 签名密钥管理器。
func NewClusterKeyManager(ctx context.Context, client redis.Cmdable, keyID, privateKeyPEM string) (IClusterKeyManager, error) {
	if keyID == "" {
		keyID = DefaultClusterKeyID
	}

	pemBytes := []byte(strings.TrimSpace(privateKeyPEM))

	// 若未显式传入私钥且 Redis 客户端有效，则尝试从 Redis 集群原子拉取或生成
	if len(pemBytes) == 0 && client != nil {
		clusterPEM, err := getOrSetClusterSigningKey(ctx, client, keyID)
		if err != nil {
			return nil, fmt.Errorf("集群共享签名密钥初始化失败: %w", err)
		}
		pemBytes = []byte(clusterPEM)
	}

	priv, err := parseOrGeneratePrivateKey(pemBytes)
	if err != nil {
		return nil, err
	}

	pubPEM, err := cryptorsa.EncodePublicKeyPEM(&priv.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("编码 RSA 公钥失败: %w", err)
	}

	// 确保 Redis 中同步存在可公开访问的公钥 PEM (采用 SetNX 避免多副本并发启动重复写入)
	if client != nil {
		_ = client.SetNX(ctx, ClusterPublicKeyPrefix+keyID, pubPEM, 0).Err()
	}

	return &clusterKeyManager{
		keyID:      keyID,
		privateKey: priv,
		publicKey:  &priv.PublicKey,
		pubPEM:     pubPEM,
		client:     client,
	}, nil
}

func getOrSetClusterSigningKey(ctx context.Context, client redis.Cmdable, keyID string) (string, error) {
	key := ClusterSigningKeyPrefix + keyID
	val, err := client.Get(ctx, key).Result()
	if err == nil && val != "" {
		return val, nil
	}
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", err
	}

	newPEM, genErr := cryptorsa.GenerateRSAPEM(2048)
	if genErr != nil {
		return "", genErr
	}

	ok, setErr := client.SetNX(ctx, key, newPEM, 0).Result()
	if setErr != nil {
		return "", setErr
	}
	if !ok {
		// 并发竞争中由其他节点写入成功，重新读取获取权威 PEM
		return client.Get(ctx, key).Result()
	}
	return newPEM, nil
}

func parseOrGeneratePrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	if len(pemBytes) == 0 {
		return cryptorsa.GenerateRSAKey(2048)
	}
	return cryptorsa.ParsePrivateKeyPEM(string(pemBytes))
}

func (m *clusterKeyManager) KeyID() string {
	return m.keyID
}

func (m *clusterKeyManager) GetPublicKey() *rsa.PublicKey {
	return m.publicKey
}

func (m *clusterKeyManager) ExportPublicKeyPEM() string {
	return m.pubPEM
}

// SignJWT 使用集群 RSA 私钥签发 RS256 格式的 JWT 令牌 (不可变对象直接读取，零锁开销)
func (m *clusterKeyManager) SignJWT(customClaims jwt.MapClaims) (string, error) {
	claims := jwt.MapClaims{
		"iat": time.Now().Unix(),
		"iss": DefaultIssuer,
		"exp": time.Now().Add(DefaultExpiration).Unix(),
	}
	for k, v := range customClaims {
		claims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = m.keyID

	return token.SignedString(m.privateKey)
}

// BlockNode 将指定 NodeID 写入黑名单，实现秒级下线与回收
func (m *clusterKeyManager) BlockNode(ctx context.Context, nodeID string, ttl time.Duration) error {
	if m.client == nil || nodeID == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return m.client.Set(ctx, ClusterBlacklistedNodePrefix+nodeID, "1", ttl).Err()
}

// IsNodeBlocked 检查目标节点是否处于黑名单中
func (m *clusterKeyManager) IsNodeBlocked(ctx context.Context, nodeID string) (bool, error) {
	if m.client == nil || nodeID == "" {
		return false, nil
	}
	exists, err := m.client.Exists(ctx, ClusterBlacklistedNodePrefix+nodeID).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// --- 通用公钥缓存与并发加载器（模板化收敛 singleflight、本地缓存与容灾降级） ---

type cachedKeyProvider struct {
	cacheKey   string
	cacheTTL   time.Duration
	loader     func(ctx context.Context) (string, error)
	sf         singleflight.Group
	mu         sync.RWMutex
	cachedKey  *rsa.PublicKey
	cachedTime time.Time
}

func newCachedKeyProvider(cacheKey string, cacheTTL time.Duration, loader func(ctx context.Context) (string, error)) *cachedKeyProvider {
	return &cachedKeyProvider{
		cacheKey: cacheKey,
		cacheTTL: cacheTTL,
		loader:   loader,
	}
}

// GetPublicKey 获取公钥：优先读本地有效缓存，失效时通过 singleflight 归并穿透加载，网络抖动时优雅降级
func (p *cachedKeyProvider) GetPublicKey() (*rsa.PublicKey, error) {
	// 1. 快速读取有效本地缓存
	p.mu.RLock()
	if p.cachedKey != nil && time.Since(p.cachedTime) < p.cacheTTL {
		pk := p.cachedKey
		p.mu.RUnlock()
		return pk, nil
	}
	staleKey := p.cachedKey
	p.mu.RUnlock()

	// 2. 缓存失效时，使用 singleflight 抑制并发击穿 (惊群效应)
	res, err, _ := p.sf.Do(p.cacheKey, func() (interface{}, error) {
		// 双重检测锁 (DCL)
		p.mu.RLock()
		if p.cachedKey != nil && time.Since(p.cachedTime) < p.cacheTTL {
			pk := p.cachedKey
			p.mu.RUnlock()
			return pk, nil
		}
		p.mu.RUnlock()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		pemStr, loadErr := p.loader(ctx)
		if loadErr != nil {
			// 容灾降级：网络抖动时优雅降级返回历史公钥
			if staleKey != nil {
				return staleKey, nil
			}
			return nil, loadErr
		}

		pubKey, parseErr := cryptorsa.ParsePublicKeyPEM(pemStr)
		if parseErr != nil {
			if staleKey != nil {
				return staleKey, nil
			}
			return nil, fmt.Errorf("解析公钥 PEM 失败: %w", parseErr)
		}

		p.mu.Lock()
		p.cachedKey = pubKey
		p.cachedTime = time.Now()
		p.mu.Unlock()

		return pubKey, nil
	})

	if err != nil {
		return nil, err
	}
	return res.(*rsa.PublicKey), nil
}

// NewRedisPublicKeyProvider 创建基于 Redis 的公钥提供者
func NewRedisPublicKeyProvider(client redis.Cmdable, keyID string, cacheTTL time.Duration) PublicKeyProvider {
	if keyID == "" {
		keyID = DefaultClusterKeyID
	}
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}
	provider := newCachedKeyProvider(keyID, cacheTTL, func(ctx context.Context) (string, error) {
		if client == nil {
			return "", errors.New("redis 客户端未就绪，无法获取公钥")
		}
		pemStr, err := client.Get(ctx, ClusterPublicKeyPrefix+keyID).Result()
		if err != nil {
			return "", fmt.Errorf("从 Redis 拉取公钥失败: %w", err)
		}
		return pemStr, nil
	})
	return provider.GetPublicKey
}

// NewRegistryPublicKeyProvider 从注册中心 (如 Etcd) 服务实例元数据拉取调度中心发布的 RSA 公钥。
// 彻底解耦 Executor 节点对中心 Redis 的直连依赖，保障生产网络隔离与安全。
func NewRegistryPublicKeyProvider(reg registry.Registry, serviceName string, cacheTTL time.Duration) PublicKeyProvider {
	if serviceName == "" {
		serviceName = "scheduler"
	}
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}
	provider := newCachedKeyProvider(serviceName, cacheTTL, func(ctx context.Context) (string, error) {
		if reg == nil {
			return "", errors.New("注册中心未就绪，无法获取公钥")
		}
		instances, err := reg.ListServices(ctx, serviceName)
		if err != nil {
			return "", fmt.Errorf("从注册中心拉取服务 %s 实例失败: %w", serviceName, err)
		}
		for _, inst := range instances {
			if keyVal, ok := inst.Metadata["public_key"]; ok {
				if s, ok := keyVal.(string); ok && strings.TrimSpace(s) != "" {
					return s, nil
				}
			}
		}
		return "", fmt.Errorf("未在 %s 服务实例元数据中发现公钥 public_key", serviceName)
	})
	return provider.GetPublicKey
}
