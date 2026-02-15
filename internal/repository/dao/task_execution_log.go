package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TaskExecutionLog 任务执行日志DAO对象
type TaskExecutionLog struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	ExecutionID int64  `gorm:"index;not null;comment:'任务执行ID'"`
	Content     string `gorm:"type:text;comment:'日志内容'"`
	Ctime       int64  `gorm:"autoCreateTime:milli;comment:'创建时间'"`
}

func (TaskExecutionLog) TableName() string {
	return "task_execution_logs"
}

type TaskExecutionLogDAO interface {
	// Create 创建日志记录
	Create(ctx context.Context, log TaskExecutionLog) error
	// BatchCreate 批量创建日志记录
	BatchCreate(ctx context.Context, logs []TaskExecutionLog) error
	// GetLogsByExecutionID 获取指定执行ID的日志
	// executionID: 执行ID
	// minID: 最小ID(用于增量查询，传0查所有)
	// limit: 限制条数
	GetLogsByExecutionID(ctx context.Context, executionID int64, minID int64, limit int) ([]TaskExecutionLog, error)
}

type GORMTaskExecutionLogDAO struct {
	db *gorm.DB
}

func NewGORMTaskExecutionLogDAO(db *gorm.DB) TaskExecutionLogDAO {
	return &GORMTaskExecutionLogDAO{db: db}
}

func (g *GORMTaskExecutionLogDAO) Create(ctx context.Context, log TaskExecutionLog) error {
	log.Ctime = time.Now().UnixMilli()
	return g.db.WithContext(ctx).Create(&log).Error
}

func (g *GORMTaskExecutionLogDAO) BatchCreate(ctx context.Context, logs []TaskExecutionLog) error {
	now := time.Now().UnixMilli()
	for i := range logs {
		logs[i].Ctime = now
	}
	// 批量插入
	return g.db.WithContext(ctx).CreateInBatches(logs, len(logs)).Error
}

func (g *GORMTaskExecutionLogDAO) GetLogsByExecutionID(ctx context.Context, executionID int64, minID int64, limit int) ([]TaskExecutionLog, error) {
	var logs []TaskExecutionLog
	err := g.db.WithContext(ctx).
		Where("execution_id = ? AND id > ?", executionID, minID).
		Order("id ASC").
		Limit(limit).
		Find(&logs).Error

	if err != nil {
		return nil, fmt.Errorf("查询日志失败: %w", err)
	}
	return logs, nil
}
