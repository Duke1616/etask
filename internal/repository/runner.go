package repository

import (
	"context"

	"github.com/Duke1616/ecmdb/pkg/cryptox"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/repository/dao"
	"github.com/Duke1616/etask/pkg/sqlx"
	"github.com/ecodeclub/ekit/slice"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// ErrRunnerNotFound 表示执行单元不存在。
var ErrRunnerNotFound = gorm.ErrRecordNotFound

// RunnerRepository 定义执行单元的领域仓储操作。
type RunnerRepository interface {
	// Create 保存执行单元。
	Create(ctx context.Context, req domain.Runner) (int64, error)
	// Update 更新执行单元可变字段。
	Update(ctx context.Context, req domain.Runner) (int64, error)
	// Delete 根据主键 ID 删除执行单元。
	Delete(ctx context.Context, id int64) (int64, error)
	// FindByID 根据主键 ID 加载执行单元。
	FindByID(ctx context.Context, id int64) (domain.Runner, error)
	// List 分页查询执行单元。
	List(ctx context.Context, offset, limit int64, keyword, kind string) ([]domain.Runner, error)
	// Count 统计匹配条件的执行单元总数。
	Count(ctx context.Context, keyword, kind string) (int64, error)
	// ListByCodebookUID 查询绑定指定脚本模板 UID 的执行单元。
	ListByCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]domain.Runner, error)
	// CountByCodebookUID 统计绑定指定脚本模板 UID 的执行单元数量。
	CountByCodebookUID(ctx context.Context, codebookUID, keyword, kind string) (int64, error)
	// ListExcludeCodebookUID 查询未绑定指定脚本模板 UID 的执行单元。
	ListExcludeCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]domain.Runner, error)
	// CountExcludeCodebookUID 统计未绑定指定脚本模板 UID 的执行单元数量。
	CountExcludeCodebookUID(ctx context.Context, codebookUID, keyword, kind string) (int64, error)
	// ListByCodebookUIDs 查询绑定任一脚本模板 UID 的执行单元。
	ListByCodebookUIDs(ctx context.Context, codebookUIDs []string) ([]domain.Runner, error)
	// ListByIDs 根据 ID 列表批量查询执行单元。
	ListByIDs(ctx context.Context, ids []int64) ([]domain.Runner, error)
	// FindByCodebookUIDAndTag 根据脚本模板 UID 和标签加载执行单元。
	FindByCodebookUIDAndTag(ctx context.Context, codebookUID string, tag string) (domain.Runner, error)
	// AggregateTags 按脚本模板 UID 聚合执行单元标签。
	AggregateTags(ctx context.Context) ([]domain.RunnerTags, error)
}

type runnerRepository struct {
	dao    dao.RunnerDAO
	crypto cryptox.Crypto
}

// NewRunnerRepository 创建基于 RunnerDAO 的执行单元仓储。
func NewRunnerRepository(runnerDAO dao.RunnerDAO, crypto cryptox.Crypto) RunnerRepository {
	return &runnerRepository{dao: runnerDAO, crypto: crypto}
}

// Create 保存执行单元。
func (repo *runnerRepository) Create(ctx context.Context, req domain.Runner) (int64, error) {
	return repo.dao.Create(ctx, repo.toEntity(req))
}

// Update 更新执行单元可变字段。
func (repo *runnerRepository) Update(ctx context.Context, req domain.Runner) (int64, error) {
	return repo.dao.Update(ctx, repo.toEntity(req))
}

// Delete 根据主键 ID 删除执行单元。
func (repo *runnerRepository) Delete(ctx context.Context, id int64) (int64, error) {
	return repo.dao.Delete(ctx, id)
}

// FindByID 根据主键 ID 加载执行单元。
func (repo *runnerRepository) FindByID(ctx context.Context, id int64) (domain.Runner, error) {
	r, err := repo.dao.FindByID(ctx, id)
	if err != nil {
		return domain.Runner{}, err
	}
	return repo.toDomain(r), nil
}

// List 分页查询执行单元。
func (repo *runnerRepository) List(ctx context.Context, offset, limit int64, keyword, kind string) ([]domain.Runner, error) {
	rs, err := repo.dao.List(ctx, offset, limit, keyword, kind)
	if err != nil {
		return nil, err
	}
	return slice.Map(rs, func(_ int, src dao.Runner) domain.Runner {
		return repo.toDomain(src)
	}), nil
}

// Count 统计匹配条件的执行单元总数。
func (repo *runnerRepository) Count(ctx context.Context, keyword, kind string) (int64, error) {
	return repo.dao.Count(ctx, keyword, kind)
}

// ListByCodebookUID 查询绑定指定脚本模板 UID 的执行单元。
func (repo *runnerRepository) ListByCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]domain.Runner, error) {
	rs, err := repo.dao.ListByCodebookUID(ctx, offset, limit, codebookUID, keyword, kind)
	if err != nil {
		return nil, err
	}
	return slice.Map(rs, func(_ int, src dao.Runner) domain.Runner {
		return repo.toDomain(src)
	}), nil
}

// CountByCodebookUID 统计绑定指定脚本模板 UID 的执行单元数量。
func (repo *runnerRepository) CountByCodebookUID(ctx context.Context, codebookUID, keyword, kind string) (int64, error) {
	return repo.dao.CountByCodebookUID(ctx, codebookUID, keyword, kind)
}

// ListExcludeCodebookUID 查询未绑定指定脚本模板 UID 的执行单元。
func (repo *runnerRepository) ListExcludeCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]domain.Runner, error) {
	rs, err := repo.dao.ListExcludeCodebookUID(ctx, offset, limit, codebookUID, keyword, kind)
	if err != nil {
		return nil, err
	}
	return slice.Map(rs, func(_ int, src dao.Runner) domain.Runner {
		return repo.toDomain(src)
	}), nil
}

// CountExcludeCodebookUID 统计未绑定指定脚本模板 UID 的执行单元数量。
func (repo *runnerRepository) CountExcludeCodebookUID(ctx context.Context, codebookUID, keyword, kind string) (int64, error) {
	return repo.dao.CountExcludeCodebookUID(ctx, codebookUID, keyword, kind)
}

// ListByCodebookUIDs 查询绑定任一脚本模板 UID 的执行单元。
func (repo *runnerRepository) ListByCodebookUIDs(ctx context.Context, codebookUIDs []string) ([]domain.Runner, error) {
	rs, err := repo.dao.ListByCodebookUIDs(ctx, codebookUIDs)
	if err != nil {
		return nil, err
	}
	return slice.Map(rs, func(_ int, src dao.Runner) domain.Runner {
		return repo.toDomain(src)
	}), nil
}

// ListByIDs 根据 ID 列表批量查询执行单元。
func (repo *runnerRepository) ListByIDs(ctx context.Context, ids []int64) ([]domain.Runner, error) {
	rs, err := repo.dao.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return slice.Map(rs, func(_ int, src dao.Runner) domain.Runner {
		return repo.toDomain(src)
	}), nil
}

// FindByCodebookUIDAndTag 根据脚本模板 UID 和标签加载执行单元。
func (repo *runnerRepository) FindByCodebookUIDAndTag(ctx context.Context, codebookUID string, tag string) (domain.Runner, error) {
	r, err := repo.dao.FindByCodebookUIDAndTag(ctx, codebookUID, tag)
	if err != nil {
		return domain.Runner{}, err
	}
	return repo.toDomain(r), nil
}

// AggregateTags 按脚本模板 UID 聚合执行单元标签。
func (repo *runnerRepository) AggregateTags(ctx context.Context) ([]domain.RunnerTags, error) {
	runners, err := repo.dao.AggregateTags(ctx)
	if err != nil {
		return nil, err
	}
	grouped := lo.GroupBy(runners, func(r dao.Runner) string {
		return r.CodebookUID
	})
	return lo.MapToSlice(grouped, func(uid string, rs []dao.Runner) domain.RunnerTags {
		return domain.RunnerTags{
			CodebookUID: uid,
			TagsMapping: repo.toTagMapping(rs),
		}
	}), nil
}

func (repo *runnerRepository) toTagMapping(rs []dao.Runner) map[string]domain.RunnerTagDetail {
	tagSet := make(map[string]domain.RunnerTagDetail)
	for _, r := range rs {
		for _, tag := range r.Tags.Val {
			tagSet[tag] = domain.RunnerTagDetail{
				Kind:    domain.RunnerKind(r.Kind),
				Target:  r.Target,
				Handler: r.Handler,
			}
		}
	}
	return tagSet
}

func (repo *runnerRepository) toEntity(req domain.Runner) dao.Runner {
	return dao.Runner{
		ID:             req.ID,
		TenantID:       req.TenantID,
		Name:           req.Name,
		CodebookUID:    req.CodebookUID,
		CodebookSecret: req.CodebookSecret,
		Kind:           req.Kind.String(),
		Target:         req.Target,
		Handler:        req.Handler,
		Tags:           sqlx.JSONColumn[[]string]{Val: req.Tags, Valid: true},
		Action:         req.Action.Uint8(),
		Desc:           req.Desc,
		Variables: sqlx.JSONColumn[[]dao.RunnerVariable]{
			Val:   repo.toDAOVariables(req.Variables),
			Valid: true,
		},
		CTime: req.CTime,
		UTime: req.UTime,
	}
}

func (repo *runnerRepository) toDomain(req dao.Runner) domain.Runner {
	r := domain.Runner{
		ID:             req.ID,
		TenantID:       req.TenantID,
		Name:           req.Name,
		CodebookUID:    req.CodebookUID,
		CodebookSecret: req.CodebookSecret,
		Kind:           domain.RunnerKind(req.Kind),
		Target:         req.Target,
		Handler:        req.Handler,
		Tags:           req.Tags.Val,
		Action:         domain.RunnerAction(req.Action),
		Desc:           req.Desc,
		Variables:      repo.toDomainVariables(req.Variables.Val),
		CTime:          req.CTime,
		UTime:          req.UTime,
	}
	if req.Kind == "" {
		r.Kind = domain.RunnerKindKafka
	}
	return r
}

func (repo *runnerRepository) toDAOVariables(vars []domain.RunnerVariable) []dao.RunnerVariable {
	return slice.Map(vars, func(_ int, src domain.RunnerVariable) dao.RunnerVariable {
		val := src.Value
		if src.Secret && val != "" {
			if encVal, err := repo.crypto.Encrypt(val); err == nil {
				val = encVal
			}
		}
		return dao.RunnerVariable{
			Key:    src.Key,
			Value:  val,
			Secret: src.Secret,
		}
	})
}

func (repo *runnerRepository) toDomainVariables(vars []dao.RunnerVariable) []domain.RunnerVariable {
	return slice.Map(vars, func(_ int, src dao.RunnerVariable) domain.RunnerVariable {
		val := src.Value
		if src.Secret && val != "" {
			if decVal, err := repo.crypto.Decrypt(val); err == nil {
				val = decVal
			}
		}
		return domain.RunnerVariable{
			Key:    src.Key,
			Value:  val,
			Secret: src.Secret,
		}
	})
}
