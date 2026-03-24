package bizid

import (
	"context"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryClientInterceptor 客户端一元拦截器
// 自动将 context 中的 biz_id (本地值) 提取并注入到 gRPC Metadata (跨网络传输) 中
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 1. 从 context 获取 biz_id
		if v := ctx.Value(ContextKey); v != nil {
			if bizID, ok := v.(int64); ok && bizID > 0 {
				// 2. 注入到 metadata (即：x-biz-id)
				ctx = metadata.AppendToOutgoingContext(ctx, MetadataKey, strconv.FormatInt(bizID, 10))
			}
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor 客户端流拦截器
// 自动将 context 中的 biz_id 提取并注入到 gRPC Metadata 中
func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if v := ctx.Value(ContextKey); v != nil {
			if bizID, ok := v.(int64); ok && bizID > 0 {
				ctx = metadata.AppendToOutgoingContext(ctx, MetadataKey, strconv.FormatInt(bizID, 10))
			}
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}
