package dao

import (
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// isDuplicateKeyError 统一识别数据库驱动和 GORM 返回的唯一键冲突。
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysql.MySQLError
	return errors.Is(err, gorm.ErrDuplicatedKey) ||
		(errors.As(err, &mysqlErr) && mysqlErr.Number == 1062)
}
