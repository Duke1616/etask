package executor

import (
	"encoding/json"
	"strconv"

	reporterv1 "github.com/Duke1616/ework-runner/api/proto/gen/etask/reporter/v1"
	"github.com/gotomicro/ego/core/elog"
)

type Variable struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

// TaskHandler 任务处理函数接口
type TaskHandler interface {
	// Name 处理器名称
	Name() string
	// Desc 处理器功能详情信息
	Desc() string
	// Run 处理器具体执行
	Run(*Context) error
}

// Context 任务执行上下文
// Context 任务执行上下文
type Context struct {
	ExecutionID int64
	TaskID      int64
	TaskName    string
	HandlerName string
	Params      map[string]string

	// 内部字段
	reporter reporterv1.ReporterServiceClient
	logger   *elog.Component

	// 日志模块
	taskLogger TaskLogger
}

// newContext 创建上下文(内部使用)
func newContext(eid, taskID int64, taskName, handlerName string, params map[string]string,
	reporter reporterv1.ReporterServiceClient, logger *elog.Component) *Context {

	var masks []string
	if variablesStr := params["variables"]; variablesStr != "" {
		var vars []Variable
		if err := json.Unmarshal([]byte(variablesStr), &vars); err == nil {
			for _, v := range vars {
				if v.Secret && v.Value != "" {
					masks = append(masks, v.Value)
				}
			}
		}
	}

	ctx := &Context{
		ExecutionID: eid,
		TaskID:      taskID,
		TaskName:    taskName,
		HandlerName: handlerName,
		Params:      params,
		reporter:    reporter,
		logger:      logger,
		taskLogger:  newTaskLogger(eid, reporter, logger, masks),
	}

	return ctx
}

// Log 记录日志 (代理给 taskLogger)
func (c *Context) Log(format string, args ...any) {
	c.taskLogger.Log(format, args...)
}

// Close 关闭 Context，清理资源
func (c *Context) Close() {
	if c.taskLogger != nil {
		c.taskLogger.Close()
	}
}

// Param 获取字符串参数
func (c *Context) Param(key string) string {
	return c.Params[key]
}

// ParamInt 获取整数参数
func (c *Context) ParamInt(key string) int {
	val := c.Params[key]
	if val == "" {
		return 0
	}
	i, _ := strconv.Atoi(val)
	return i
}

// ParamInt64 获取 int64 参数
func (c *Context) ParamInt64(key string) int64 {
	val := c.Params[key]
	if val == "" {
		return 0
	}
	i, _ := strconv.ParseInt(val, 10, 64)
	return i
}

// ParamBool 获取布尔参数
func (c *Context) ParamBool(key string) bool {
	val := c.Params[key]
	if val == "" {
		return false
	}
	b, _ := strconv.ParseBool(val)
	return b
}

// ReportProgress 上报进度 (可选)
// NOTE: 对于没有进度的任务,不调用此方法也完全OK
func (c *Context) ReportProgress(progress int) error {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	// TODO: 实现进度上报
	// 当前简化版本,可以后续增强
	c.logger.Debug("进度上报", elog.Int("progress", progress))
	return nil
}

// Logger 获取日志组件
func (c *Context) Logger() *elog.Component {
	return c.logger.With(
		elog.Int64("executionID", c.ExecutionID),
		elog.Int64("taskID", c.TaskID),
		elog.String("taskName", c.TaskName),
	)
}
