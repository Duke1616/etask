package executor

import (
	"encoding/json"
	"strconv"
	"sync"

	reporterv1 "github.com/Duke1616/etask/api/proto/gen/etask/reporter/v1"
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

// HandlerMeta 处理器元数据 (用于序列化和展示)
type HandlerMeta struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// Context 任务执行上下文
type Context struct {
	ExecutionID int64
	TaskID      int64
	TaskName    string
	HandlerName string
	Params      map[string]string

	// 结果流处理模块
	results map[string]any
	resLock sync.RWMutex

	// 内部字段
	reporter reporterv1.ReporterServiceClient
	logger   *elog.Component

	// 日志模块
	taskLogger TaskLogger
}

// NewContext 创建上下文 (供 gRPC 模式使用)
func NewContext(eid, taskID int64, taskName, handlerName string, params map[string]string,
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

	return &Context{
		ExecutionID: eid,
		TaskID:      taskID,
		TaskName:    taskName,
		HandlerName: handlerName,
		Params:      params,
		results:     make(map[string]any),
		reporter:    reporter,
		logger:      logger,
		taskLogger:  newTaskLogger(eid, reporter, logger, masks),
	}
}

// NewContextWithLogger 创建带有指定 Logger 的上下文 (供 Kafka 等非 gRPC 模式使用)
func NewContextWithLogger(eid, taskID int64, taskName, handlerName string, params map[string]string,
	logger *elog.Component, taskLogger TaskLogger) *Context {
	return &Context{
		ExecutionID: eid,
		TaskID:      taskID,
		TaskName:    taskName,
		HandlerName: handlerName,
		Params:      params,
		results:     make(map[string]any),
		logger:      logger,
		taskLogger:  taskLogger,
	}
}

// AddResult 合并部分结果数据
func (c *Context) AddResult(data map[string]any) {
	c.resLock.Lock()
	defer c.resLock.Unlock()

	for k, v := range data {
		c.results[k] = v
	}
}

// SetResult 设置单个结果键值对
func (c *Context) SetResult(key string, value any) {
	c.resLock.Lock()
	defer c.resLock.Unlock()
	c.results[key] = value
}

// SetResults 批量设置结果（替换现有结果）
func (c *Context) SetResults(data map[string]any) {
	c.resLock.Lock()
	defer c.resLock.Unlock()
	c.results = data
}

// GetResultJson 获取最终合并后的结果 JSON 字符串
func (c *Context) GetResultJson() string {
	c.resLock.RLock()
	defer c.resLock.RUnlock()

	if len(c.results) == 0 {
		return ""
	}

	bytes, err := json.Marshal(c.results)
	if err != nil {
		c.logger.Error("序列化任务结果失败", elog.FieldErr(err))
		return ""
	}
	return string(bytes)
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
	i, err := strconv.Atoi(val)
	if err != nil {
		c.logger.Warn("参数解析为整数失败", elog.String("key", key), elog.String("value", val), elog.FieldErr(err))
		return 0
	}
	return i
}

// ParamInt64 获取 int64 参数
func (c *Context) ParamInt64(key string) int64 {
	val := c.Params[key]
	if val == "" {
		return 0
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		c.logger.Warn("参数解析为 int64 失败", elog.String("key", key), elog.String("value", val), elog.FieldErr(err))
		return 0
	}
	return i
}

// ParamBool 获取布尔参数
func (c *Context) ParamBool(key string) bool {
	val := c.Params[key]
	if val == "" {
		return false
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		c.logger.Warn("参数解析为布尔值失败", elog.String("key", key), elog.String("value", val), elog.FieldErr(err))
		return false
	}
	return b
}

// ReportProgress 上报进度 (可选)
func (c *Context) ReportProgress(progress int) error {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

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
