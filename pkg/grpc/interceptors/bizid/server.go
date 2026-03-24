package bizid

import (
	"context"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryServerInterceptor 从 metadata 中获取 x-biz-id，并注入 context
// 适合在不开启 JWT 认证但需要传递 biz_id 的场景下使用
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 提取 metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}

		// 获取 x-biz-id
		bizIDs := md.Get(MetadataKey)
		if len(bizIDs) > 0 {
			bizID, _ := strconv.ParseInt(bizIDs[0], 10, 64)
			if bizID > 0 {
				ctx = context.WithValue(ctx, ContextKey, bizID)
			}
		}

		return handler(ctx, req)
	}
}

// StreamServerInterceptor 从 metadata 中获取 x-biz-id，并注入 context
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		// 提取 metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			// 获取 x-biz-id
			bizIDs := md.Get(MetadataKey)
			if len(bizIDs) > 0 {
				bizID, _ := strconv.ParseInt(bizIDs[0], 10, 64)
				if bizID > 0 {
					ctx = context.WithValue(ctx, ContextKey, bizID)
				}
			}
		}

		// 注意：StreamServer 需要使用带有新 Context 的包装过的 Stream
		wrapped := &wrappedStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrapped)
	}
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
