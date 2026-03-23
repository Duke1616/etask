package jwt

// BizType 业务模块标识，用于区分任务的创建来源
const (
	BizTypeAlert  int64 = 1 // 告警模版
	BizTypeTicket int64 = 2 // 工单模版
	BizTypeTask   int64 = 3 // 任务模块
)
