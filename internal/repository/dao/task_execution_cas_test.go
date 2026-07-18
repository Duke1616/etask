package dao

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestWithExecutionStatusCAS(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8mb4&parseTime=True&loc=Local",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	require.NoError(t, err)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return withExecutionStatusCAS(tx.Model(&TaskExecution{}), 42,
			[]string{TaskExecutionStatusPrepare, TaskExecutionStatusRunning}).
			Updates(map[string]any{"status": "SUCCESS"})
	})

	require.Contains(t, sql, "WHERE id = 42 AND status IN ('PREPARE','RUNNING')")
}
