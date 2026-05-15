package jwt

import (
	"context"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc/metadata"
)

func ContextWithJWT(ctx context.Context, key string) context.Context {
	jwtAuth := NewJwtAuth(key)
	tokenString, err := jwtAuth.Encode(jwt.MapClaims{})
	if err != nil {
		return ctx
	}

	// 同样改成追加
	return metadata.AppendToOutgoingContext(ctx, AuthorizationKey, BearerPrefix+tokenString)
}
