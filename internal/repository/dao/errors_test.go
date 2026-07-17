package dao

import (
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func TestIsDuplicateKeyError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "MySQL 唯一键冲突", err: &mysql.MySQLError{Number: 1062}, want: true},
		{name: "GORM 唯一键冲突", err: fmt.Errorf("保存失败: %w", gorm.ErrDuplicatedKey), want: true},
		{name: "其他 MySQL 错误", err: &mysql.MySQLError{Number: 1048}},
		{name: "没有错误"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isDuplicateKeyError(testCase.err); got != testCase.want {
				t.Fatalf("isDuplicateKeyError() = %t, 期望 %t", got, testCase.want)
			}
		})
	}
}
