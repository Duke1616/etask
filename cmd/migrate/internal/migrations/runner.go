package migrations

import (
	"github.com/Duke1616/eiam/pkg/migration"
	"github.com/Duke1616/etask/internal/repository/dao"
	"github.com/Duke1616/etask/pkg/sqlx"
)

// mongoRunnerVariable 是 eflow MongoDB 中的执行单元变量。
type mongoRunnerVariable struct {
	Key    string `bson:"key"`
	Value  string `bson:"value"`
	Secret bool   `bson:"secret"`
}

// mongoRunner 是 eflow MongoDB 中的执行单元源数据。
type mongoRunner struct {
	ID             int64                 `bson:"id"`
	Name           string                `bson:"name"`
	CodebookUID    string                `bson:"codebook_uid"`
	CodebookSecret string                `bson:"codebook_secret"`
	Kind           string                `bson:"kind"`
	Target         string                `bson:"target"`
	Handler        string                `bson:"handler"`
	Tags           []string              `bson:"tags"`
	Action         uint8                 `bson:"action"`
	Desc           string                `bson:"desc"`
	Variables      []mongoRunnerVariable `bson:"variables"`
	CTime          int64                 `bson:"ctime"`
	UTime          int64                 `bson:"utime"`
}

type runnerMigrator struct{}

// NewRunnerMigrator 创建执行单元迁移器。
func NewRunnerMigrator() migration.Migrator {
	return migration.NewMongoMigrator[mongoRunner, dao.Runner](runnerMigrator{})
}

func (runnerMigrator) Name() string {
	return "runner"
}

func (runnerMigrator) CollectionName() string {
	return "c_runner"
}

func (runnerMigrator) Convert(src mongoRunner) dao.Runner {
	return dao.Runner{
		ID:             src.ID,
		TenantID:       DefaultTenantID,
		Name:           src.Name,
		CodebookUID:    src.CodebookUID,
		CodebookSecret: src.CodebookSecret,
		Kind:           src.Kind,
		Target:         src.Target,
		Handler:        src.Handler,
		Tags:           sqlx.JSONColumn[[]string]{Val: src.Tags, Valid: src.Tags != nil},
		Action:         src.Action,
		Desc:           src.Desc,
		CTime:          src.CTime,
		UTime:          src.UTime,
	}
}
