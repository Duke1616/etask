package ioc

import (
	"database/sql"

	"github.com/Duke1616/ework-runner/deploy/migrations"
	"github.com/gotomicro/ego/core/elog"
	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

// RunMigrations 在应用启动时自动执行所有待执行的 SQL 迁移。
// NOTE: 迁移文件通过 embed.FS 编译进二进制，生产环境无需额外文件。
// 若迁移失败则直接 panic，阻止带有不兼容 schema 的服务启动。
func RunMigrations(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		panic("获取 *sql.DB 失败: " + err.Error())
	}

	if err = runGooseMigrations(sqlDB); err != nil {
		panic("数据库迁移失败: " + err.Error())
	}

	elog.DefaultLogger.Info("数据库迁移完成")
}

func runGooseMigrations(sqlDB *sql.DB) error {
	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("mysql"); err != nil {
		return err
	}

	// NOTE: 目录名对应 embed.FS 内的路径（就是 fs.go 所在目录名）
	return goose.Up(sqlDB, ".")
}
