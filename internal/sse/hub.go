package sse

import (
	"sync"

	"github.com/Duke1616/etask/pkg/sse"
)

const (
	TASK_STATUS_CHANGE_EVENT = "task_status_change"
	TASK_LOG_EVENT           = "task_log"
	TASK_EXECUTION_EVENT     = "task_execution"
)

// TaskStatusEvent 任务状态变更事件定义
type TaskStatusEvent struct {
	TaskID   int64  `json:"task_id"`
	Status   string `json:"status"`
	NextTime int64  `json:"next_time"`
}

// TaskLogEvent 任务日志实时推送事件定义
type TaskLogEvent struct {
	ID          int64  `json:"id"`
	TaskID      int64  `json:"task_id"`
	ExecutionID int64  `json:"execution_id"`
	Content     string `json:"content"`
	CTime       int64  `json:"c_time"`
}

// TaskExecutionEvent 任务执行记录变更事件定义
type TaskExecutionEvent struct {
	ID              int64  `json:"id"`
	TaskID          int64  `json:"task_id"`
	TaskName        string `json:"task_name"`
	StartTime       int64  `json:"start_time"`
	EndTime         int64  `json:"end_time"`
	Status          string `json:"status"`
	RunningProgress int32  `json:"running_progress"`
	ExecutorNodeId  string `json:"executor_node_id"`
	TaskResult      string `json:"task_result"`
	CTime           int64  `json:"c_time"`
}

var (
	globalSSEHub      *sse.Hub[TaskStatusEvent]
	executionLogsHub  *sse.TopicHub[int64, TaskLogEvent]
	taskExecutionsHub *sse.TopicHub[int64, TaskExecutionEvent]
	once              sync.Once
	logOnce           sync.Once
	execOnce          sync.Once
)

// GetSSEHub 获取全局任务状态推送中心单例
func GetSSEHub() *sse.Hub[TaskStatusEvent] {
	once.Do(func() {
		globalSSEHub = sse.NewHub[TaskStatusEvent]()
	})
	return globalSSEHub
}

// GetExecutionLogsHub 获取全局日志推送中心单例
func GetExecutionLogsHub() *sse.TopicHub[int64, TaskLogEvent] {
	logOnce.Do(func() {
		executionLogsHub = sse.NewTopicHub[int64, TaskLogEvent]()
	})
	return executionLogsHub
}

// GetTaskExecutionsHub 获取全局执行记录推送中心单例
func GetTaskExecutionsHub() *sse.TopicHub[int64, TaskExecutionEvent] {
	execOnce.Do(func() {
		taskExecutionsHub = sse.NewTopicHub[int64, TaskExecutionEvent]()
	})
	return taskExecutionsHub
}
