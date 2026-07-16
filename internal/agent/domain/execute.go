package domain

const ServiceName = "agent"

// ExecutionOutput 汇总 Agent 执行产生的结构化结果和日志。
type ExecutionOutput struct {
	Result string
	Logs   []string
}
