package migrations

import "github.com/Duke1616/eiam/pkg/migration"

const DefaultTenantID int64 = 2

// All 返回 etask 需要从 eflow 迁移的脚本模板和执行单元任务。
func All() []migration.Migrator {
	return []migration.Migrator{
		NewCodebookMigrator(),
		NewRunnerMigrator(),
	}
}
