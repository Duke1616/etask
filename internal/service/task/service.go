package task

import (
	"context"
	"fmt"
	"time"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/errs"
	"github.com/Duke1616/etask/internal/repository"
	"golang.org/x/sync/errgroup"
)

// Service 任务服务接口
type Service interface {
	// Create 创建任务
	Create(ctx context.Context, task domain.Task) (domain.Task, error)
	// SchedulableTasks 获取可调度的任务列表，preemptedTimeoutMs 表示处于 PREEMPTED 状态任务的超时时间（毫秒）
	SchedulableTasks(ctx context.Context, preemptedTimeoutMs int64, limit int) ([]domain.Task, error)
	// UpdateNextTime 更新任务的下次执行时间
	UpdateNextTime(ctx context.Context, id int64) (domain.Task, error)
	// GetByID 根据ID获取task
	GetByID(ctx context.Context, id int64) (domain.Task, error)
	// GetByName 根据名称获取task
	GetByName(ctx context.Context, name string) (domain.Task, error)
	// UpdateScheduleParams 更新调度参数
	UpdateScheduleParams(ctx context.Context, task domain.Task, params map[string]string) (domain.Task, error)
	// RetryByID 根据 ID 重试任务
	RetryByID(ctx context.Context, id int64) (domain.Task, error)
	// RetryByName 根据名称重试任务
	RetryByName(ctx context.Context, name string) (domain.Task, error)
	// List 分页获取任务列表
	List(ctx context.Context, bizID int64, offset, limit int) ([]domain.Task, int64, error)
	// Update 更新任务配置
	Update(ctx context.Context, task domain.Task) error
	// Delete 删除任务
	Delete(ctx context.Context, id int64) error
	// Stop 停止任务
	Stop(ctx context.Context, id int64) error
	// Run 运行任务（从停止状态恢复）
	Run(ctx context.Context, id int64) error
}

type service struct {
	repo repository.TaskRepository
}

// NewService 创建任务服务实例
func NewService(repo repository.TaskRepository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) Create(ctx context.Context, task domain.Task) (domain.Task, error) {
	// 计算并设置下次执行时间
	nextTime, err := task.CalculateNextTime()
	if err != nil {
		return domain.Task{}, fmt.Errorf("%w: %w", errs.ErrInvalidTaskCronExpr, err)
	}
	if nextTime.IsZero() {
		return domain.Task{}, errs.ErrInvalidTaskCronExpr
	}
	task.NextTime = nextTime.UnixMilli()
	return s.repo.Create(ctx, task)
}

func (s *service) SchedulableTasks(ctx context.Context, preemptedTimeoutMs int64, limit int) ([]domain.Task, error) {
	return s.repo.SchedulableTasks(ctx, preemptedTimeoutMs, limit)
}

func (s *service) UpdateNextTime(ctx context.Context, id int64) (domain.Task, error) {
	task, err := s.GetByID(ctx, id)
	if err != nil {
		return domain.Task{}, err
	}

	// 一次性任务：如果 NextTime 在过去，说明已执行完成，直接设置为 COMPLETED
	// 这样可以避免 CalculateNextTime 计算出下一次时间
	if task.Type.IsOneTime() && task.NextTime > 0 && task.NextTime < time.Now().UnixMilli() {
		return s.repo.UpdateStatus(ctx, id, domain.TaskStatusCompleted)
	}

	// 计算下次执行时间
	nextTime, err := task.CalculateNextTime()
	if err != nil {
		return domain.Task{}, fmt.Errorf("%w: %w", errs.ErrInvalidTaskCronExpr, err)
	}

	// 如果下次执行时间为零值
	if nextTime.IsZero() {
		// 一次性任务：已经是 INACTIVE 状态（由上面的判断设置）
		// 定时任务：cron 不再触发，直接返回（保持原状态）
		return task, nil
	}

	// 更新下次执行时间
	task.NextTime = nextTime.UnixMilli()
	return s.repo.UpdateNextTime(ctx, task.ID, task.Version, task.NextTime)
}

func (s *service) GetByID(ctx context.Context, id int64) (domain.Task, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GetByName(ctx context.Context, name string) (domain.Task, error) {
	return s.repo.GetByName(ctx, name)
}

func (s *service) UpdateScheduleParams(ctx context.Context, task domain.Task, params map[string]string) (domain.Task, error) {
	task.UpdateScheduleParams(params)
	return s.repo.UpdateScheduleParams(ctx, task.ID, task.Version, task.ScheduleParams)
}

func (s *service) RetryByID(ctx context.Context, id int64) (domain.Task, error) {
	task, err := s.GetByID(ctx, id)
	if err != nil {
		return domain.Task{}, err
	}

	return s.retry(ctx, task)
}

func (s *service) RetryByName(ctx context.Context, name string) (domain.Task, error) {
	task, err := s.GetByName(ctx, name)
	if err != nil {
		return domain.Task{}, err
	}

	return s.retry(ctx, task)
}

func (s *service) retry(ctx context.Context, task domain.Task) (domain.Task, error) {
	// 运行中的任务不允许重试，防止状态竞争
	if task.Status == domain.TaskStatusPreempted {
		return domain.Task{}, fmt.Errorf("任务正在运行中，请等结束后再重试")
	}

	// 重置为立即执行
	return s.repo.Retry(ctx, task.ID, task.Version, time.Now().UnixMilli())
}

func (s *service) List(ctx context.Context, bizID int64, offset, limit int) ([]domain.Task, int64, error) {
	var (
		tasks []domain.Task
		total int64
	)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var err error
		tasks, err = s.repo.List(ctx, bizID, offset, limit)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = s.repo.Count(ctx, bizID)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (s *service) Update(ctx context.Context, task domain.Task) error {
	// 针对更新配置，可能需要重新计算下次执行时间（且只有在任务是 ACTIVE 状态且非正在运行中才安全）
	// 但通常简单的字段更新（如名称）不影响调度。
	// 如果 CronExpr 变了，则必须重新计算。

	// 这里简化处理，直接调用仓库更新。复杂逻辑应根据业务需求在 domain 层或在此处完善。
	return s.repo.Update(ctx, task)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 判定是否已经停止，如果不是停止状态禁止删除
	// 只有 INACTIVE (手动停止) 和 COMPLETED (一次性任务执行完成) 视为停止状态
	if task.Status != domain.TaskStatusInactive && task.Status != domain.TaskStatusCompleted {
		return fmt.Errorf("只能删除已停止的任务（当前状态: %s），请先停止任务后再试", task.Status)
	}

	return s.repo.Delete(ctx, id)
}

func (s *service) Stop(ctx context.Context, id int64) error {
	_, err := s.repo.UpdateStatus(ctx, id, domain.TaskStatusInactive)
	return err
}

func (s *service) Run(ctx context.Context, id int64) error {
	_, err := s.repo.UpdateStatus(ctx, id, domain.TaskStatusActive)
	return err
}
