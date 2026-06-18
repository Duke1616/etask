package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Codebook 映射脚本模板持久化表。
type Codebook struct {
	ID         int64  `gorm:"primaryKey;column:id;type:bigint;autoIncrement;comment:'脚本模板自增ID'"`
	TenantID   int64  `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;index;comment:'租户ID'"`
	Name       string `gorm:"column:name;type:varchar(128);not null;comment:'脚本模板名称'"`
	Owner      string `gorm:"column:owner;type:varchar(128);not null;default:'';comment:'模板所有者'"`
	Identifier string `gorm:"column:identifier;type:varchar(64);not null;uniqueIndex:uniq_codebook_identifier_tenant;comment:'脚本唯一标识码'"`
	Code       string `gorm:"column:code;type:text;comment:'脚本源码快照内容'"`
	Language   string `gorm:"column:language;type:varchar(32);comment:'脚本语言'"`
	Secret     string `gorm:"column:secret;type:varchar(128);comment:'脚本访问密钥'"`
	CTime      int64  `gorm:"column:ctime;type:bigint;comment:'创建时间(毫秒)'"`
	UTime      int64  `gorm:"column:utime;type:bigint;comment:'更新时间(毫秒)'"`
}

// CodebookDAO 定义脚本模板的数据访问操作。
type CodebookDAO interface {
	// Create 插入一条脚本模板并返回生成的主键 ID。
	Create(ctx context.Context, c Codebook) (int64, error)
	// GetByID 根据主键 ID 查询脚本模板。
	GetByID(ctx context.Context, id int64) (Codebook, error)
	// GetByIdentifier 根据业务唯一标识查询脚本模板。
	GetByIdentifier(ctx context.Context, identifier string) (Codebook, error)
	// List 分页查询脚本模板，按创建时间倒序返回。
	List(ctx context.Context, offset, limit int64) ([]Codebook, error)
	// Count 统计脚本模板总数。
	Count(ctx context.Context) (int64, error)
	// Update 更新脚本模板可变字段。
	Update(ctx context.Context, c Codebook) (int64, error)
	// Delete 根据主键 ID 删除脚本模板。
	Delete(ctx context.Context, id int64) (int64, error)
}

// GORMCodebookDAO 基于 GORM 实现 CodebookDAO。
type GORMCodebookDAO struct {
	db *gorm.DB
}

// NewGORMCodebookDAO 创建 GORM 版 CodebookDAO。
func NewGORMCodebookDAO(db *gorm.DB) CodebookDAO {
	return &GORMCodebookDAO{db: db}
}

// Create 插入一条脚本模板并返回生成的主键 ID。
func (g *GORMCodebookDAO) Create(ctx context.Context, c Codebook) (int64, error) {
	now := time.Now().UnixMilli()
	c.CTime, c.UTime = now, now
	err := g.db.WithContext(ctx).Create(&c).Error
	return c.ID, err
}

// GetByID 根据主键 ID 查询脚本模板。
func (g *GORMCodebookDAO) GetByID(ctx context.Context, id int64) (Codebook, error) {
	var res Codebook
	err := g.db.WithContext(ctx).Where("id = ?", id).First(&res).Error
	return res, err
}

// GetByIdentifier 根据业务唯一标识查询脚本模板。
func (g *GORMCodebookDAO) GetByIdentifier(ctx context.Context, identifier string) (Codebook, error) {
	var res Codebook
	err := g.db.WithContext(ctx).Where("identifier = ?", identifier).First(&res).Error
	return res, err
}

// List 分页查询脚本模板，按创建时间倒序返回。
func (g *GORMCodebookDAO) List(ctx context.Context, offset, limit int64) ([]Codebook, error) {
	var res []Codebook
	err := g.db.WithContext(ctx).
		Order("ctime DESC").
		Offset(int(offset)).
		Limit(int(limit)).
		Find(&res).Error
	return res, err
}

// Count 统计脚本模板总数。
func (g *GORMCodebookDAO) Count(ctx context.Context) (int64, error) {
	var count int64
	err := g.db.WithContext(ctx).Model(&Codebook{}).Count(&count).Error
	return count, err
}

// Update 更新脚本模板可变字段。
func (g *GORMCodebookDAO) Update(ctx context.Context, c Codebook) (int64, error) {
	res := g.db.WithContext(ctx).
		Model(&Codebook{}).
		Where("id = ?", c.ID).
		Updates(map[string]any{
			"name":     c.Name,
			"owner":    c.Owner,
			"code":     c.Code,
			"language": c.Language,
			"utime":    time.Now().UnixMilli(),
		})
	return res.RowsAffected, res.Error
}

// Delete 根据主键 ID 删除脚本模板。
func (g *GORMCodebookDAO) Delete(ctx context.Context, id int64) (int64, error) {
	res := g.db.WithContext(ctx).Where("id = ?", id).Delete(&Codebook{})
	return res.RowsAffected, res.Error
}
