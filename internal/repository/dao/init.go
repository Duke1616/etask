package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&Task{},
		&Codebook{},
		&Runner{},
		&Variable{},
		&TaskExecution{},
		&TaskExecutionLog{},
	)
}
