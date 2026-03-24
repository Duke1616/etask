package bizid

import (
	"context"
	"fmt"
	"strconv"

	"google.golang.org/grpc/metadata"
)

// Set 将 biz_id 写入 context
func Set(ctx context.Context, bizID int64) context.Context {
	return context.WithValue(ctx, ContextKey, bizID)
}

// AppendToOutgoing 将 bizID 放入 outgoing metadata
func AppendToOutgoing(ctx context.Context, bizID int64) context.Context {
	return metadata.AppendToOutgoingContext(ctx, MetadataKey, strconv.FormatInt(bizID, 10))
}

// SetAlert 设置告警模块的 biz_id
func SetAlert(ctx context.Context) context.Context {
	return Set(ctx, Alert)
}

// SetTicket 设置工单模块的 biz_id
func SetTicket(ctx context.Context) context.Context {
	return Set(ctx, Ticket)
}

// SetTask 设置任务模块的 biz_id
func SetTask(ctx context.Context) context.Context {
	return Set(ctx, Task)
}

// FromContext 从 context 中获取 biz_id
func FromContext(ctx context.Context) (int64, error) {
	v := ctx.Value(ContextKey)
	if v == nil {
		return 0, fmt.Errorf("context 中缺少 %s", ContextKey)
	}
	id, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("%s 类型断言失败，实际类型: %T", ContextKey, v)
	}
	return id, nil
}
