package dao

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Duke1616/eiam/pkg/ctxutil"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const defaultCodebookAuthorUserID int64 = 1

// Codebook 映射代码资源节点表，目录和文件统一建模。
type Codebook struct {
	ID               int64  `gorm:"primaryKey;column:id;type:bigint;autoIncrement;comment:'代码节点自增ID'"`
	TenantID         int64  `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;index;uniqueIndex:uniq_codebook_space_parent_name,priority:1;comment:'租户ID'" eiam:"shared:scope = 'SYSTEM'"`
	Scope            string `gorm:"column:scope;type:varchar(32);not null;default:'TENANT';index;uniqueIndex:uniq_codebook_space_parent_name,priority:2;comment:'作用域 SYSTEM/TENANT'"`
	ProjectID        int64  `gorm:"column:project_id;type:bigint;not null;default:0;index;uniqueIndex:uniq_codebook_space_parent_name,priority:3;comment:'所属项目ID，系统组件库为0'"`
	ParentID         int64  `gorm:"column:parent_id;type:bigint;not null;default:0;index;uniqueIndex:uniq_codebook_space_parent_name,priority:4;comment:'父级节点ID'"`
	PathIDs          string `gorm:"column:path_ids;type:varchar(512);not null;default:'/';index;comment:'祖先路径ID，如 /1/2/'"`
	Depth            int    `gorm:"column:depth;type:int;not null;default:0;comment:'节点深度'"`
	Name             string `gorm:"column:name;type:varchar(128);not null;uniqueIndex:uniq_codebook_space_parent_name,priority:5;comment:'节点名称'"`
	Owner            string `gorm:"column:owner;type:varchar(128);not null;default:'';comment:'模板所有者展示名'"`
	Kind             string `gorm:"column:kind;type:varchar(32);not null;default:'FILE';index;comment:'节点类型 DIRECTORY/FILE'"`
	SortNo           int64  `gorm:"column:sort_no;type:bigint;not null;default:0;comment:'排序号'"`
	Secret           string `gorm:"column:secret;type:varchar(128);comment:'脚本访问密钥'"`
	CurrentVersionID int64  `gorm:"column:current_version_id;type:bigint;not null;default:0;comment:'当前使用版本ID'"`
	CTime            int64  `gorm:"column:ctime;type:bigint;comment:'创建时间(毫秒)'"`
	UTime            int64  `gorm:"column:utime;type:bigint;comment:'更新时间(毫秒)'"`
}

func (Codebook) TableName() string {
	return "codebook"
}

// CodebookProject 映射代码资源项目表。
type CodebookProject struct {
	ID       int64  `gorm:"primaryKey;column:id;type:bigint;autoIncrement;comment:'代码项目自增ID'"`
	TenantID int64  `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;index;uniqueIndex:uniq_codebook_project_tenant_name,priority:1;comment:'租户ID'"`
	Scope    string `gorm:"column:scope;type:varchar(32);not null;default:'TENANT';index;comment:'作用域，项目仅用于TENANT'"`
	Name     string `gorm:"column:name;type:varchar(128);not null;uniqueIndex:uniq_codebook_project_tenant_name,priority:2;comment:'项目名称'"`
	Desc     string `gorm:"column:description;type:varchar(255);not null;default:'';comment:'项目描述'"`
	SortNo   int64  `gorm:"column:sort_no;type:bigint;not null;default:0;comment:'排序号'"`
	Status   string `gorm:"column:status;type:varchar(32);not null;default:'NORMAL';index;comment:'项目状态 NORMAL/ARCHIVED'"`
	CTime    int64  `gorm:"column:ctime;type:bigint;comment:'创建时间(毫秒)'"`
	UTime    int64  `gorm:"column:utime;type:bigint;comment:'更新时间(毫秒)'"`
}

func (CodebookProject) TableName() string {
	return "codebook_project"
}

// CodebookVersion 映射代码资源内容版本表。
type CodebookVersion struct {
	ID           int64  `gorm:"primaryKey;column:id;type:bigint;autoIncrement;comment:'代码版本自增ID'"`
	NodeID       int64  `gorm:"column:node_id;type:bigint;not null;index;uniqueIndex:uniq_codebook_version_no,priority:1;comment:'代码节点ID'"`
	TenantID     int64  `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;index;comment:'租户ID'" eiam:"shared:scope = 'SYSTEM'"`
	Scope        string `gorm:"column:scope;type:varchar(32);not null;default:'TENANT';index;comment:'作用域 SYSTEM/TENANT'"`
	VersionNo    int64  `gorm:"column:version_no;type:bigint;not null;default:1;uniqueIndex:uniq_codebook_version_no,priority:2;comment:'版本号'"`
	Code         string `gorm:"column:code;type:text;comment:'脚本源码内容'"`
	Hash         string `gorm:"column:hash;type:varchar(64);comment:'源码哈希'"`
	Message      string `gorm:"column:message;type:varchar(255);comment:'版本说明'"`
	AuthorUserID int64  `gorm:"column:author_user_id;type:bigint unsigned;not null;default:0;comment:'作者用户ID'"`
	CTime        int64  `gorm:"column:ctime;type:bigint;comment:'创建时间(毫秒)'"`
}

func (CodebookVersion) TableName() string {
	return "codebook_version"
}

// CodebookSortItem 表示代码资源排序更新项。
type CodebookSortItem struct {
	ID        int64
	ProjectID int64
	ParentID  int64
	PathIDs   string
	Depth     int
	SortNo    int64
}

// CodebookProjectDAO 定义代码资源项目的数据访问操作。
type CodebookProjectDAO interface {
	// Create 插入一个代码资源项目。
	Create(ctx context.Context, p CodebookProject) (int64, error)
	// GetByID 根据主键 ID 查询代码资源项目。
	GetByID(ctx context.Context, id int64) (CodebookProject, error)
	// List 分页查询代码资源项目。
	List(ctx context.Context, offset, limit int64) ([]CodebookProject, error)
	// Count 统计代码资源项目总数。
	Count(ctx context.Context) (int64, error)
	// GetMaxSortNo 查询当前租户项目最大的排序号。
	GetMaxSortNo(ctx context.Context, tenantID int64) (int64, error)
	// Update 更新代码资源项目。
	Update(ctx context.Context, p CodebookProject) (int64, error)
	// Delete 归档代码资源项目。
	Delete(ctx context.Context, id int64) (int64, error)
}

// CodebookDAO 定义代码资源的数据访问操作。
type CodebookDAO interface {
	// Create 插入一个代码节点，文件节点会同步创建初始版本。
	Create(ctx context.Context, c Codebook, code string) (int64, error)
	// GetByID 根据主键 ID 查询代码节点。
	GetByID(ctx context.Context, id int64) (Codebook, error)
	// GetCurrentVersion 查询节点当前使用版本。
	GetCurrentVersion(ctx context.Context, nodeID int64) (CodebookVersion, error)
	// ListVersions 查询指定节点的所有版本。
	ListVersions(ctx context.Context, nodeID int64) ([]CodebookVersion, error)
	// ListVersionsByIDs 根据版本 ID 批量查询代码版本。
	ListVersionsByIDs(ctx context.Context, ids []int64) ([]CodebookVersion, error)
	// GetVersionByID 根据主键 ID 查询代码版本。
	GetVersionByID(ctx context.Context, id int64) (CodebookVersion, error)
	// List 分页查询代码节点，按创建时间倒序返回。
	List(ctx context.Context, offset, limit int64) ([]Codebook, error)
	// ListChildren 查询指定项目和父节点下的子节点。
	ListChildren(ctx context.Context, projectID, parentID int64) ([]Codebook, error)
	// ListChildrenBySpace 查询指定空间和父节点下的子节点。
	ListChildrenBySpace(ctx context.Context, projectID, parentID int64, scope string) ([]Codebook, error)
	// Tree 查询指定项目视图下的节点树，系统组件库由租户插件透出。
	Tree(ctx context.Context, projectID int64) ([]Codebook, error)
	// Count 统计代码节点总数。
	Count(ctx context.Context) (int64, error)
	// GetMaxSortNo 查询指定空间和父节点下最大的排序号。
	GetMaxSortNo(ctx context.Context, projectID, parentID int64, scope string) (int64, error)
	// Update 更新代码节点可变字段。
	Update(ctx context.Context, c Codebook, code string) (int64, error)
	// CreateVersion 创建代码版本。
	CreateVersion(ctx context.Context, version CodebookVersion) (int64, error)
	// UseVersion 设置代码节点当前使用版本。
	UseVersion(ctx context.Context, nodeID, versionID int64) (int64, error)
	// UpdateSort 更新单个代码节点的父级、路径和排序号。
	UpdateSort(ctx context.Context, item CodebookSortItem) error
	// BatchUpdateSort 批量更新代码节点排序。
	BatchUpdateSort(ctx context.Context, items []CodebookSortItem) error
	// Delete 根据主键 ID 删除代码节点。
	Delete(ctx context.Context, id int64) (int64, error)
}

// GORMCodebookDAO 基于 GORM 实现 CodebookDAO。
type GORMCodebookDAO struct {
	db *gorm.DB
}

// GORMCodebookProjectDAO 基于 GORM 实现 CodebookProjectDAO。
type GORMCodebookProjectDAO struct {
	db *gorm.DB
}

// NewGORMCodebookDAO 创建 GORM 版 CodebookDAO。
func NewGORMCodebookDAO(db *gorm.DB) CodebookDAO {
	return &GORMCodebookDAO{db: db}
}

// NewGORMCodebookProjectDAO 创建 GORM 版 CodebookProjectDAO。
func NewGORMCodebookProjectDAO(db *gorm.DB) CodebookProjectDAO {
	return &GORMCodebookProjectDAO{db: db}
}

// Create 插入一个代码资源项目。
func (g *GORMCodebookProjectDAO) Create(ctx context.Context, p CodebookProject) (int64, error) {
	now := time.Now().UnixMilli()
	p.CTime, p.UTime = now, now
	if p.Status == "" {
		p.Status = domain.CodebookProjectStatusNormal.String()
	}
	if p.Scope == "" {
		p.Scope = domain.CodebookScopeTenant.String()
	}
	err := g.db.WithContext(ctx).Create(&p).Error
	return p.ID, err
}

// GetByID 根据主键 ID 查询代码资源项目。
func (g *GORMCodebookProjectDAO) GetByID(ctx context.Context, id int64) (CodebookProject, error) {
	var res CodebookProject
	err := g.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, domain.CodebookProjectStatusNormal.String()).
		First(&res).Error
	return res, err
}

// List 分页查询代码资源项目。
func (g *GORMCodebookProjectDAO) List(ctx context.Context, offset, limit int64) ([]CodebookProject, error) {
	var res []CodebookProject
	err := g.db.WithContext(ctx).
		Where("status = ?", domain.CodebookProjectStatusNormal.String()).
		Order("sort_no ASC, id ASC").
		Offset(int(offset)).
		Limit(int(limit)).
		Find(&res).Error
	return res, err
}

// Count 统计代码资源项目总数。
func (g *GORMCodebookProjectDAO) Count(ctx context.Context) (int64, error) {
	var count int64
	err := g.db.WithContext(ctx).
		Model(&CodebookProject{}).
		Where("status = ?", domain.CodebookProjectStatusNormal.String()).
		Count(&count).Error
	return count, err
}

// GetMaxSortNo 查询当前租户项目最大的排序号。
func (g *GORMCodebookProjectDAO) GetMaxSortNo(ctx context.Context, tenantID int64) (int64, error) {
	var sortNo int64
	err := g.db.WithContext(ctx).
		Model(&CodebookProject{}).
		Where("tenant_id = ? AND status = ?", tenantID, domain.CodebookProjectStatusNormal.String()).
		Select("COALESCE(MAX(sort_no), 0)").
		Scan(&sortNo).Error
	return sortNo, err
}

// Update 更新代码资源项目。
func (g *GORMCodebookProjectDAO) Update(ctx context.Context, p CodebookProject) (int64, error) {
	res := g.db.WithContext(ctx).Model(&CodebookProject{}).
		Where("id = ?", p.ID).
		Updates(map[string]any{
			"name":        p.Name,
			"description": p.Desc,
			"sort_no":     p.SortNo,
			"utime":       time.Now().UnixMilli(),
		})
	return res.RowsAffected, res.Error
}

// Delete 归档代码资源项目。
func (g *GORMCodebookProjectDAO) Delete(ctx context.Context, id int64) (int64, error) {
	res := g.db.WithContext(ctx).Model(&CodebookProject{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status": domain.CodebookProjectStatusArchived.String(),
			"utime":  time.Now().UnixMilli(),
		})
	return res.RowsAffected, res.Error
}

// Create 插入一个代码节点，文件节点会同步创建初始版本。
func (g *GORMCodebookDAO) Create(ctx context.Context, c Codebook, code string) (int64, error) {
	now := time.Now().UnixMilli()
	c.CTime, c.UTime = now, now
	if c.Kind == "" {
		c.Kind = domain.CodebookKindFile.String()
	}
	err := g.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&c).Error; err != nil {
			return err
		}
		if c.Kind != domain.CodebookKindFile.String() {
			return nil
		}
		if strings.TrimSpace(code) == "" {
			return nil
		}
		version := CodebookVersion{
			NodeID:       c.ID,
			TenantID:     c.TenantID,
			Scope:        c.Scope,
			VersionNo:    1,
			Code:         code,
			Hash:         hashCode(code),
			AuthorUserID: codebookAuthorUserID(ctx),
			CTime:        now,
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		return tx.Model(&Codebook{}).
			Where("id = ?", c.ID).
			Updates(map[string]any{
				"current_version_id": version.ID,
				"utime":              now,
			}).Error
	})
	return c.ID, err
}

// GetByID 根据主键 ID 查询代码节点。
func (g *GORMCodebookDAO) GetByID(ctx context.Context, id int64) (Codebook, error) {
	var res Codebook
	err := g.dbWithContext(ctx).
		Where("id = ?", id).
		First(&res).Error
	return res, err
}

// GetCurrentVersion 查询节点当前使用版本。
func (g *GORMCodebookDAO) GetCurrentVersion(ctx context.Context, nodeID int64) (CodebookVersion, error) {
	node, err := g.GetByID(ctx, nodeID)
	if err != nil {
		return CodebookVersion{}, err
	}

	var res CodebookVersion
	if node.CurrentVersionID > 0 {
		err = g.dbWithContext(ctx).
			Where("id = ? AND node_id = ?", node.CurrentVersionID, nodeID).
			First(&res).Error
		return res, err
	}
	err = g.dbWithContext(ctx).
		Where("node_id = ?", nodeID).
		Order("version_no DESC, id DESC").
		First(&res).Error
	return res, err
}

// ListVersions 查询指定节点的所有版本。
func (g *GORMCodebookDAO) ListVersions(ctx context.Context, nodeID int64) ([]CodebookVersion, error) {
	var res []CodebookVersion
	err := g.dbWithContext(ctx).
		Where("node_id = ?", nodeID).
		Order("version_no DESC, id DESC").
		Find(&res).Error
	return res, err
}

// ListVersionsByIDs 根据版本 ID 批量查询代码版本。
func (g *GORMCodebookDAO) ListVersionsByIDs(ctx context.Context, ids []int64) ([]CodebookVersion, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var res []CodebookVersion
	err := g.dbWithContext(ctx).
		Where("id IN ?", ids).
		Find(&res).Error
	return res, err
}

// GetVersionByID 根据主键 ID 查询代码版本。
func (g *GORMCodebookDAO) GetVersionByID(ctx context.Context, id int64) (CodebookVersion, error) {
	var res CodebookVersion
	err := g.dbWithContext(ctx).Where("id = ?", id).First(&res).Error
	return res, err
}

// List 分页查询代码节点，按创建时间倒序返回。
func (g *GORMCodebookDAO) List(ctx context.Context, offset, limit int64) ([]Codebook, error) {
	var res []Codebook
	err := g.dbWithContext(ctx).
		Order("ctime DESC").
		Offset(int(offset)).
		Limit(int(limit)).
		Find(&res).Error
	return res, err
}

// ListChildren 查询指定项目和父节点下的子节点。
func (g *GORMCodebookDAO) ListChildren(ctx context.Context, projectID, parentID int64) ([]Codebook, error) {
	var res []Codebook
	err := g.dbWithContext(ctx).
		Where("project_id = ? AND parent_id = ?", projectID, parentID).
		Order("sort_no ASC, kind ASC, name ASC, id ASC").
		Find(&res).Error
	return res, err
}

// ListChildrenBySpace 查询指定空间和父节点下的子节点。
func (g *GORMCodebookDAO) ListChildrenBySpace(ctx context.Context, projectID, parentID int64, scope string) ([]Codebook, error) {
	var res []Codebook
	err := g.dbWithContext(ctx).
		Where("project_id = ? AND parent_id = ? AND scope = ?", projectID, parentID, scope).
		Order("sort_no ASC, kind ASC, name ASC, id ASC").
		Find(&res).Error
	return res, err
}

// Tree 查询指定项目视图下的节点树，SYSTEM 组件库由租户插件透出。
func (g *GORMCodebookDAO) Tree(ctx context.Context, projectID int64) ([]Codebook, error) {
	var res []Codebook
	err := g.dbWithContext(ctx).
		Where("project_id IN ?", []int64{projectID, 0}).
		Order("path_ids ASC, sort_no ASC, kind ASC, name ASC, id ASC").
		Find(&res).Error
	return res, err
}

// Count 统计代码节点总数。
func (g *GORMCodebookDAO) Count(ctx context.Context) (int64, error) {
	var count int64
	err := g.dbWithContext(ctx).Model(&Codebook{}).Count(&count).Error
	return count, err
}

// GetMaxSortNo 查询指定空间和父节点下最大的排序号。
func (g *GORMCodebookDAO) GetMaxSortNo(ctx context.Context, projectID, parentID int64, scope string) (int64, error) {
	var sortNo int64
	err := g.dbWithContext(ctx).
		Model(&Codebook{}).
		Where("project_id = ? AND parent_id = ? AND scope = ?", projectID, parentID, scope).
		Select("COALESCE(MAX(sort_no), 0)").
		Scan(&sortNo).Error
	return sortNo, err
}

// Update 更新代码节点可变字段。
func (g *GORMCodebookDAO) Update(ctx context.Context, c Codebook, code string) (int64, error) {
	now := time.Now().UnixMilli()
	updates := map[string]any{
		"tenant_id": c.TenantID,
		"name":      c.Name,
		"owner":     c.Owner,
		"secret":    c.Secret,
		"scope":     c.Scope,
		"sort_no":   c.SortNo,
		"utime":     now,
	}
	var rowsAffected int64
	err := g.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&Codebook{}).Where("id = ?", c.ID).Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		rowsAffected = res.RowsAffected

		if c.Kind != domain.CodebookKindFile.String() || strings.TrimSpace(code) == "" {
			return nil
		}
		return g.updateCurrentVersionCode(ctx, tx, c, code, now)
	})
	return rowsAffected, err
}

func (g *GORMCodebookDAO) updateCurrentVersionCode(ctx context.Context, tx *gorm.DB, c Codebook, code string, now int64) error {
	var version CodebookVersion
	if c.CurrentVersionID > 0 {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND node_id = ?", c.CurrentVersionID, c.ID).
			First(&version).Error
		if err == nil {
			return g.updateVersionCode(tx, version.ID, c.ID, code)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("node_id = ?", c.ID).
		Order("version_no DESC, id DESC").
		First(&version).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		version = CodebookVersion{
			NodeID:       c.ID,
			TenantID:     c.TenantID,
			Scope:        c.Scope,
			VersionNo:    1,
			Code:         code,
			Hash:         hashCode(code),
			AuthorUserID: codebookAuthorUserID(ctx),
			CTime:        now,
		}
		if err = tx.Create(&version).Error; err != nil {
			return err
		}
		return tx.Model(&Codebook{}).
			Where("id = ?", c.ID).
			Updates(map[string]any{
				"current_version_id": version.ID,
				"utime":              now,
			}).Error
	}
	if err != nil {
		return err
	}

	return g.updateVersionCode(tx, version.ID, c.ID, code)
}

func (g *GORMCodebookDAO) updateVersionCode(tx *gorm.DB, versionID, nodeID int64, code string) error {
	return tx.Model(&CodebookVersion{}).
		Where("id = ? AND node_id = ?", versionID, nodeID).
		Updates(map[string]any{
			"code": code,
			"hash": hashCode(code),
		}).Error
}

// CreateVersion 创建代码版本。
func (g *GORMCodebookDAO) CreateVersion(ctx context.Context, version CodebookVersion) (int64, error) {
	now := time.Now().UnixMilli()
	err := g.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var node Codebook
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND kind = ?", version.NodeID, domain.CodebookKindFile.String()).
			First(&node).Error; err != nil {
			return err
		}

		var maxVersionNo int64
		if err := tx.Model(&CodebookVersion{}).
			Where("node_id = ?", version.NodeID).
			Select("COALESCE(MAX(version_no), 0)").
			Scan(&maxVersionNo).Error; err != nil {
			return err
		}

		version.TenantID = node.TenantID
		version.Scope = node.Scope
		version.VersionNo = maxVersionNo + 1
		if version.AuthorUserID == 0 {
			version.AuthorUserID = codebookAuthorUserID(ctx)
		}
		version.CTime = now
		version.Hash = hashCode(version.Code)
		return tx.Create(&version).Error
	})
	return version.ID, err
}

// UseVersion 设置代码节点当前使用版本。
func (g *GORMCodebookDAO) UseVersion(ctx context.Context, nodeID, versionID int64) (int64, error) {
	var rowsAffected int64
	err := g.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var version CodebookVersion
		if err := tx.Where("id = ? AND node_id = ?", versionID, nodeID).First(&version).Error; err != nil {
			return err
		}
		res := tx.Model(&Codebook{}).
			Where("id = ?", nodeID).
			Updates(map[string]any{
				"current_version_id": versionID,
				"utime":              time.Now().UnixMilli(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		rowsAffected = res.RowsAffected
		return nil
	})
	return rowsAffected, err
}

// UpdateSort 更新单个代码节点的父级、路径和排序号。
func (g *GORMCodebookDAO) UpdateSort(ctx context.Context, item CodebookSortItem) error {
	return g.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return g.updateSortInTx(tx, item)
	})
}

// BatchUpdateSort 批量更新代码节点排序。
func (g *GORMCodebookDAO) BatchUpdateSort(ctx context.Context, items []CodebookSortItem) error {
	if len(items) == 0 {
		return nil
	}
	return g.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := g.updateSortInTx(tx, item); err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete 根据主键 ID 删除代码节点，目录会级联删除整棵子树和对应版本。
func (g *GORMCodebookDAO) Delete(ctx context.Context, id int64) (int64, error) {
	var rowsAffected int64
	err := g.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var node Codebook
		if err := tx.Where("id = ?", id).First(&node).Error; err != nil {
			return err
		}
		subtreePath := fmt.Sprintf("%s%d/%%", node.PathIDs, node.ID)
		var nodes []Codebook
		if err := tx.
			Where("id = ? OR (project_id = ? AND path_ids LIKE ?)", id, node.ProjectID, subtreePath).
			Find(&nodes).Error; err != nil {
			return err
		}
		if len(nodes) == 0 {
			return gorm.ErrRecordNotFound
		}
		ids := lo.Map(nodes, func(n Codebook, _ int) int64 { return n.ID })
		if err := tx.Where("node_id IN ?", ids).Delete(&CodebookVersion{}).Error; err != nil {
			return err
		}
		res := tx.Where("id IN ?", ids).Delete(&Codebook{})
		if res.Error != nil {
			return res.Error
		}
		rowsAffected = res.RowsAffected
		return nil
	})
	return rowsAffected, err
}

func (g *GORMCodebookDAO) dbWithContext(ctx context.Context) *gorm.DB {
	return g.db.WithContext(ctx)
}

func (g *GORMCodebookDAO) updateSortInTx(tx *gorm.DB, item CodebookSortItem) error {
	now := time.Now().UnixMilli()
	var old Codebook
	if err := tx.Where("id = ?", item.ID).First(&old).Error; err != nil {
		return err
	}
	res := tx.Model(&Codebook{}).
		Where("id = ?", item.ID).
		Updates(map[string]any{
			"parent_id":  item.ParentID,
			"project_id": item.ProjectID,
			"path_ids":   item.PathIDs,
			"depth":      item.Depth,
			"sort_no":    item.SortNo,
			"utime":      now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	if old.Kind != domain.CodebookKindDirectory.String() || old.PathIDs == item.PathIDs {
		return nil
	}
	oldPrefix := fmt.Sprintf("%s%d/", old.PathIDs, old.ID)
	newPrefix := fmt.Sprintf("%s%d/", item.PathIDs, item.ID)
	var descendants []Codebook
	if err := tx.
		Where("path_ids LIKE ?", oldPrefix+"%").
		Find(&descendants).Error; err != nil {
		return err
	}
	depthDelta := item.Depth - old.Depth
	for _, descendant := range descendants {
		newPathIDs := strings.Replace(descendant.PathIDs, oldPrefix, newPrefix, 1)
		if err := tx.Model(&Codebook{}).
			Where("id = ?", descendant.ID).
			Updates(map[string]any{
				"path_ids": newPathIDs,
				"depth":    descendant.Depth + depthDelta,
				"utime":    now,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func codebookAuthorUserID(ctx context.Context) int64 {
	userID := ctxutil.GetUserID(ctx).Int64()
	if userID > 0 {
		return userID
	}
	return defaultCodebookAuthorUserID
}
