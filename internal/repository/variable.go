package repository

import (
	"context"

	"github.com/Duke1616/ecmdb/pkg/cryptox"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/repository/dao"
	"github.com/samber/lo"
)

type VariableRepository interface {
	// CreateGlobalVariable 创建全局变量。
	CreateGlobalVariable(ctx context.Context, variable domain.Variable) (int64, error)
	// FindGlobalVariable 根据主键 ID 查询全局变量。
	FindGlobalVariable(ctx context.Context, id int64) (domain.Variable, error)
	// UpdateGlobalVariable 更新全局变量。
	UpdateGlobalVariable(ctx context.Context, variable domain.Variable) (int64, error)
	// DeleteGlobalVariable 根据主键 ID 删除全局变量。
	DeleteGlobalVariable(ctx context.Context, id int64) (int64, error)
	// ListGlobalVariables 分页查询全局变量。
	ListGlobalVariables(ctx context.Context, offset, limit int64, keyword string) ([]domain.Variable, error)
	// CountGlobalVariables 统计全局变量数量。
	CountGlobalVariables(ctx context.Context, keyword string) (int64, error)
}

type variableRepository struct {
	dao    dao.VariableDAO
	crypto cryptox.Crypto
}

func NewVariableRepository(variableDAO dao.VariableDAO, crypto cryptox.Crypto) VariableRepository {
	return &variableRepository{dao: variableDAO, crypto: crypto}
}

// CreateGlobalVariable 创建全局变量。
func (repo *variableRepository) CreateGlobalVariable(ctx context.Context, variable domain.Variable) (int64, error) {
	variable.MarkGlobal()
	if err := variable.Validate(); err != nil {
		return 0, err
	}
	return repo.dao.Create(ctx, repo.toDAO(variable))
}

// FindGlobalVariable 根据主键 ID 查询全局变量。
func (repo *variableRepository) FindGlobalVariable(ctx context.Context, id int64) (domain.Variable, error) {
	if err := domain.ValidateVariableID(id); err != nil {
		return domain.Variable{}, err
	}
	variable, err := repo.dao.FindByID(ctx, id)
	if err != nil {
		return domain.Variable{}, err
	}
	res := repo.toDomain(variable)
	if err = res.ValidateGlobalScope(); err != nil {
		return domain.Variable{}, err
	}
	return res, nil
}

// UpdateGlobalVariable 更新全局变量。
func (repo *variableRepository) UpdateGlobalVariable(ctx context.Context, variable domain.Variable) (int64, error) {
	if err := variable.ValidateID(); err != nil {
		return 0, err
	}
	variable.MarkGlobal()
	if err := variable.Validate(); err != nil {
		return 0, err
	}
	return repo.dao.Update(ctx, repo.toDAO(variable))
}

// DeleteGlobalVariable 根据主键 ID 删除全局变量。
func (repo *variableRepository) DeleteGlobalVariable(ctx context.Context, id int64) (int64, error) {
	if err := domain.ValidateVariableID(id); err != nil {
		return 0, err
	}
	return repo.dao.DeleteByIDAndScope(ctx, id, domain.VariableScopeGlobal.String(), domain.VariableGlobalTarget)
}

// ListGlobalVariables 分页查询全局变量。
func (repo *variableRepository) ListGlobalVariables(ctx context.Context, offset, limit int64, keyword string) ([]domain.Variable, error) {
	variables, err := repo.dao.ListByScopePage(ctx, domain.VariableScopeGlobal.String(), domain.VariableGlobalTarget, offset, limit, keyword)
	if err != nil {
		return nil, err
	}
	return lo.Map(variables, func(src dao.Variable, _ int) domain.Variable {
		return repo.toDomain(src)
	}), nil
}

// CountGlobalVariables 统计全局变量数量。
func (repo *variableRepository) CountGlobalVariables(ctx context.Context, keyword string) (int64, error) {
	return repo.dao.CountByScope(ctx, domain.VariableScopeGlobal.String(), domain.VariableGlobalTarget, keyword)
}

func (repo *variableRepository) toDAO(src domain.Variable) dao.Variable {
	return dao.Variable{
		ID:       src.ID,
		TenantID: src.TenantID,
		Scope:    src.Scope.String(),
		TargetID: src.TargetID,
		Key:      src.Key,
		Value:    repo.encryptSecretValue(src.Value, src.Secret),
		Secret:   src.Secret,
		CTime:    src.CTime,
		UTime:    src.UTime,
	}
}

func (repo *variableRepository) toDomain(src dao.Variable) domain.Variable {
	return domain.Variable{
		ID:       src.ID,
		TenantID: src.TenantID,
		Scope:    domain.VariableScope(src.Scope),
		TargetID: src.TargetID,
		Key:      src.Key,
		Value:    repo.decryptSecretValue(src.Value, src.Secret),
		Secret:   src.Secret,
		CTime:    src.CTime,
		UTime:    src.UTime,
	}
}

func (repo *variableRepository) encryptSecretValue(value string, secret bool) string {
	if !secret || value == "" {
		return value
	}
	encVal, err := repo.crypto.Encrypt(value)
	if err != nil {
		return value
	}
	return encVal
}

func (repo *variableRepository) decryptSecretValue(value string, secret bool) string {
	if !secret || value == "" {
		return value
	}
	decVal, err := repo.crypto.Decrypt(value)
	if err != nil {
		return value
	}
	return decVal
}
