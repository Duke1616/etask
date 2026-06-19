package migrations

import (
	"github.com/Duke1616/eiam/pkg/migration"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/repository/dao"
)

type runnerVariableMigrator struct{}

// NewRunnerVariableMigrator 创建执行单元变量迁移器。
func NewRunnerVariableMigrator() migration.Migrator {
	return migration.NewMongoMigratorMany[mongoRunner, dao.Variable](runnerVariableMigrator{})
}

func (runnerVariableMigrator) Name() string {
	return "variables"
}

func (runnerVariableMigrator) CollectionName() string {
	return "c_runner"
}

func (runnerVariableMigrator) ConvertMany(src mongoRunner) []dao.Variable {
	variables := make([]dao.Variable, 0, len(src.Variables))
	for _, v := range src.Variables {
		if v.Key == "" {
			continue
		}
		variables = append(variables, dao.Variable{
			TenantID: DefaultTenantID,
			Scope:    domain.VariableScopeRunner.String(),
			TargetID: src.ID,
			Key:      v.Key,
			Value:    v.Value,
			Secret:   v.Secret,
			CTime:    src.CTime,
			UTime:    src.UTime,
		})
	}
	return variables
}
