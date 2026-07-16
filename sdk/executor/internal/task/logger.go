package task

// 本文件实现任务日志缓冲、脱敏和上报。

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	executorv1 "github.com/Duke1616/etask/api/proto/gen/etask/executor/v1"
	reporterv1 "github.com/Duke1616/etask/api/proto/gen/etask/reporter/v1"
	"github.com/gotomicro/ego/core/elog"
)

const (
	defaultBufferSize  = 10
	defaultLogInterval = 3 * time.Second
)

// TaskLogger 任务日志记录器接口
type TaskLogger interface {
	// Log 记录一条支持格式化参数的任务日志。
	Log(format string, args ...any)
	// Close 刷新剩余日志并释放后台资源。
	Close()
}

type maskingTaskLogger struct {
	next  TaskLogger
	masks []string
}

func newMaskingTaskLogger(next TaskLogger, masks []string) TaskLogger {
	if len(masks) == 0 {
		return next
	}
	return &maskingTaskLogger{next: next, masks: masks}
}

func (l *maskingTaskLogger) Log(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	for _, mask := range l.masks {
		message = strings.ReplaceAll(message, mask, "[MASKED]")
	}
	l.next.Log("%s", message)
}

func (l *maskingTaskLogger) Close() {
	l.next.Close()
}

// bufferTaskLogger 带缓冲和脱敏功能的日志记录器
type bufferTaskLogger struct {
	ctx         context.Context // 任务执行的原生上下文，承载多租户信息
	executionID int64
	reporter    reporterv1.ReporterServiceClient
	sysLogger   *elog.Component

	masks      []string
	buffer     []string
	bufferLock sync.Mutex
	bufferSize int
	closed     bool
	ticker     *time.Ticker
	flushCh    chan struct{}
	done       chan struct{}
	closeOnce  sync.Once
	wg         sync.WaitGroup
}

func newTaskLogger(ctx context.Context, executionID int64, reporter reporterv1.ReporterServiceClient, sysLogger *elog.Component, masks []string) TaskLogger {
	l := &bufferTaskLogger{
		ctx:         ctx,
		executionID: executionID,
		reporter:    reporter,
		sysLogger:   sysLogger,
		masks:       masks,
		buffer:      make([]string, 0, defaultBufferSize),
		bufferSize:  defaultBufferSize,
		ticker:      time.NewTicker(defaultLogInterval),
		flushCh:     make(chan struct{}, 1),
		done:        make(chan struct{}),
	}
	// 单独协程统一处理定时和容量触发的刷新，Log 调用不执行网络请求。
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		l.loop()
	}()
	return l
}

func (l *bufferTaskLogger) Log(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)

	for _, mask := range l.masks {
		msg = strings.ReplaceAll(msg, mask, "[MASKED]")
	}

	l.bufferLock.Lock()
	if l.closed {
		l.bufferLock.Unlock()
		return
	}
	l.buffer = append(l.buffer, msg)
	shouldFlush := len(l.buffer) >= l.bufferSize
	l.bufferLock.Unlock()

	if shouldFlush {
		// 非阻塞信号只表达“需要刷新”，多个并发信号可以合并。
		select {
		case l.flushCh <- struct{}{}:
		default:
		}
	}
}

func (l *bufferTaskLogger) Close() {
	l.closeOnce.Do(func() {
		// 先禁止继续写入，再通知后台协程完成最后一次刷新。
		l.bufferLock.Lock()
		l.closed = true
		l.bufferLock.Unlock()
		close(l.done)
		l.wg.Wait()
	})
}

func (l *bufferTaskLogger) loop() {
	defer l.ticker.Stop()
	for {
		select {
		case <-l.ticker.C:
			l.flush()
		case <-l.flushCh:
			l.flush()
		case <-l.done:
			l.flush()
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

	// 锁内只交换缓冲区，实际 gRPC 上报在锁外进行。
	logs := l.buffer
	l.buffer = make([]string, 0, l.bufferSize)
	l.bufferLock.Unlock()

	if l.reporter == nil {
		return
	}
	baseCtx := l.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	// 日志上报不继承任务取消信号，保证结束阶段的剩余日志仍可发送。
	reportCtx, cancel := context.WithTimeout(context.WithoutCancel(baseCtx), 10*time.Second)
	defer cancel()

	_, err := l.reporter.Report(reportCtx, &reporterv1.ReportRequest{
		ExecutionState: &executorv1.ExecutionState{
			Id: l.executionID,
		},
		LogChunks: logs,
		LogOnly:   true,
	})
	if err != nil && l.sysLogger != nil {
		l.sysLogger.Error("上报任务日志失败", elog.FieldErr(err))
	}
}
