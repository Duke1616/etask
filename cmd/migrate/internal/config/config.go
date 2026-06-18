package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 是迁移命令的运行配置。
type Config struct {
	MongoDSN    string
	MongoDBName string
	MySQLDstDSN string
	BatchSize   int
	Timeout     time.Duration
	AutoMigrate bool
	Truncate    bool
	DryRun      bool
	ConfigFile  string
}

// Load 从全局 viper 读取迁移配置。
func Load() (Config, error) {
	cfg := Config{
		MongoDSN:    viper.GetString("migration.source.mongo.dsn"),
		MongoDBName: viper.GetString("migration.source.mongo.database"),
		MySQLDstDSN: viper.GetString("mysql.dsn"),
		BatchSize:   viper.GetInt("migration.batch_size"),
		Timeout:     viper.GetDuration("migration.timeout"),
		AutoMigrate: viper.GetBool("migration.auto_migrate"),
		Truncate:    viper.GetBool("migration.truncate"),
		DryRun:      viper.GetBool("migration.dry_run"),
		ConfigFile:  viper.ConfigFileUsed(),
	}

	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Minute
	}
	return cfg, cfg.validate()
}

func (cfg Config) validate() error {
	if cfg.BatchSize <= 0 {
		return fmt.Errorf("migration.batch_size 必须大于 0")
	}
	if cfg.Timeout <= 0 {
		return fmt.Errorf("migration.timeout 必须大于 0")
	}
	if cfg.MongoDSN == "" {
		return fmt.Errorf("migration.source.mongo.dsn 不能为空")
	}
	if cfg.MongoDBName == "" {
		return fmt.Errorf("migration.source.mongo.database 不能为空")
	}
	if cfg.MySQLDstDSN == "" {
		return fmt.Errorf("mysql.dsn 不能为空")
	}
	return nil
}
