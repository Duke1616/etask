package runner

import (
	"context"
	"fmt"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/errs"
	"github.com/Duke1616/etask/internal/repository"
	"golang.org/x/sync/errgroup"
)

// Service 定义脚本执行单元业务操作。
type Service interface {
	// Create 校验并创建执行单元。
	Create(ctx context.Context, req domain.Runner) (int64, error)
	// Update 校验并更新执行单元。
	Update(ctx context.Context, req domain.Runner) (int64, error)
	// FindByID 根据主键 ID 获取执行单元。
	FindByID(ctx context.Context, id int64) (domain.Runner, error)
	// Delete 根据主键 ID 删除执行单元。
	Delete(ctx context.Context, id int64) (int64, error)
	// List 分页获取执行单元列表和总数。
	List(ctx context.Context, offset, limit int64, keyword, kind string) ([]domain.Runner, int64, error)
	// FindByCodebookUIDAndTag 根据脚本模板 UID 和标签获取执行单元。
	FindByCodebookUIDAndTag(ctx context.Context, codebookUID string, tag string) (domain.Runner, error)
	// ListByCodebookUID 获取绑定指定脚本模板 UID 的执行单元列表。
	ListByCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]domain.Runner, int64, error)
	// ListExcludeCodebookUID 获取未绑定指定脚本模板 UID 的执行单元列表。
	ListExcludeCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]domain.Runner, int64, error)
	// ListByCodebookUIDs 获取绑定任一脚本模板 UID 的执行单元列表。
	ListByCodebookUIDs(ctx context.Context, codebookUIDs []string) ([]domain.Runner, error)
	// ListByIDs 根据 ID 列表批量获取执行单元。
	ListByIDs(ctx context.Context, ids []int64) ([]domain.Runner, error)
	// AggregateTags 按脚本模板 UID 聚合执行单元标签。
	AggregateTags(ctx context.Context) ([]domain.RunnerTags, error)
}

type service struct {
	repo repository.RunnerRepository
}

// NewService 创建执行单元服务。
func NewService(repo repository.RunnerRepository) Service {
	return &service{repo: repo}
}

// Create 校验并创建执行单元。
func (s *service) Create(ctx context.Context, req domain.Runner) (int64, error) {
	if err := req.Validate(); err != nil {
		return 0, err
	}
	if req.Action == 0 {
		req.Action = domain.RunnerActionRegistered
	}
	return s.repo.Create(ctx, req)
}

// Update 校验并更新执行单元。
func (s *service) Update(ctx context.Context, req domain.Runner) (int64, error) {
	if req.ID <= 0 {
		return 0, fmt.Errorf("%w: id = %d", errs.ErrInvalidParameter, req.ID)
	}
	if err := req.Validate(); err != nil {
		return 0, err
	}
	return s.repo.Update(ctx, req)
}

// FindByID 根据主键 ID 获取执行单元。
func (s *service) FindByID(ctx context.Context, id int64) (domain.Runner, error) {
	if id <= 0 {
		return domain.Runner{}, fmt.Errorf("%w: id = %d", errs.ErrInvalidParameter, id)
	}
	return s.repo.FindByID(ctx, id)
}

// Delete 根据主键 ID 删除执行单元。
func (s *service) Delete(ctx context.Context, id int64) (int64, error) {
	if id <= 0 {
		return 0, fmt.Errorf("%w: id = %d", errs.ErrInvalidParameter, id)
	}
	return s.repo.Delete(ctx, id)
}

// List 分页获取执行单元列表和总数。
func (s *service) List(ctx context.Context, offset, limit int64, keyword, kind string) ([]domain.Runner, int64, error) {
	var (
		eg    errgroup.Group
		res   []domain.Runner
		total int64
	)
	eg.Go(func() error {
		var err error
		res, err = s.repo.List(ctx, offset, limit, keyword, kind)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = s.repo.Count(ctx, keyword, kind)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, 0, err
	}
	return res, total, nil
}

// FindByCodebookUIDAndTag 根据脚本模板 UID 和标签获取执行单元。
func (s *service) FindByCodebookUIDAndTag(ctx context.Context, codebookUID string, tag string) (domain.Runner, error) {
	if codebookUID == "" {
		return domain.Runner{}, fmt.Errorf("%w: codebook_uid is empty", errs.ErrInvalidParameter)
	}
	if tag == "" {
		return domain.Runner{}, fmt.Errorf("%w: tag is empty", errs.ErrInvalidParameter)
	}
	return s.repo.FindByCodebookUIDAndTag(ctx, codebookUID, tag)
}

// ListByCodebookUID 获取绑定指定脚本模板 UID 的执行单元列表。
func (s *service) ListByCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]domain.Runner, int64, error) {
	var (
		eg    errgroup.Group
		res   []domain.Runner
		total int64
	)
	eg.Go(func() error {
		var err error
		res, err = s.repo.ListByCodebookUID(ctx, offset, limit, codebookUID, keyword, kind)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = s.repo.CountByCodebookUID(ctx, codebookUID, keyword, kind)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, 0, err
	}
	return res, total, nil
}

// ListExcludeCodebookUID 获取未绑定指定脚本模板 UID 的执行单元列表。
func (s *service) ListExcludeCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]domain.Runner, int64, error) {
	var (
		eg    errgroup.Group
		res   []domain.Runner
		total int64
	)
	eg.Go(func() error {
		var err error
		res, err = s.repo.ListExcludeCodebookUID(ctx, offset, limit, codebookUID, keyword, kind)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = s.repo.CountExcludeCodebookUID(ctx, codebookUID, keyword, kind)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, 0, err
	}
	return res, total, nil
}

// ListByCodebookUIDs 获取绑定任一脚本模板 UID 的执行单元列表。
func (s *service) ListByCodebookUIDs(ctx context.Context, codebookUIDs []string) ([]domain.Runner, error) {
	return s.repo.ListByCodebookUIDs(ctx, codebookUIDs)
}

// ListByIDs 根据 ID 列表批量获取执行单元。
func (s *service) ListByIDs(ctx context.Context, ids []int64) ([]domain.Runner, error) {
	return s.repo.ListByIDs(ctx, ids)
}

// AggregateTags 按脚本模板 UID 聚合执行单元标签。
func (s *service) AggregateTags(ctx context.Context) ([]domain.RunnerTags, error) {
	return s.repo.AggregateTags(ctx)
}
