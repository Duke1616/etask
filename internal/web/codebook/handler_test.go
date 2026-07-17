package codebook

import (
	"fmt"
	"testing"

	"github.com/Duke1616/etask/internal/errs"
)

func TestTranslateError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string
	}{
		{
			name:     "名称冲突",
			err:      fmt.Errorf("%w：deploy.sh", errs.ErrCodebookNameConflict),
			wantCode: CodebookNameConflictCode,
			wantMsg:  "同级目录下已存在同名文件或目录：deploy.sh",
		},
		{name: "参数非法", err: errs.ErrInvalidParameter, wantCode: InvalidParameterCode, wantMsg: "参数非法"},
		{name: "系统错误", err: fmt.Errorf("database unavailable"), wantCode: SystemErrorCode, wantMsg: "系统错误"},
	}

	handler := &Handler{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := handler.translateError(testCase.err)
			if result.Code != testCase.wantCode || result.Msg != testCase.wantMsg {
				t.Fatalf("translateError() = (%d, %q), 期望 (%d, %q)",
					result.Code, result.Msg, testCase.wantCode, testCase.wantMsg)
			}
		})
	}
}
