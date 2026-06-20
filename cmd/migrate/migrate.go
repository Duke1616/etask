package migrate

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Duke1616/eiam/pkg/migration"
	"github.com/Duke1616/etask/cmd/migrate/internal/config"
	"github.com/Duke1616/etask/cmd/migrate/internal/migrations"
	"github.com/Duke1616/etask/internal/repository/dao"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var force bool

func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

// NewCommand 返回 migrate 子命令。
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "迁移 eflow 的脚本模板和执行单元数据",
		Run: func(cmd *cobra.Command, args []string) {
			runMigrate(force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "强制重新执行迁移（清除历史迁移记录）")
	return cmd
}

func runMigrate(force bool) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("使用迁移配置: %s", cfg.ConfigFile)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	mCfg := migration.Config{
		MongoDSN:    cfg.MongoDSN,
		MongoDBName: cfg.MongoDBName,
		MySQLDstDSN: cfg.MySQLDstDSN,
		BatchSize:   cfg.BatchSize,
		Timeout:     cfg.Timeout,
		AutoMigrate: cfg.AutoMigrate,
		Truncate:    cfg.Truncate,
		DryRun:      cfg.DryRun,
		Force:       force,
	}

	runner := migration.NewRunner(mCfg, migrations.All(),
		migration.WithDefaultTenantID(migrations.DefaultTenantID),
		migration.WithAutoMigrateFunc(func(db *gorm.DB) error {
			if err := db.AutoMigrate(&dao.Codebook{}, &dao.CodebookVersion{}, &dao.CodebookProject{}, &dao.Runner{}, &dao.Variable{}); err != nil {
				return err
			}
			var count int64
			if err := db.Model(&dao.CodebookProject{}).Where("id = ?", 1).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				p := dao.CodebookProject{
					ID:       1,
					TenantID: 2,
					Scope:    "TENANT",
					Name:     "工单",
					Desc:     "系统自动创建的工单代码资源项目",
					SortNo:   10000,
					Status:   "NORMAL",
					CTime:    time.Now().UnixMilli(),
					UTime:    time.Now().UnixMilli(),
				}
				if err := db.Create(&p).Error; err != nil {
					return err
				}
			}
			return nil
		}),
		migration.WithPostHooks(migrations.ResolveRunnerCodebookIDs),
	)

	if err = runner.Run(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("迁移完成")
}
