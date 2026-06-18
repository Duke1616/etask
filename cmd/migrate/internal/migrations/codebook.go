package migrations

import (
	"github.com/Duke1616/eiam/pkg/migration"
	"github.com/Duke1616/etask/internal/repository/dao"
)

// mongoCodebook 是 eflow MongoDB 中的脚本模板源数据。
type mongoCodebook struct {
	ID         int64  `bson:"id"`
	Name       string `bson:"name"`
	Owner      string `bson:"owner"`
	Identifier string `bson:"identifier"`
	Code       string `bson:"code"`
	Language   string `bson:"language"`
	Secret     string `bson:"secret"`
	CTime      int64  `bson:"ctime"`
	UTime      int64  `bson:"utime"`
}

type codebookMigrator struct{}

// NewCodebookMigrator 创建脚本模板迁移器。
func NewCodebookMigrator() migration.Migrator {
	return migration.NewMongoMigrator[mongoCodebook, dao.Codebook](codebookMigrator{})
}

func (codebookMigrator) Name() string {
	return "codebook"
}

func (codebookMigrator) CollectionName() string {
	return "c_codebook"
}

func (codebookMigrator) Convert(src mongoCodebook) dao.Codebook {
	return dao.Codebook{
		ID:         src.ID,
		TenantID:   DefaultTenantID,
		Name:       src.Name,
		Owner:      src.Owner,
		Identifier: src.Identifier,
		Code:       src.Code,
		Language:   src.Language,
		Secret:     src.Secret,
		CTime:      src.CTime,
		UTime:      src.UTime,
	}
}
