package jwt

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc/metadata"
)

// ContextWithJWT 创建带有 JWT 的 context
// defaultBizID: 如果 context 中没有 biz_id,使用此默认值
func ContextWithJWT(ctx context.Context, key string, defaultBizID int64) context.Context {
	// 使用项目已有的JWT包创建令牌
	jwtAuth := NewJwtAuth(key)

	// NOTE: 优先从 context 中获取 biz_id,如果没有则使用默认值
	bizID := defaultBizID
	if v := ctx.Value(BizIDName); v != nil {
		if id, ok := v.(int64); ok {
			bizID = id
		}
	}

	claims := jwt.MapClaims{
		BizIDName: float64(bizID),
	}

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

// GetBizIDFromContext 从 context 中获取 biz_id，由 JwtAuthInterceptor 解码后注入
func GetBizIDFromContext(ctx context.Context) (int64, error) {
	v := ctx.Value(BizIDName)
	if v == nil {
		return 0, fmt.Errorf("context 中缺少 %s", BizIDName)
	}
	id, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("%s 类型断言失败，实际类型: %T", BizIDName, v)
	}
	return id, nil
}
