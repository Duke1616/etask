package ioc

import (
	"context"

	jwtinterceptor "github.com/Duke1616/etask/pkg/grpc/interceptors/jwt"
	"github.com/gotomicro/ego/core/elog"
	"github.com/redis/go-redis/v9"
)

// InitClusterKeyManager 初始化集群 RSA 密钥管理器单例。
// 若 Redis 客户端未就绪 (例如 Executor 单机运行或未配置 Redis)，则安全返回 nil。
func InitClusterKeyManager(rdb redis.UniversalClient) jwtinterceptor.IClusterKeyManager {
	if rdb == nil {
		return nil
	}
	km, err := jwtinterceptor.NewClusterKeyManager(context.Background(), rdb, jwtinterceptor.DefaultClusterKeyID, "")
	if err != nil {
		elog.DefaultLogger.Warn("初始化集群 RSA 密钥管理器失败", elog.FieldErr(err))
		return nil
	}
	return km
}
