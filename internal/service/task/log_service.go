package task

import (
	"context"

	"github.com/Duke1616/ework-runner/internal/domain"
	"github.com/Duke1616/ework-runner/internal/repository/dao"
)

// LogService 任务日志服务接口
type LogService interface {
	// AddLog 添加任务日志
	AddLog(ctx context.Context, log domain.TaskExecutionLog) error
	// BatchAddLogs 批量添加日志
	BatchAddLogs(ctx context.Context, logs []domain.TaskExecutionLog) error
	// GetLogs 获取任务日志
	GetLogs(ctx context.Context, executionID int64, minID int64, limit int) ([]domain.TaskExecutionLog, error)
}

type logService struct {
	dao dao.TaskExecutionLogDAO
}

func NewLogService(dao dao.TaskExecutionLogDAO) LogService {
	return &logService{
		dao: dao,
	}
}

func (s *logService) AddLog(ctx context.Context, log domain.TaskExecutionLog) error {
	return s.dao.Create(ctx, s.toDAO(log))
}

func (s *logService) BatchAddLogs(ctx context.Context, logs []domain.TaskExecutionLog) error {
	if len(logs) == 0 {
		return nil
	}
	daoLogs := make([]dao.TaskExecutionLog, len(logs))
	for i, log := range logs {
		daoLogs[i] = s.toDAO(log)
	}
	return s.dao.BatchCreate(ctx, daoLogs)
}

func (s *logService) GetLogs(ctx context.Context, executionID int64, minID int64, limit int) ([]domain.TaskExecutionLog, error) {
	daoLogs, err := s.dao.GetLogsByExecutionID(ctx, executionID, minID, limit)
	if err != nil {
		return nil, err
	}
	logs := make([]domain.TaskExecutionLog, len(daoLogs))
	for i, log := range daoLogs {
		logs[i] = s.toDomain(log)
	}
	return logs, nil
}

func (s *logService) toDAO(log domain.TaskExecutionLog) dao.TaskExecutionLog {
	return dao.TaskExecutionLog{
		ID:          log.ID,
		ExecutionID: log.ExecutionID,
		Content:     log.Content,
		Ctime:       log.CTime,
	}
}

func (s *logService) toDomain(log dao.TaskExecutionLog) domain.TaskExecutionLog {
	return domain.TaskExecutionLog{
		ID:          log.ID,
		ExecutionID: log.ExecutionID,
		Content:     log.Content,
		CTime:       log.Ctime,
	}
}
