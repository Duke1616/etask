package tenant

import (
	"context"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryServerInterceptor 从 metadata 中获取 x-tenant-id，并注入 context
// 专门用于内网微服务未开启 JWT 认证，但需要安全传递 tenant_id 的场景
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(injectTenantContext(ctx), req)
	}
}

// StreamServerInterceptor 从 metadata 中获取 x-tenant-id 并注入 Stream context
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapped := &wrappedStream{
			ServerStream: ss,
			ctx:          injectTenantContext(ss.Context()),
		}
		return handler(srv, wrapped)
	}
}

// injectTenantContext 提取元数据中的租户 ID 并注入上下文
func injectTenantContext(ctx context.Context) context.Context {
	if tid, ok := extractTenantID(ctx); ok {
		ctx = Set(ctx, tid)
	}
	return ctx
}

// extractTenantID 从 incoming metadata 中安全提取租户 ID
func extractTenantID(ctx context.Context) (int64, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, false
	}

	tenantIDs := md.Get(MetadataKey)
	if len(tenantIDs) == 0 {
		return 0, false
	}

	tid, err := strconv.ParseInt(tenantIDs[0], 10, 64)
	if err != nil || tid <= 0 {
		return 0, false
	}

	return tid, true
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
