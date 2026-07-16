package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/Duke1616/etask/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ArtifactRelease 映射代码制品的不可变发布记录。
type ArtifactRelease struct {
	ID             int64  `gorm:"primaryKey;column:id;type:bigint;autoIncrement;comment:'制品发布记录自增ID'"`
	TenantID       int64  `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;index;uniqueIndex:uniq_artifact_target_revision_digest,priority:1;comment:'租户ID'" eiam:"shared:scope = 'SYSTEM'"`
	Scope          string `gorm:"column:scope;type:varchar(32);not null;index;uniqueIndex:uniq_artifact_target_revision_digest,priority:2;comment:'制品作用域 SYSTEM/TENANT'"`
	ProjectID      int64  `gorm:"column:project_id;type:bigint;not null;default:0;index;uniqueIndex:uniq_artifact_target_revision_digest,priority:3;comment:'租户代码项目ID，系统制品为0'"`
	SourceRevision int64  `gorm:"column:source_revision;type:bigint;not null;default:0;uniqueIndex:uniq_artifact_target_revision_digest,priority:4;comment:'发布时的项目源码修订号'"`
	Digest         string `gorm:"column:digest;type:char(64);not null;uniqueIndex:uniq_artifact_target_revision_digest,priority:5;comment:'制品语义内容SHA-256摘要'"`
	BlobChecksum   string `gorm:"column:blob_checksum;type:char(64);not null;comment:'压缩对象SHA-256校验和'"`
	ObjectKey      string `gorm:"column:object_key;type:varchar(512);not null;comment:'制品存储对象键'"`
	Size           int64  `gorm:"column:compressed_size;type:bigint;not null;comment:'压缩对象大小（字节）'"`
	Format         string `gorm:"column:format;type:varchar(32);not null;comment:'制品压缩格式'"`
	FormatVersion  int32  `gorm:"column:format_version;type:int;not null;comment:'制品格式版本'"`
	Message        string `gorm:"column:message;type:varchar(255);not null;default:'';comment:'发布说明'"`
	AuthorUserID   int64  `gorm:"column:author_user_id;type:bigint unsigned;not null;default:0;comment:'发布用户ID'"`
	Active         bool   `gorm:"column:active;type:tinyint(1);not null;default:0;index;comment:'是否为当前激活版本'"`
	CTime          int64  `gorm:"column:ctime;type:bigint;not null;comment:'创建时间（毫秒）'"`
}

// TableName 返回制品发布记录表名。
func (ArtifactRelease) TableName() string { return "codebook_artifact_releases" }

// ArtifactDAO 定义制品发布记录的数据访问操作。
type ArtifactDAO interface {
	// CreateAndActivate 创建目标下的制品发布记录，并将其设置为当前激活版本。
	CreateAndActivate(ctx context.Context, release ArtifactRelease) (ArtifactRelease, error)
	// FindActive 查询目标当前激活的制品发布记录。
	FindActive(ctx context.Context, scope string, projectID int64) (ArtifactRelease, error)
	// ListActiveByProjectIDs 批量查询租户项目当前激活的制品发布记录。
	ListActiveByProjectIDs(ctx context.Context, projectIDs []int64) ([]ArtifactRelease, error)
	// FindByID 根据发布 ID 查询制品发布记录。
	FindByID(ctx context.Context, id int64) (ArtifactRelease, error)
	// List 分页查询目标下的制品发布记录及总数。
	List(ctx context.Context, scope string, projectID, offset, limit int64) ([]ArtifactRelease, int64, error)
	// Activate 将目标下指定发布记录切换为当前激活版本。
	Activate(ctx context.Context, scope string, projectID, id int64) error
}

// GORMArtifactDAO 基于 GORM 实现制品发布记录的数据访问能力。
type GORMArtifactDAO struct{ db *gorm.DB }

// NewGORMArtifactDAO 创建 GORM 版 ArtifactDAO。
func NewGORMArtifactDAO(db *gorm.DB) ArtifactDAO { return &GORMArtifactDAO{db: db} }

// CreateAndActivate 创建或复用制品发布记录，并在同一事务中将其激活。
func (g *GORMArtifactDAO) CreateAndActivate(ctx context.Context, release ArtifactRelease) (ArtifactRelease, error) {
	release.CTime = time.Now().UnixMilli()
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 租户项目先锁定源码修订号，防止上传期间的编辑被错误发布。
		if err := verifyArtifactSourceRevision(tx, release); err != nil {
			return err
		}
		// 相同修订和摘要复用不可变记录，重复发布不会制造历史噪声。
		var persisted ArtifactRelease
		if err := tx.Where("scope = ? AND project_id = ? AND source_revision = ? AND digest = ?",
			release.Scope, release.ProjectID, release.SourceRevision, release.Digest).
			Attrs(release).FirstOrCreate(&persisted).Error; err != nil {
			return err
		}
		release = persisted
		// 锁定目标现有发布后再切换 active，串行化并发发布和回滚。
		if err := lockArtifactTarget(tx, release.Scope, release.ProjectID); err != nil {
			return err
		}
		if err := activateArtifactRelease(tx, release.Scope, release.ProjectID, release.ID); err != nil {
			return err
		}
		release.Active = true
		return nil
	})
	return release, err
}

// verifyArtifactSourceRevision 锁定租户项目并确认上传期间源码未发生变化。
func verifyArtifactSourceRevision(tx *gorm.DB, release ArtifactRelease) error {
	if release.Scope != domain.CodebookScopeTenant.String() {
		return nil
	}
	var project CodebookProject
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id", "source_revision").
		Where("id = ?", release.ProjectID).
		First(&project).Error; err != nil {
		return fmt.Errorf("锁定制品对应代码项目失败: %w", err)
	}
	if project.SourceRevision != release.SourceRevision {
		return fmt.Errorf("代码项目在制品上传期间发生变更，请重新发布")
	}
	return nil
}

// FindActive 查询目标当前激活的制品发布记录。
func (g *GORMArtifactDAO) FindActive(ctx context.Context, scope string, projectID int64) (ArtifactRelease, error) {
	var release ArtifactRelease
	err := g.db.WithContext(ctx).Where("scope = ? AND project_id = ? AND active = ?", scope, projectID, true).
		Order("id DESC").First(&release).Error
	return release, err
}

// ListActiveByProjectIDs 批量查询租户项目当前激活的制品发布记录。
func (g *GORMArtifactDAO) ListActiveByProjectIDs(ctx context.Context, projectIDs []int64) ([]ArtifactRelease, error) {
	if len(projectIDs) == 0 {
		return []ArtifactRelease{}, nil
	}
	var releases []ArtifactRelease
	err := g.db.WithContext(ctx).
		Where("scope = ? AND project_id IN ? AND active = ?",
			domain.CodebookScopeTenant.String(), projectIDs, true).
		Find(&releases).Error
	return releases, err
}

// FindByID 根据发布 ID 查询制品发布记录。
func (g *GORMArtifactDAO) FindByID(ctx context.Context, id int64) (ArtifactRelease, error) {
	var release ArtifactRelease
	err := g.db.WithContext(ctx).Where("id = ?", id).First(&release).Error
	return release, err
}

// List 分页查询目标下的制品发布记录及总数。
func (g *GORMArtifactDAO) List(ctx context.Context, scope string, projectID, offset, limit int64) ([]ArtifactRelease, int64, error) {
	var releases []ArtifactRelease
	var total int64
	db := g.db.WithContext(ctx).Model(&ArtifactRelease{}).Where("scope = ? AND project_id = ?", scope, projectID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order("id DESC").Offset(int(offset)).Limit(int(limit)).Find(&releases).Error
	return releases, total, err
}

// Activate 在事务中将目标下指定发布记录切换为当前激活版本。
func (g *GORMArtifactDAO) Activate(ctx context.Context, scope string, projectID, id int64) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 与发布流程使用同一目标锁，保证任意时刻最多一个激活版本。
		if err := lockArtifactTarget(tx, scope, projectID); err != nil {
			return err
		}
		var release ArtifactRelease
		if err := tx.Select("id").Where("id = ? AND scope = ? AND project_id = ?", id, scope, projectID).
			First(&release).Error; err != nil {
			return err
		}
		return activateArtifactRelease(tx, scope, projectID, id)
	})
}

// lockArtifactTarget 串行化同一目标的激活切换，避免并发产生多条激活记录。
func lockArtifactTarget(tx *gorm.DB, scope string, projectID int64) error {
	var ids []int64
	return tx.Model(&ArtifactRelease{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("scope = ? AND project_id = ?", scope, projectID).
		Order("id ASC").
		Pluck("id", &ids).Error
}

// activateArtifactRelease 关闭目标原激活记录并启用指定发布记录。
func activateArtifactRelease(tx *gorm.DB, scope string, projectID, id int64) error {
	if err := tx.Model(&ArtifactRelease{}).
		Where("scope = ? AND project_id = ? AND active = ? AND id <> ?", scope, projectID, true, id).
		Update("active", false).Error; err != nil {
		return err
	}
	return tx.Model(&ArtifactRelease{}).
		Where("id = ? AND scope = ? AND project_id = ?", id, scope, projectID).
		Update("active", true).Error
}
