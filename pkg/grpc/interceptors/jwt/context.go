package jwt

import (
	"context"

	"github.com/Duke1616/eiam/pkg/ctxutil"
	"github.com/golang-jwt/jwt/v4"
)

// Set 解析受信的 JWT Claims 并提取 tenant_id 与 user_id 强类型注入 Context
func Set(ctx context.Context, claims jwt.MapClaims) context.Context {
	if claims == nil {
		return ctx
	}

	// 提取并强转租户 ID
	if tenantVal, ok := claims["tenant_id"]; ok {
		tid := parseID(tenantVal)
		if tid > 0 {
			ctx = ctxutil.WithTenantID(ctx, tid)
			ctx = ctxutil.WithOriginTenantID(ctx, tid)
		}
	}

	// 提取并强转用户 ID
	if uidVal, ok := claims["user_id"]; ok {
		uid := parseID(uidVal)
		if uid > 0 {
			ctx = ctxutil.WithUserID(ctx, uid)
		}
	}

	return ctx
}

// parseID 处理 JWT claims 解密后不同数值类型的防呆转换
func parseID(val interface{}) int64 {
	switch v := val.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return 0
	}
}
