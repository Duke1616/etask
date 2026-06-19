package codebook

import (
	"context"
	"fmt"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/errs"
	"github.com/Duke1616/etask/internal/repository"
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen -source=./codebook.go -package=codebookmocks -destination=./mocks/codebook.mock.go -typed Service

// Service 定义脚本模板业务操作。
type Service interface {
	// Create 校验并创建脚本模板。
	Create(ctx context.Context, req domain.Codebook) (int64, error)
	// GetByID 根据主键 ID 获取脚本模板。
	GetByID(ctx context.Context, id int64) (domain.Codebook, error)
	// GetByIdentifier 根据业务唯一标识获取脚本模板。
	GetByIdentifier(ctx context.Context, identifier string) (domain.Codebook, error)
	// List 分页获取脚本模板列表和总数。
	List(ctx context.Context, offset, limit int64) ([]domain.Codebook, int64, error)
	// Update 校验并更新脚本模板。
	Update(ctx context.Context, req domain.Codebook) (int64, error)
	// Delete 根据主键 ID 删除脚本模板。
	Delete(ctx context.Context, id int64) (int64, error)
}

type service struct {
	repo repository.CodebookRepository
}

// NewService 创建脚本模板服务。
func NewService(repo repository.CodebookRepository) Service {
	return &service{repo: repo}
}

// Create 校验并创建脚本模板。
func (s *service) Create(ctx context.Context, req domain.Codebook) (int64, error) {
	if err := req.Validate(); err != nil {
		return 0, err
	}
	return s.repo.Create(ctx, req)
}

// GetByID 根据主键 ID 获取脚本模板。
func (s *service) GetByID(ctx context.Context, id int64) (domain.Codebook, error) {
	if id <= 0 {
		return domain.Codebook{}, fmt.Errorf("%w: id = %d", errs.ErrInvalidParameter, id)
	}
	return s.repo.GetByID(ctx, id)
}

// GetByIdentifier 根据业务唯一标识获取脚本模板。
func (s *service) GetByIdentifier(ctx context.Context, identifier string) (domain.Codebook, error) {
	if identifier == "" {
		return domain.Codebook{}, fmt.Errorf("%w: identifier is empty", errs.ErrInvalidParameter)
	}
	return s.repo.GetByIdentifier(ctx, identifier)
}

// List 分页获取脚本模板列表和总数。
func (s *service) List(ctx context.Context, offset, limit int64) ([]domain.Codebook, int64, error) {
	var (
		eg    errgroup.Group
		res   []domain.Codebook
		total int64
	)
	eg.Go(func() error {
		var err error
		res, err = s.repo.List(ctx, offset, limit)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = s.repo.Total(ctx)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, 0, err
	}
	return res, total, nil
}

// Update 校验并更新脚本模板。
func (s *service) Update(ctx context.Context, req domain.Codebook) (int64, error) {
	if req.ID <= 0 {
		return 0, fmt.Errorf("%w: id = %d", errs.ErrInvalidParameter, req.ID)
	}
	if err := req.Validate(); err != nil {
		return 0, err
	}
	return s.repo.Update(ctx, req)
}

// Delete 根据主键 ID 删除脚本模板。
func (s *service) Delete(ctx context.Context, id int64) (int64, error) {
	if id <= 0 {
		return 0, fmt.Errorf("%w: id = %d", errs.ErrInvalidParameter, id)
	}
	return s.repo.Delete(ctx, id)
}
