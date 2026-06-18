package repository

import (
	"context"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/repository/dao"
	"github.com/ecodeclub/ekit/slice"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrCodebookNotFound 表示脚本模板不存在。
var ErrCodebookNotFound = gorm.ErrRecordNotFound

// CodebookRepository 定义脚本模板的领域仓储操作。
type CodebookRepository interface {
	// Create 保存脚本模板，未传入 Secret 时自动生成。
	Create(ctx context.Context, req domain.Codebook) (int64, error)
	// GetByID 根据主键 ID 加载脚本模板。
	GetByID(ctx context.Context, id int64) (domain.Codebook, error)
	// GetByIdentifier 根据业务唯一标识加载脚本模板。
	GetByIdentifier(ctx context.Context, identifier string) (domain.Codebook, error)
	// List 分页查询脚本模板。
	List(ctx context.Context, offset, limit int64) ([]domain.Codebook, error)
	// Total 统计脚本模板总数。
	Total(ctx context.Context) (int64, error)
	// Update 更新脚本模板可变字段。
	Update(ctx context.Context, req domain.Codebook) (int64, error)
	// Delete 根据主键 ID 删除脚本模板。
	Delete(ctx context.Context, id int64) (int64, error)
}

type codebookRepository struct {
	dao dao.CodebookDAO
}

// NewCodebookRepository 创建基于 CodebookDAO 的脚本模板仓储。
func NewCodebookRepository(codebookDAO dao.CodebookDAO) CodebookRepository {
	return &codebookRepository{dao: codebookDAO}
}

// Create 保存脚本模板，未传入 Secret 时自动生成。
func (repo *codebookRepository) Create(ctx context.Context, req domain.Codebook) (int64, error) {
	return repo.dao.Create(ctx, repo.toEntity(req))
}

// GetByID 根据主键 ID 加载脚本模板。
func (repo *codebookRepository) GetByID(ctx context.Context, id int64) (domain.Codebook, error) {
	c, err := repo.dao.GetByID(ctx, id)
	if err != nil {
		return domain.Codebook{}, err
	}
	return repo.toDomain(c), nil
}

// GetByIdentifier 根据业务唯一标识加载脚本模板。
func (repo *codebookRepository) GetByIdentifier(ctx context.Context, identifier string) (domain.Codebook, error) {
	c, err := repo.dao.GetByIdentifier(ctx, identifier)
	if err != nil {
		return domain.Codebook{}, err
	}
	return repo.toDomain(c), nil
}

// List 分页查询脚本模板。
func (repo *codebookRepository) List(ctx context.Context, offset, limit int64) ([]domain.Codebook, error) {
	cs, err := repo.dao.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	return slice.Map(cs, func(_ int, src dao.Codebook) domain.Codebook {
		return repo.toDomain(src)
	}), nil
}

// Total 统计脚本模板总数。
func (repo *codebookRepository) Total(ctx context.Context) (int64, error) {
	return repo.dao.Count(ctx)
}

// Update 更新脚本模板可变字段。
func (repo *codebookRepository) Update(ctx context.Context, req domain.Codebook) (int64, error) {
	return repo.dao.Update(ctx, repo.toEntity(req))
}

// Delete 根据主键 ID 删除脚本模板。
func (repo *codebookRepository) Delete(ctx context.Context, id int64) (int64, error) {
	return repo.dao.Delete(ctx, id)
}

func (repo *codebookRepository) toEntity(req domain.Codebook) dao.Codebook {
	secret := req.Secret
	if secret == "" {
		secret = uuid.NewString()
	}
	return dao.Codebook{
		ID:         req.ID,
		TenantID:   req.TenantID,
		Name:       req.Name,
		Owner:      req.Owner,
		Code:       req.Code,
		Language:   req.Language,
		Secret:     secret,
		Identifier: req.Identifier,
		CTime:      req.CTime,
		UTime:      req.UTime,
	}
}

func (repo *codebookRepository) toDomain(req dao.Codebook) domain.Codebook {
	return domain.Codebook{
		ID:         req.ID,
		TenantID:   req.TenantID,
		Name:       req.Name,
		Owner:      req.Owner,
		Code:       req.Code,
		Language:   req.Language,
		Secret:     req.Secret,
		Identifier: req.Identifier,
		CTime:      req.CTime,
		UTime:      req.UTime,
	}
}
