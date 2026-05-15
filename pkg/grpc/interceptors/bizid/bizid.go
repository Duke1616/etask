package bizid

// 业务模块标识，用于区分任务的创建来源
const (
	Alert  int64 = 1 // 告警模版
	Ticket int64 = 2 // 工单模版
	Task   int64 = 3 // 任务模块
)

type bizIDKey struct{}

var ContextKey bizIDKey

const (
	// MetadataKey gRPC Metadata key
	MetadataKey = "x-biz-id"
)
