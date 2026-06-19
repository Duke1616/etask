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
	// ListMergedVariables 获取执行单元变量，私有变量覆盖全局变量。
	ListMergedVariables(ctx context.Context, runnerID int64) ([]domain.RunnerVariable, error)
}

type runnerRepository struct {
	runnerDAO   dao.RunnerDAO
	variableDAO dao.VariableDAO
	crypto      cryptox.Crypto
}

// NewRunnerRepository 创建基于 RunnerDAO 的执行单元仓储。
func NewRunnerRepository(runnerDAO dao.RunnerDAO, variableDAO dao.VariableDAO, crypto cryptox.Crypto) RunnerRepository {
	return &runnerRepository{runnerDAO: runnerDAO, variableDAO: variableDAO, crypto: crypto}
}

// Create 保存执行单元。
func (repo *runnerRepository) Create(ctx context.Context, req domain.Runner) (int64, error) {
	return repo.runnerDAO.CreateWithVariables(ctx, repo.toEntity(req), repo.toVariables(req.ID, req.Variables))
}

// Update 更新执行单元可变字段。
func (repo *runnerRepository) Update(ctx context.Context, req domain.Runner) (int64, error) {
	return repo.runnerDAO.UpdateWithVariables(ctx, repo.toEntity(req), repo.toVariables(req.ID, req.Variables))
}

// Delete 根据主键 ID 删除执行单元。
func (repo *runnerRepository) Delete(ctx context.Context, id int64) (int64, error) {
	return repo.runnerDAO.DeleteWithVariables(ctx, id)
}

// FindByID 根据主键 ID 加载执行单元。
func (repo *runnerRepository) FindByID(ctx context.Context, id int64) (domain.Runner, error) {
	r, err := repo.runnerDAO.FindByID(ctx, id)
	if err != nil {
		return domain.Runner{}, err
	}
	res := repo.toDomain(r)
	res.Variables, err = repo.listRunnerVariables(ctx, id)
	if err != nil {
		return domain.Runner{}, err
	}
	return res, nil
}

// List 分页查询执行单元。
func (repo *runnerRepository) List(ctx context.Context, offset, limit int64, keyword, kind string) ([]domain.Runner, error) {
	rs, err := repo.runnerDAO.List(ctx, offset, limit, keyword, kind)
	if err != nil {
		return nil, err
	}
	return slice.Map(rs, func(_ int, src dao.Runner) domain.Runner {
		return repo.toDomain(src)
	}), nil
}

// Count 统计匹配条件的执行单元总数。
func (repo *runnerRepository) Count(ctx context.Context, keyword, kind string) (int64, error) {
	return repo.runnerDAO.Count(ctx, keyword, kind)
}

// ListByCodebookUID 查询绑定指定脚本模板 UID 的执行单元。
func (repo *runnerRepository) ListByCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]domain.Runner, error) {
	rs, err := repo.runnerDAO.ListByCodebookUID(ctx, offset, limit, codebookUID, keyword, kind)
	if err != nil {
		return nil, err
	}
	return slice.Map(rs, func(_ int, src dao.Runner) domain.Runner {
		return repo.toDomain(src)
	}), nil
}

// CountByCodebookUID 统计绑定指定脚本模板 UID 的执行单元数量。
func (repo *runnerRepository) CountByCodebookUID(ctx context.Context, codebookUID, keyword, kind string) (int64, error) {
	return repo.runnerDAO.CountByCodebookUID(ctx, codebookUID, keyword, kind)
}

// ListExcludeCodebookUID 查询未绑定指定脚本模板 UID 的执行单元。
func (repo *runnerRepository) ListExcludeCodebookUID(ctx context.Context, offset, limit int64, codebookUID, keyword, kind string) ([]domain.Runner, error) {
	rs, err := repo.runnerDAO.ListExcludeCodebookUID(ctx, offset, limit, codebookUID, keyword, kind)
	if err != nil {
		return nil, err
	}
	return slice.Map(rs, func(_ int, src dao.Runner) domain.Runner {
		return repo.toDomain(src)
	}), nil
}

// CountExcludeCodebookUID 统计未绑定指定脚本模板 UID 的执行单元数量。
func (repo *runnerRepository) CountExcludeCodebookUID(ctx context.Context, codebookUID, keyword, kind string) (int64, error) {
	return repo.runnerDAO.CountExcludeCodebookUID(ctx, codebookUID, keyword, kind)
}

// ListByCodebookUIDs 查询绑定任一脚本模板 UID 的执行单元。
func (repo *runnerRepository) ListByCodebookUIDs(ctx context.Context, codebookUIDs []string) ([]domain.Runner, error) {
	rs, err := repo.runnerDAO.ListByCodebookUIDs(ctx, codebookUIDs)
	if err != nil {
		return nil, err
	}
	return slice.Map(rs, func(_ int, src dao.Runner) domain.Runner {
		return repo.toDomain(src)
	}), nil
}

// ListByIDs 根据 ID 列表批量查询执行单元。
func (repo *runnerRepository) ListByIDs(ctx context.Context, ids []int64) ([]domain.Runner, error) {
	rs, err := repo.runnerDAO.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return slice.Map(rs, func(_ int, src dao.Runner) domain.Runner {
		return repo.toDomain(src)
	}), nil
}

// FindByCodebookUIDAndTag 根据脚本模板 UID 和标签加载执行单元。
func (repo *runnerRepository) FindByCodebookUIDAndTag(ctx context.Context, codebookUID string, tag string) (domain.Runner, error) {
	r, err := repo.runnerDAO.FindByCodebookUIDAndTag(ctx, codebookUID, tag)
	if err != nil {
		return domain.Runner{}, err
	}
	return repo.toDomain(r), nil
}

// AggregateTags 按脚本模板 UID 聚合执行单元标签。
func (repo *runnerRepository) AggregateTags(ctx context.Context) ([]domain.RunnerTags, error) {
	runners, err := repo.runnerDAO.AggregateTags(ctx)
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

func (repo *runnerRepository) ListMergedVariables(ctx context.Context, runnerID int64) ([]domain.RunnerVariable, error) {
	variables, err := repo.variableDAO.ListGlobalAndRunner(ctx, runnerID)
	if err != nil {
		return nil, err
	}
	return repo.toRunnerVariables(mergeVariablesByKey(variables)), nil
}

func mergeVariablesByKey(variables []dao.Variable) []dao.Variable {
	merged := make([]dao.Variable, 0, len(variables))
	indexes := make(map[string]int, len(variables))
	for _, variable := range variables {
		idx, ok := indexes[variable.Key]
		if ok {
			merged[idx] = variable
			continue
		}
		indexes[variable.Key] = len(merged)
		merged = append(merged, variable)
	}
	return merged
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
		CTime:          req.CTime,
		UTime:          req.UTime,
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
		CTime:          req.CTime,
		UTime:          req.UTime,
	}
	if req.Kind == "" {
		r.Kind = domain.RunnerKindKafka
	}
	return r
}

func (repo *runnerRepository) toVariables(targetID int64, variables []domain.RunnerVariable) []dao.Variable {
	merged := make(map[string]domain.RunnerVariable, len(variables))
	keys := make([]string, 0, len(variables))
	for _, src := range variables {
		if src.Key == "" {
			continue
		}
		if _, ok := merged[src.Key]; !ok {
			keys = append(keys, src.Key)
		}
		merged[src.Key] = src
	}
	return lo.Map(keys, func(key string, _ int) dao.Variable {
		src := merged[key]
		value := src.Value
		if src.Secret && value != "" {
			if encVal, err := repo.crypto.Encrypt(value); err == nil {
				value = encVal
			}
		}
		return dao.Variable{
			Scope:    domain.VariableScopeRunner.String(),
			TargetID: targetID,
			Key:      src.Key,
			Value:    value,
			Secret:   src.Secret,
		}
	})
}

func (repo *runnerRepository) listRunnerVariables(ctx context.Context, runnerID int64) ([]domain.RunnerVariable, error) {
	variables, err := repo.variableDAO.ListByScope(ctx, domain.VariableScopeRunner.String(), runnerID)
	if err != nil {
		return nil, err
	}
	return repo.toRunnerVariables(variables), nil
}

func (repo *runnerRepository) toRunnerVariables(variables []dao.Variable) []domain.RunnerVariable {
	return lo.Map(variables, func(src dao.Variable, _ int) domain.RunnerVariable {
		value := src.Value
		if src.Secret && value != "" {
			if decVal, err := repo.crypto.Decrypt(value); err == nil {
				value = decVal
			}
		}
		return domain.RunnerVariable{
			Key:    src.Key,
			Value:  value,
			Secret: src.Secret,
		}
	})
}
