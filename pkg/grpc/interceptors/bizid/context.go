package bizid

import (
	"context"
	"encoding/json"
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
		return 0, fmt.Errorf("context 中缺少 biz_id")
	}

	switch val := v.(type) {
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	case float64:
		return int64(val), nil
	case json.Number:
		return val.Int64()
	case string:
		return strconv.ParseInt(val, 10, 64)
	default:
		return 0, fmt.Errorf("%s 类型不支持: %T", ContextKey, v)
	}
}
