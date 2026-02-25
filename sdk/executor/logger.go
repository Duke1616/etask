package executor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	executorv1 "github.com/Duke1616/ework-runner/api/proto/gen/etask/executor/v1"
	reporterv1 "github.com/Duke1616/ework-runner/api/proto/gen/etask/reporter/v1"
	"github.com/gotomicro/ego/core/elog"
)

const (
	defaultBufferSize  = 10
	defaultLogInterval = 3 * time.Second
)

// TaskLogger 任务日志记录器接口
type TaskLogger interface {
	Log(format string, args ...any)
	Close()
}

// bufferTaskLogger 带缓冲和脱敏功能的日志记录器
type bufferTaskLogger struct {
	executionID int64
	reporter    reporterv1.ReporterServiceClient
	sysLogger   *elog.Component

	masks      []string      // 敏感词掩码列表
	buffer     []string      // 日志缓冲
	bufferLock sync.Mutex    // 缓冲锁
	bufferSize int           // 缓冲区阈值
	ticker     *time.Ticker  // 定时触发器
	done       chan struct{} // 关闭信号
}

func newTaskLogger(executionID int64, reporter reporterv1.ReporterServiceClient, sysLogger *elog.Component, masks []string) TaskLogger {
	l := &bufferTaskLogger{
		executionID: executionID,
		reporter:    reporter,
		sysLogger:   sysLogger,
		masks:       masks,
		buffer:      make([]string, 0, defaultBufferSize),
		bufferSize:  defaultBufferSize,
		ticker:      time.NewTicker(defaultLogInterval),
		done:        make(chan struct{}),
	}
	go l.loop()
	return l
}

func (l *bufferTaskLogger) Log(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)

	// 敏感信息脱敏
	for _, mask := range l.masks {
		msg = strings.ReplaceAll(msg, mask, "[MASKED]")
	}

	l.bufferLock.Lock()
	l.buffer = append(l.buffer, msg)
	shouldFlush := len(l.buffer) >= l.bufferSize
	l.bufferLock.Unlock()

	if shouldFlush {
		// 异步flush，避免阻塞Log调用方
		// 注意：这里简单的go flush可能会导致并发flush，但flush内部有锁，且我们只关注最终发送
		// 更严谨的做法是发送信号给loop，但为了简单起见，这里直接调用flush
		// 或者 better: 只在loop中flush，这里通过channel传递日志？
		// 考虑到之前的实现逻辑，这里保持异步flush
		go l.flush()
	}
}

func (l *bufferTaskLogger) Close() {
	l.ticker.Stop()
	close(l.done)
	l.flush() // 发送剩余日志
}

func (l *bufferTaskLogger) loop() {
	for {
		select {
		case <-l.ticker.C:
			l.flush()
		case <-l.done:
			return
		}
	}
}

func (l *bufferTaskLogger) flush() {
	l.bufferLock.Lock()
	if len(l.buffer) == 0 {
		l.bufferLock.Unlock()
		return
	}

	// 交换缓冲区
	logs := l.buffer
	l.buffer = make([]string, 0, l.bufferSize)
	l.bufferLock.Unlock()

	// LogOnly=true 表示"仅上传日志"，调度节点收到后只保存日志，跳过状态机处理。
	// 这样后台定时 flush goroutine 不会干扰主流程的状态迁移。
	_, err := l.reporter.Report(context.Background(), &reporterv1.ReportRequest{
		ExecutionState: &executorv1.ExecutionState{
			Id: l.executionID,
		},
		LogChunks: logs,
		LogOnly:   true,
	})
	if err != nil {
		l.sysLogger.Error("report logs failed", elog.FieldErr(err))
	}
}
