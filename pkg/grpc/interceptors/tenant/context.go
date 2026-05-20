package tenant

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Duke1616/eiam/pkg/ctxutil"
	"google.golang.org/grpc/metadata"
)

// Set 将 tenant_id 写入 context，并设置原始租户 ID (origin_tenant_id)
func Set(ctx context.Context, tenantID int64) context.Context {
	ctx = ctxutil.WithTenantID(ctx, tenantID)
	ctx = ctxutil.WithOriginTenantID(ctx, tenantID)
	return ctx
}

// AppendToOutgoing 将 tenantID 放入 outgoing metadata，支持内网透传
func AppendToOutgoing(ctx context.Context, tenantID int64) context.Context {
	return metadata.AppendToOutgoingContext(ctx, MetadataKey, strconv.FormatInt(tenantID, 10))
}

// FromContext 从 context 中安全获取租户 ID，若无则返回错误
func FromContext(ctx context.Context) (int64, error) {
	tid := ctxutil.GetTenantID(ctx)
	if tid <= 0 {
		return 0, fmt.Errorf("context 中缺少 tenant_id")
	}
	return tid.Int64(), nil
}
