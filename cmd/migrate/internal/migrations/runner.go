package migrations

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Duke1616/eiam/pkg/migration"
	"github.com/Duke1616/etask/internal/repository/dao"
	"github.com/Duke1616/etask/pkg/sqlx"
	"go.mongodb.org/mongo-driver/bson"
	"gorm.io/gorm"
)

// mongoRunnerVariable 是 eflow MongoDB 中的执行单元变量。
type mongoRunnerVariable struct {
	Key    string `bson:"key"`
	Value  string `bson:"value"`
	Secret bool   `bson:"secret"`
}

// mongoRunner 是 eflow MongoDB 中的执行单元源数据。
type mongoRunner struct {
	ID                 int64                 `bson:"id"`
	Name               string                `bson:"name"`
	CodebookIdentifier string                `bson:"codebook_uid"`
	CodebookSecret     string                `bson:"codebook_secret"`
	Topic              string                `bson:"topic"`
	Kind               string                `bson:"kind"`
	Target             string                `bson:"target"`
	Handler            string                `bson:"handler"`
	Tags               []string              `bson:"tags"`
	Action             uint8                 `bson:"action"`
	Desc               string                `bson:"desc"`
	Variables          []mongoRunnerVariable `bson:"variables"`
	CTime              int64                 `bson:"ctime"`
	UTime              int64                 `bson:"utime"`
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

// Convert 仅负责 runner 主表基础字段迁移。
// runner.codebook_id 需要在基础数据迁移完成后，通过后处理逻辑单独回填。
// 同时，在这里做最初步的老数据兼容性映射（如果 Kind 为空，则降级为 KAFKA 并以 Topic 为 Target）。
func (runnerMigrator) Convert(src mongoRunner) dao.Runner {
	kind := strings.TrimSpace(src.Kind)
	target := strings.TrimSpace(src.Target)
	if kind == "" {
		kind = "KAFKA"
		target = strings.TrimSpace(src.Topic)
	}

	return dao.Runner{
		ID:             src.ID,
		TenantID:       DefaultTenantID,
		Name:           src.Name,
		CodebookSecret: src.CodebookSecret,
		Kind:           kind,
		Target:         target,
		Handler:        src.Handler,
		Tags:           sqlx.JSONColumn[[]string]{Val: src.Tags, Valid: src.Tags != nil},
		Action:         src.Action,
		Desc:           src.Desc,
		CTime:          src.CTime,
		UTime:          src.UTime,
	}
}

// ResolveRunnerCodebookIDs 在 runner 基础数据迁移完成后，
// 根据源 runner 的 codebook 标识，回填目标 runner.codebook_id，并补全为空的 handler。
func ResolveRunnerCodebookIDs(ctx context.Context, env migration.MigrationEnv) error {
	if env.DryRun {
		log.Printf("[dry-run] 跳过 runner 数据回填与兼容性修复")
		return nil
	}

	// 1. 加载 c_codebook 的标识到 ID 与 Language 的映射
	lookup, err := loadCodebookLookup(ctx, env)
	if err != nil {
		return err
	}

	// 2. 遍历源 c_runner，匹配并更新目标 MySQL 数据库中的 runner 相关字段
	cursor, err := env.MongoDB.Collection("c_runner").Find(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("查询源 c_runner 失败: %w", err)
	}
	defer cursor.Close(ctx)

	err = env.MySQLDst.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for cursor.Next(ctx) {
			var src struct {
				ID                 int64  `bson:"id"`
				CodebookIdentifier string `bson:"codebook_uid"`
				Handler            string `bson:"handler"`
			}
			if err := cursor.Decode(&src); err != nil {
				return fmt.Errorf("解码源 c_runner 失败: %w", err)
			}

			// 如果标识为空或对应 codebook 不存在，ok 为 false，直接跳过以防脏数据中断迁移
			cbInfo, ok := lookup[strings.TrimSpace(src.CodebookIdentifier)]
			if !ok {
				continue
			}

			updates := map[string]any{
				"codebook_id": cbInfo.ID,
			}

			// 处理遗留的 handler 为空的老数据，若 handler 为空，则使用 codebook 的 language 字段回填。
			if strings.TrimSpace(src.Handler) == "" {
				updates["handler"] = strings.TrimSpace(cbInfo.Language)
			}

			// 在一个事务中逐条更新，仅更新 codebook_id 和可能缺失的 handler，避免重复更新其他已映射好的字段
			if err := tx.Model(&dao.Runner{}).
				Where("id = ?", src.ID).
				Updates(updates).Error; err != nil {
				return fmt.Errorf("更新 runner 失败: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return cursor.Err()
}

type codebookInfo struct {
	ID       int64
	Language string
}

// loadCodebookLookup 加载 codebook 标识 -> ID 与 Language 的映射。
func loadCodebookLookup(ctx context.Context, env migration.MigrationEnv) (map[string]codebookInfo, error) {
	cursor, err := env.MongoDB.Collection("c_codebook").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("查询源 c_codebook 失败: %w", err)
	}
	defer cursor.Close(ctx)

	lookup := make(map[string]codebookInfo)
	for cursor.Next(ctx) {
		var src struct {
			ID         int64  `bson:"id"`
			Identifier string `bson:"identifier"`
			Language   string `bson:"language"`
		}
		if err := cursor.Decode(&src); err != nil {
			return nil, fmt.Errorf("解码源 c_codebook 失败: %w", err)
		}

		if key := strings.TrimSpace(src.Identifier); key != "" {
			lookup[key] = codebookInfo{
				ID:       src.ID,
				Language: src.Language,
			}
		}
	}

	return lookup, cursor.Err()
}
