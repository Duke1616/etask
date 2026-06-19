package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/pkg/sqlx"
	"gorm.io/gorm"
)

// Runner 映射脚本执行单元持久化表。
type Runner struct {
	ID             int64                     `gorm:"primaryKey;column:id;type:bigint;autoIncrement;comment:'执行单元自增ID'"`
	TenantID       int64                     `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;index;comment:'租户ID'"`
	Name           string                    `gorm:"column:name;type:varchar(128);not null;comment:'执行单元名称'"`
	CodebookUID    string                    `gorm:"column:codebook_uid;type:varchar(64);index;comment:'关联脚本模板UID'"`
	CodebookSecret string                    `gorm:"column:codebook_secret;type:varchar(128);comment:'脚本模板认证密钥'"`
	Kind           string                    `gorm:"column:kind;type:varchar(32);comment:'派发管道协议(KAFKA/GRPC)'"`
	Target         string                    `gorm:"column:target;type:varchar(128);comment:'派发物理目标(Topic/ServiceName)'"`
	Handler        string                    `gorm:"column:handler;type:varchar(128);comment:'执行器承载业务方法'"`
	Tags           sqlx.JSONColumn[[]string] `gorm:"column:tags;type:json;comment:'匹配标签JSON'"`
	Action         uint8                     `gorm:"column:action;type:tinyint unsigned;comment:'活跃动作状态 1:REGISTER 2:UNREGISTER'"`
	Desc           string                    `gorm:"column:desc;type:text;comment:'备注说明'"`
	CTime          int64                     `gorm:"column:ctime;type:bigint;comment:'创建时间(毫秒)'"`
	UTime          int64                     `gorm:"column:utime;type:bigint;comment:'更新时间(毫秒)'"`
}

// RunnerDAO 定义脚本执行单元的数据访问操作。
type RunnerDAO interface {
	// Create 插入执行单元并返回生成的主键 ID。
	Create(ctx context.Context, r Runner) (int64, error)
	// CreateWithVariables 在同一事务中插入执行单元和私有变量。
	CreateWithVariables(ctx context.Context, r Runner, variables []Variable) (int64, error)
	// Update 更新执行单元可变字段。
	Update(ctx context.Context, req Runner) (int64, error)
	// UpdateWithVariables 在同一事务中更新执行单元和私有变量。
	UpdateWithVariables(ctx context.Context, req Runner, variables []Variable) (int64, error)
	// Delete 根据主键 ID 删除执行单元。
	Delete(ctx context.Context, id int64) (int64, error)
	// DeleteWithVariables 在同一事务中删除执行单元和私有变量。
	DeleteWithVariables(ctx context.Context, id int64) (int64, error)
	// FindByID 根据主键 ID 查询执行单元。
	FindByID(ctx context.Context, id int64) (Runner, error)
	// List 分页查询执行单元。
	List(ctx context.Context, offset, limit int64, keyword, kind string) ([]Runner, error)
	// Count 统计匹配条件的执行单元总数。
	Count(ctx context.Context, keyword, kind string) (int64, error)
	// FindByCodebookUIDAndTag 根据脚本模板 UID 和标签查询执行单元。
	FindByCodebookUIDAndTag(ctx context.Context, codebookUID string, tag string) (Runner, error)
	// ListByCodebookUID 查询绑定指定脚本模板 UID 的执行单元。
	ListByCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]Runner, error)
	// CountByCodebookUID 统计绑定指定脚本模板 UID 的执行单元数量。
	CountByCodebookUID(ctx context.Context, codebookUID, keyword, kind string) (int64, error)
	// ListExcludeCodebookUID 查询未绑定指定脚本模板 UID 的执行单元。
	ListExcludeCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]Runner, error)
	// CountExcludeCodebookUID 统计未绑定指定脚本模板 UID 的执行单元数量。
	CountExcludeCodebookUID(ctx context.Context, codebookUID, keyword, kind string) (int64, error)
	// ListByCodebookUIDs 查询绑定任一脚本模板 UID 的执行单元。
	ListByCodebookUIDs(ctx context.Context, codebookUIDs []string) ([]Runner, error)
	// ListByIDs 根据 ID 列表批量查询执行单元。
	ListByIDs(ctx context.Context, ids []int64) ([]Runner, error)
	// AggregateTags 返回全部执行单元用于内存聚合标签。
	AggregateTags(ctx context.Context) ([]Runner, error)
}

// GORMRunnerDAO 基于 GORM 实现 RunnerDAO。
type GORMRunnerDAO struct {
	db *gorm.DB
}

// NewGORMRunnerDAO 创建 GORM 版 RunnerDAO。
func NewGORMRunnerDAO(db *gorm.DB) RunnerDAO {
	return &GORMRunnerDAO{db: db}
}

// Create 插入执行单元并返回生成的主键 ID。
func (g *GORMRunnerDAO) Create(ctx context.Context, r Runner) (int64, error) {
	now := time.Now().UnixMilli()
	r.CTime, r.UTime = now, now
	err := g.db.WithContext(ctx).Create(&r).Error
	return r.ID, err
}

func (g *GORMRunnerDAO) CreateWithVariables(ctx context.Context, r Runner, variables []Variable) (int64, error) {
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()
		r.CTime, r.UTime = now, now
		if err := tx.Create(&r).Error; err != nil {
			return err
		}
		return upsertVariablesByScope(tx, domain.VariableScopeRunner.String(), r.ID, variables)
	})
	return r.ID, err
}

// Update 更新执行单元可变字段。
func (g *GORMRunnerDAO) Update(ctx context.Context, req Runner) (int64, error) {
	res := g.db.WithContext(ctx).
		Model(&Runner{}).
		Where("id = ?", req.ID).
		Updates(map[string]any{
			"name":            req.Name,
			"codebook_uid":    req.CodebookUID,
			"codebook_secret": req.CodebookSecret,
			"kind":            req.Kind,
			"target":          req.Target,
			"handler":         req.Handler,
			"tags":            req.Tags,
			"desc":            req.Desc,
			"utime":           time.Now().UnixMilli(),
		})
	return res.RowsAffected, res.Error
}

func (g *GORMRunnerDAO) UpdateWithVariables(ctx context.Context, req Runner, variables []Variable) (int64, error) {
	var rowsAffected int64
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.
			Model(&Runner{}).
			Where("id = ?", req.ID).
			Updates(map[string]any{
				"name":            req.Name,
				"codebook_uid":    req.CodebookUID,
				"codebook_secret": req.CodebookSecret,
				"kind":            req.Kind,
				"target":          req.Target,
				"handler":         req.Handler,
				"tags":            req.Tags,
				"desc":            req.Desc,
				"utime":           time.Now().UnixMilli(),
			})
		if res.Error != nil {
			return res.Error
		}
		rowsAffected = res.RowsAffected
		return replaceVariablesByScope(tx, domain.VariableScopeRunner.String(), req.ID, variables)
	})
	return rowsAffected, err
}

// Delete 根据主键 ID 删除执行单元。
func (g *GORMRunnerDAO) Delete(ctx context.Context, id int64) (int64, error) {
	res := g.db.WithContext(ctx).Where("id = ?", id).Delete(&Runner{})
	return res.RowsAffected, res.Error
}

func (g *GORMRunnerDAO) DeleteWithVariables(ctx context.Context, id int64) (int64, error) {
	var rowsAffected int64
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ?", id).Delete(&Runner{})
		if res.Error != nil {
			return res.Error
		}
		rowsAffected = res.RowsAffected
		return tx.Where("scope = ? AND target_id = ?", domain.VariableScopeRunner.String(), id).Delete(&Variable{}).Error
	})
	return rowsAffected, err
}

// FindByID 根据主键 ID 查询执行单元。
func (g *GORMRunnerDAO) FindByID(ctx context.Context, id int64) (Runner, error) {
	var res Runner
	err := g.db.WithContext(ctx).Where("id = ?", id).First(&res).Error
	return res, err
}

// List 分页查询执行单元。
func (g *GORMRunnerDAO) List(ctx context.Context, offset, limit int64, keyword, kind string) ([]Runner, error) {
	var res []Runner
	query := g.db.WithContext(ctx)
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if kind != "" {
		query = query.Where("kind = ?", kind)
	}
	err := query.Order("ctime DESC").Offset(int(offset)).Limit(int(limit)).Find(&res).Error
	return res, err
}

// Count 统计匹配条件的执行单元总数。
func (g *GORMRunnerDAO) Count(ctx context.Context, keyword, kind string) (int64, error) {
	var count int64
	query := g.db.WithContext(ctx).Model(&Runner{})
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if kind != "" {
		query = query.Where("kind = ?", kind)
	}
	err := query.Count(&count).Error
	return count, err
}

// FindByCodebookUIDAndTag 根据脚本模板 UID 和标签查询执行单元。
func (g *GORMRunnerDAO) FindByCodebookUIDAndTag(ctx context.Context, codebookUID string, tag string) (Runner, error) {
	var res Runner
	err := g.db.WithContext(ctx).
		Where("codebook_uid = ? AND JSON_CONTAINS(tags, ?)", codebookUID, fmt.Sprintf("%q", tag)).
		First(&res).Error
	return res, err
}

// ListByCodebookUID 查询绑定指定脚本模板 UID 的执行单元。
func (g *GORMRunnerDAO) ListByCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]Runner, error) {
	var res []Runner
	query := g.db.WithContext(ctx).Where("codebook_uid = ?", codebookUID)
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if kind != "" {
		query = query.Where("kind = ?", kind)
	}
	err := query.Order("ctime DESC").Offset(int(offset)).Limit(int(limit)).Find(&res).Error
	return res, err
}

// CountByCodebookUID 统计绑定指定脚本模板 UID 的执行单元数量。
func (g *GORMRunnerDAO) CountByCodebookUID(ctx context.Context, codebookUID, keyword, kind string) (int64, error) {
	var count int64
	query := g.db.WithContext(ctx).Model(&Runner{}).Where("codebook_uid = ?", codebookUID)
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if kind != "" {
		query = query.Where("kind = ?", kind)
	}
	err := query.Count(&count).Error
	return count, err
}

// ListExcludeCodebookUID 查询未绑定指定脚本模板 UID 的执行单元。
func (g *GORMRunnerDAO) ListExcludeCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]Runner, error) {
	var res []Runner
	query := g.db.WithContext(ctx).Where("codebook_uid != ?", codebookUID)
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if kind != "" {
		query = query.Where("kind = ?", kind)
	}
	err := query.Order("ctime DESC").Offset(int(offset)).Limit(int(limit)).Find(&res).Error
	return res, err
}

// CountExcludeCodebookUID 统计未绑定指定脚本模板 UID 的执行单元数量。
func (g *GORMRunnerDAO) CountExcludeCodebookUID(ctx context.Context, codebookUID, keyword, kind string) (int64, error) {
	var count int64
	query := g.db.WithContext(ctx).Model(&Runner{}).Where("codebook_uid != ?", codebookUID)
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if kind != "" {
		query = query.Where("kind = ?", kind)
	}
	err := query.Count(&count).Error
	return count, err
}

// ListByCodebookUIDs 查询绑定任一脚本模板 UID 的执行单元。
func (g *GORMRunnerDAO) ListByCodebookUIDs(ctx context.Context, codebookUIDs []string) ([]Runner, error) {
	var res []Runner
	err := g.db.WithContext(ctx).Where("codebook_uid IN ?", codebookUIDs).Find(&res).Error
	return res, err
}

// ListByIDs 根据 ID 列表批量查询执行单元。
func (g *GORMRunnerDAO) ListByIDs(ctx context.Context, ids []int64) ([]Runner, error) {
	var res []Runner
	err := g.db.WithContext(ctx).Where("id IN ?", ids).Find(&res).Error
	return res, err
}

// AggregateTags 返回全部执行单元用于内存聚合标签。
func (g *GORMRunnerDAO) AggregateTags(ctx context.Context) ([]Runner, error) {
	var res []Runner
	err := g.db.WithContext(ctx).Find(&res).Error
	return res, err
}
