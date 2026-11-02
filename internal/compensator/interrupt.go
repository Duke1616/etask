package compensator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Duke1616/eiam/pkg/ctxutil"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/service/task"
	"github.com/Duke1616/etask/internal/service/termination"
	"github.com/gotomicro/ego/core/elog"
)

// InterruptConfig 中断补偿器配置
type InterruptConfig struct {
	BatchSize   int           `mapstructure:"batch_size" yaml:"batch_size"`     // 批次大小
	MinDuration time.Duration `mapstructure:"min_duration" yaml:"min_duration"` // 最小等待时间，防止空转
}

// InterruptCompensator 中断补偿器，负责扫描超时执行记录并委托统一终止服务推进终态与投递信号。
type InterruptCompensator struct {
	terminationSvc termination.Service
	execSvc        task.ExecutionService
	config         InterruptConfig
	logger         *elog.Component
}

// NewInterruptCompensator 创建中断补偿器
func NewInterruptCompensator(
	terminationSvc termination.Service,
	execSvc task.ExecutionService,
	config InterruptConfig,
) *InterruptCompensator {
	return &InterruptCompensator{
		terminationSvc: terminationSvc,
		execSvc:        execSvc,
		config:         config,
		logger:         elog.DefaultLogger.With(elog.FieldComponentName("compensator.interrupt")),
	}
}

// Start 启动补偿器
func (t *InterruptCompensator) Start(ctx context.Context) {
	t.logger.Info("中断补偿器启动")

	for {
		select {
		case <-ctx.Done():
			t.logger.Info("中断补偿器停止")
			return
		default:
			startTime := time.Now()

			err := t.interruptTimeoutTasks(ctx)
			if err != nil {
				t.logger.Error("中断超时任务失败", elog.FieldErr(err))
			}

			// 防空转：确保最小等待时间
			elapsed := time.Since(startTime)
			if elapsed < t.config.MinDuration {
				select {
				case <-ctx.Done():
					return
				case <-time.After(t.config.MinDuration - elapsed):
				}
			}
		}
	}
}

// interruptTimeoutTasks 中断超时任务
func (t *InterruptCompensator) interruptTimeoutTasks(ctx context.Context) error {
	// 查找超时的执行记录
	executions, err := t.execSvc.FindTimeoutExecutions(ctx, t.config.BatchSize)
	if err != nil {
		return fmt.Errorf("查找可中断任务失败: %w", err)
	}

	if len(executions) == 0 {
		t.logger.Debug("没有找到可中断的任务")
		return nil
	}

	t.logger.Info("找到可中断任务", elog.Int("count", len(executions)))

	// 处理每个超时的执行：通过统一终止服务发起终止意图
	for i := range executions {
		execution := executions[i]
		err = t.interruptTaskExecution(ctx, execution)
		if err != nil {
			t.logger.Error("中断超时任务失败",
				elog.Int64("executionId", execution.ID),
				elog.String("taskName", execution.Task.Name),
				elog.FieldErr(err))
			continue
		}
		t.logger.Info("成功触发超时任务中断",
			elog.Int64("executionId", execution.ID),
			elog.String("taskName", execution.Task.Name))
	}
	return nil
}

func (t *InterruptCompensator) interruptTaskExecution(ctx context.Context, execution domain.TaskExecution) error {
	taskCtx := ctxutil.WithTenantID(ctx, execution.TenantID)
	taskCtx = ctxutil.WithOriginTenantID(taskCtx, execution.TenantID)

	err := t.terminationSvc.RequestExecution(taskCtx, execution.ID, "任务执行超时")
	if err != nil {
		// 若任务已处于终态或已有终止请求处理中，视为正常收敛跳过
		if errors.Is(err, termination.ErrRejected) {
			t.logger.Debug("超时任务已收敛或已被终止，跳过重复处理",
				elog.Int64("executionId", execution.ID))
			return nil
		}
		return fmt.Errorf("提交超时终止意图失败: %w", err)
	}
	return nil
}
