package task

// 本文件集中实现 Context 的并发安全结果管理。

import (
	"encoding/json"
	"maps"

	"github.com/gotomicro/ego/core/elog"
)

// AddResult 将一组字段合并到任务结果。
func (c *Context) AddResult(data map[string]any) {
	c.resLock.Lock()
	defer c.resLock.Unlock()
	for key, value := range data {
		c.results[key] = value
	}
}

// SetResult 设置一个任务结果字段。
func (c *Context) SetResult(key string, value any) {
	c.resLock.Lock()
	defer c.resLock.Unlock()
	c.results[key] = value
}

// SetResults 使用给定字段替换当前任务结果。
func (c *Context) SetResults(data map[string]any) {
	c.resLock.Lock()
	defer c.resLock.Unlock()
	c.results = maps.Clone(data)
}

// ResultJSON 返回序列化后的任务结果；没有结果时返回空字符串。
func (c *Context) ResultJSON() string {
	c.resLock.RLock()
	defer c.resLock.RUnlock()
	if len(c.results) == 0 {
		return ""
	}
	data, err := json.Marshal(c.results)
	if err != nil {
		c.Logger().Error("序列化任务结果失败", elog.FieldErr(err))
		return ""
	}
	return string(data)
}
