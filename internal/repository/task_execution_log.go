package repository

import (
	"context"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/repository/dao"
	"github.com/ecodeclub/ekit/slice"
)

type TaskExecutionLogRepository interface {
	// Create 创建日志记录
	Create(ctx context.Context, log domain.TaskExecutionLog) error
	// BatchCreate 批量创建日志记录
	BatchCreate(ctx context.Context, logs []domain.TaskExecutionLog) error
	// GetLogsByExecutionID 获取指定执行ID的日志
	GetLogsByExecutionID(ctx context.Context, executionID int64, minID int64, limit int) ([]domain.TaskExecutionLog, error)
	// GetLogsByExecutionIDs 批量获取执行ID的日志
	GetLogsByExecutionIDs(ctx context.Context, executionIDs []int64) ([]domain.TaskExecutionLog, error)
	// CountByExecutionID 获取日志总数
	CountByExecutionID(ctx context.Context, executionID int64) (int64, error)
}

type taskExecutionLogRepository struct {
	dao dao.TaskExecutionLogDAO
}

func NewTaskExecutionLogRepository(dao dao.TaskExecutionLogDAO) TaskExecutionLogRepository {
	return &taskExecutionLogRepository{
		dao: dao,
	}
}

func (r *taskExecutionLogRepository) Create(ctx context.Context, log domain.TaskExecutionLog) error {
	return r.dao.Create(ctx, r.toEntity(log))
}

func (r *taskExecutionLogRepository) BatchCreate(ctx context.Context, logs []domain.TaskExecutionLog) error {
	return r.dao.BatchCreate(ctx, slice.Map(logs, func(_ int, src domain.TaskExecutionLog) dao.TaskExecutionLog {
		return r.toEntity(src)
	}))
}

func (r *taskExecutionLogRepository) GetLogsByExecutionID(ctx context.Context, executionID int64, minID int64, limit int) ([]domain.TaskExecutionLog, error) {
	daoLogs, err := r.dao.GetLogsByExecutionID(ctx, executionID, minID, limit)
	if err != nil {
		return nil, err
	}
	return slice.Map(daoLogs, func(_ int, src dao.TaskExecutionLog) domain.TaskExecutionLog {
		return r.toDomain(src)
	}), nil
}

func (r *taskExecutionLogRepository) GetLogsByExecutionIDs(ctx context.Context, executionIDs []int64) ([]domain.TaskExecutionLog, error) {
	daoLogs, err := r.dao.GetLogsByExecutionIDs(ctx, executionIDs)
	if err != nil {
		return nil, err
	}
	return slice.Map(daoLogs, func(_ int, src dao.TaskExecutionLog) domain.TaskExecutionLog {
		return r.toDomain(src)
	}), nil
}

func (r *taskExecutionLogRepository) CountByExecutionID(ctx context.Context, executionID int64) (int64, error) {
	return r.dao.CountLogsByExecutionID(ctx, executionID)
}

func (r *taskExecutionLogRepository) toEntity(log domain.TaskExecutionLog) dao.TaskExecutionLog {
	return dao.TaskExecutionLog{
		ID:          log.ID,
		TaskID:      log.TaskID,
		ExecutionID: log.ExecutionID,
		Content:     log.Content,
		Ctime:       log.CTime,
	}
}

func (r *taskExecutionLogRepository) toDomain(log dao.TaskExecutionLog) domain.TaskExecutionLog {
	return domain.TaskExecutionLog{
		ID:          log.ID,
		TaskID:      log.TaskID,
		ExecutionID: log.ExecutionID,
		Content:     log.Content,
		CTime:       log.Ctime,
	}
}
