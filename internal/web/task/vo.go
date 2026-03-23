package task

type CreateTaskReq struct {
	Name                string            `json:"name"`
	Type                string            `json:"type"`      // 任务类型: RECURRING-定时任务, ONE_TIME-一次性任务
	CronExpr            string            `json:"cron_expr"` // cron 表达式（定时任务必填，一次性任务可选用于定时触发）
	GrpcConfig          *GrpcConfig       `json:"grpc_config"`
	HTTPConfig          *HTTPConfig       `json:"http_config"`
	RetryConfig         *RetryConfig      `json:"retry_config"`
	MaxExecutionSeconds int64             `json:"max_execution_seconds"` // 最大执行秒数，默认24小时
	ScheduleParams      map[string]string `json:"schedule_params"`       // 调度参数（如分页偏移量、处理进度等）
}

type GrpcConfig struct {
	ServiceName string            `json:"service_name"` // 服务名称
	AuthToken   string            `json:"auth_token"`   // 认证 token
	HandlerName string            `json:"handler_name"` // 执行节点支持的方法名称， 如 shell、python、demo
	Params      map[string]string `json:"params"`       // 传递参数
}

type HTTPConfig struct {
	Endpoint string            `json:"endpoint"`
	Params   map[string]string `json:"params"`
}

type RetryConfig struct {
	MaxRetries      int32 `json:"max_retries"`
	InitialInterval int64 `json:"initial_interval"` // 毫秒
	MaxInterval     int64 `json:"max_interval"`     // 毫秒
}

type GetLogsReq struct {
	ExecutionID int64 `json:"execution_id" form:"execution_id"`
	MinID       int64 `json:"min_id" form:"min_id"`
	Limit       int   `json:"limit" form:"limit"`
}

type PageReq struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type TaskVO struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	Type                string `json:"type"`
	CronExpr            string `json:"cron_expr"`
	Status              string `json:"status"`
	NextTime            int64  `json:"next_time"`
	MaxExecutionSeconds int64  `json:"max_execution_seconds"`
}

type ListTaskResp struct {
	Total int64    `json:"total"`
	Tasks []TaskVO `json:"tasks"`
}
