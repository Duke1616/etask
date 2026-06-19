package dao

import (
	"context"
	"time"

	"github.com/Duke1616/etask/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Variable struct {
	ID       int64  `gorm:"primaryKey;column:id;type:bigint;autoIncrement;comment:'变量自增ID'"`
	TenantID int64  `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;uniqueIndex:idx_variables_scope_target_key,priority:1;comment:'租户ID'"`
	Scope    string `gorm:"column:scope;type:varchar(32);not null;uniqueIndex:idx_variables_scope_target_key,priority:2;comment:'作用域 GLOBAL/RUNNER'"`
	TargetID int64  `gorm:"column:target_id;type:bigint;not null;default:0;uniqueIndex:idx_variables_scope_target_key,priority:3;comment:'作用域目标ID，GLOBAL为0，RUNNER为runner_id'"`
	Key      string `gorm:"column:key;type:varchar(128);not null;uniqueIndex:idx_variables_scope_target_key,priority:4;comment:'变量名'"`
	Value    string `gorm:"column:value;type:text;comment:'变量值'"`
	Secret   bool   `gorm:"column:secret;type:boolean;not null;default:false;comment:'是否敏感变量'"`
	CTime    int64  `gorm:"column:ctime;type:bigint;comment:'创建时间(毫秒)'"`
	UTime    int64  `gorm:"column:utime;type:bigint;comment:'更新时间(毫秒)'"`
}

func (Variable) TableName() string {
	return "variables"
}

// VariableDAO 定义变量表的数据访问操作。
type VariableDAO interface {
	// Create 创建变量并返回主键 ID。
	Create(ctx context.Context, variable Variable) (int64, error)
	// FindByID 根据主键 ID 查询变量。
	FindByID(ctx context.Context, id int64) (Variable, error)
	// Update 根据主键 ID 更新变量可变字段。
	Update(ctx context.Context, variable Variable) (int64, error)
	// DeleteByIDAndScope 根据主键 ID、作用域和目标 ID 删除变量。
	DeleteByIDAndScope(ctx context.Context, id int64, scope string, targetID int64) (int64, error)
	// UpsertByScope 按作用域和目标 ID 批量新增或更新变量。
	UpsertByScope(ctx context.Context, scope string, targetID int64, variables []Variable) error
	// DeleteByScope 按作用域和目标 ID 删除变量。
	DeleteByScope(ctx context.Context, scope string, targetID int64) error
	// ListByScope 按作用域和目标 ID 查询变量列表。
	ListByScope(ctx context.Context, scope string, targetID int64) ([]Variable, error)
	// ListByScopePage 按作用域和目标 ID 分页查询变量列表。
	ListByScopePage(ctx context.Context, scope string, targetID int64, offset, limit int64, keyword string) ([]Variable, error)
	// CountByScope 按作用域和目标 ID 统计变量数量。
	CountByScope(ctx context.Context, scope string, targetID int64, keyword string) (int64, error)
	// ListGlobalAndRunner 查询全局变量和指定执行单元私有变量。
	ListGlobalAndRunner(ctx context.Context, runnerID int64) ([]Variable, error)
}

type GORMVariableDAO struct {
	db *gorm.DB
}

func NewGORMVariableDAO(db *gorm.DB) VariableDAO {
	return &GORMVariableDAO{db: db}
}

// Create 创建变量并返回主键 ID。
func (g *GORMVariableDAO) Create(ctx context.Context, variable Variable) (int64, error) {
	now := time.Now().UnixMilli()
	variable.CTime, variable.UTime = now, now
	err := g.db.WithContext(ctx).Create(&variable).Error
	return variable.ID, err
}

// FindByID 根据主键 ID 查询变量。
func (g *GORMVariableDAO) FindByID(ctx context.Context, id int64) (Variable, error) {
	var res Variable
	err := g.db.WithContext(ctx).Where("id = ?", id).First(&res).Error
	return res, err
}

// Update 根据主键 ID 更新变量可变字段。
func (g *GORMVariableDAO) Update(ctx context.Context, variable Variable) (int64, error) {
	res := g.db.WithContext(ctx).
		Model(&Variable{}).
		Where("id = ? AND scope = ? AND target_id = ?", variable.ID, variable.Scope, variable.TargetID).
		Updates(map[string]any{
			"key":    variable.Key,
			"value":  variable.Value,
			"secret": variable.Secret,
			"utime":  time.Now().UnixMilli(),
		})
	return res.RowsAffected, res.Error
}

// DeleteByIDAndScope 根据主键 ID、作用域和目标 ID 删除变量。
func (g *GORMVariableDAO) DeleteByIDAndScope(ctx context.Context, id int64, scope string, targetID int64) (int64, error) {
	res := g.db.WithContext(ctx).
		Where("id = ? AND scope = ? AND target_id = ?", id, scope, targetID).
		Delete(&Variable{})
	return res.RowsAffected, res.Error
}

// UpsertByScope 按作用域和目标 ID 批量新增或更新变量。
func (g *GORMVariableDAO) UpsertByScope(ctx context.Context, scope string, targetID int64, variables []Variable) error {
	return upsertVariablesByScope(g.db.WithContext(ctx), scope, targetID, variables)
}

func upsertVariablesByScope(tx *gorm.DB, scope string, targetID int64, variables []Variable) error {
	if len(variables) == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	for i := range variables {
		variables[i].Scope = scope
		variables[i].TargetID = targetID
		variables[i].CTime = now
		variables[i].UTime = now
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"},
			{Name: "scope"},
			{Name: "target_id"},
			{Name: "key"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"value", "secret", "utime"}),
	}).Create(&variables).Error
}

func replaceVariablesByScope(tx *gorm.DB, scope string, targetID int64, variables []Variable) error {
	if err := tx.Where("scope = ? AND target_id = ?", scope, targetID).Delete(&Variable{}).Error; err != nil {
		return err
	}
	return upsertVariablesByScope(tx, scope, targetID, variables)
}

// DeleteByScope 按作用域和目标 ID 删除变量。
func (g *GORMVariableDAO) DeleteByScope(ctx context.Context, scope string, targetID int64) error {
	return g.db.WithContext(ctx).Where("scope = ? AND target_id = ?", scope, targetID).Delete(&Variable{}).Error
}

// ListByScope 按作用域和目标 ID 查询变量列表。
func (g *GORMVariableDAO) ListByScope(ctx context.Context, scope string, targetID int64) ([]Variable, error) {
	return g.ListByScopePage(ctx, scope, targetID, 0, 0, "")
}

// ListByScopePage 按作用域和目标 ID 分页查询变量列表。
func (g *GORMVariableDAO) ListByScopePage(ctx context.Context, scope string, targetID int64, offset, limit int64, keyword string) ([]Variable, error) {
	var res []Variable
	query := g.buildScopeQuery(ctx, scope, targetID, keyword).Order("ctime ASC, id ASC")
	if offset > 0 {
		query = query.Offset(int(offset))
	}
	if limit > 0 {
		query = query.Limit(int(limit))
	}
	err := query.Find(&res).Error
	return res, err
}

// CountByScope 按作用域和目标 ID 统计变量数量。
func (g *GORMVariableDAO) CountByScope(ctx context.Context, scope string, targetID int64, keyword string) (int64, error) {
	var count int64
	err := g.buildScopeQuery(ctx, scope, targetID, keyword).Count(&count).Error
	return count, err
}

func (g *GORMVariableDAO) buildScopeQuery(ctx context.Context, scope string, targetID int64, keyword string) *gorm.DB {
	query := g.db.WithContext(ctx).
		Model(&Variable{}).
		Where("scope = ? AND target_id = ?", scope, targetID)
	if keyword != "" {
		query = query.Where("`key` LIKE ?", "%"+keyword+"%")
	}
	return query
}

// ListGlobalAndRunner 查询全局变量和指定执行单元私有变量。
func (g *GORMVariableDAO) ListGlobalAndRunner(ctx context.Context, runnerID int64) ([]Variable, error) {
	var res []Variable
	err := g.db.WithContext(ctx).
		Where(
			"(scope = ? AND target_id = 0) OR (scope = ? AND target_id = ?)",
			domain.VariableScopeGlobal.String(),
			domain.VariableScopeRunner.String(),
			runnerID,
		).
		Order("CASE WHEN scope = 'GLOBAL' THEN 0 WHEN scope = 'RUNNER' THEN 1 ELSE 2 END, ctime ASC, id ASC").
		Find(&res).Error
	return res, err
}
