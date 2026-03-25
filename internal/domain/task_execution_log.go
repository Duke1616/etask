package domain

// TaskExecutionLog 任务执行日志
type TaskExecutionLog struct {
	ID          int64
	TaskID      int64
	ExecutionID int64
	Content     string
	CTime       int64
}
