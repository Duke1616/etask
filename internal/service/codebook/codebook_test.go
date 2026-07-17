package codebook

import (
	"errors"
	"testing"

	"github.com/Duke1616/etask/internal/errs"
)

func TestCodebookNameConflict(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		fileName string
		want     string
	}{
		{name: "补充冲突文件名", err: errs.ErrCodebookNameConflict, fileName: "deploy.sh", want: "同级目录下已存在同名文件或目录：deploy.sh"},
		{name: "保留其他错误", err: errs.ErrInvalidParameter, fileName: "deploy.sh", want: "参数非法"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := codebookNameConflict(testCase.err, testCase.fileName)
			if got.Error() != testCase.want {
				t.Fatalf("codebookNameConflict() = %q, 期望 %q", got, testCase.want)
			}
			if errors.Is(testCase.err, errs.ErrCodebookNameConflict) &&
				!errors.Is(got, errs.ErrCodebookNameConflict) {
				t.Fatal("转换后的错误未保留 Codebook 名称冲突语义")
			}
		})
	}
}
