package tenant

import (
	"context"

	"google.golang.org/grpc"
)

// UnaryClientInterceptor 客户端一元拦截器
// 自动将本地 Context 中的 tenant_id 提取并注入到 gRPC Metadata 中传递给内网下游
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if tid, err := FromContext(ctx); err == nil && tid > 0 {
			ctx = AppendToOutgoing(ctx, tid)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor 客户端流拦截器
// 自动将本地 Context 中的 tenant_id 注入到 gRPC Metadata 中传递给内网下游
func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if tid, err := FromContext(ctx); err == nil && tid > 0 {
			ctx = AppendToOutgoing(ctx, tid)
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}
