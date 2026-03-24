package jwt

import (
	"context"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc/metadata"
)

// ContextWithJWT 创建带有 JWT 的 context
func ContextWithJWT(ctx context.Context, key string) context.Context {
	// 使用项目已有的JWT包创建令牌
	jwtAuth := NewJwtAuth(key)

	// 此时不再附带 bizID 的逻辑
	claims := jwt.MapClaims{}

	// 使用JWT认证包的Encode方法生成令牌
	tokenString, err := jwtAuth.Encode(claims)
	if err != nil {
		// NOTE: 如果生成失败,返回原始 context
		return ctx
	}

	// 创建带有授权信息的元数据
	md := metadata.New(map[string]string{
		AuthorizationKey: BearerPrefix + tokenString,
	})
	return metadata.NewOutgoingContext(ctx, md)
}
