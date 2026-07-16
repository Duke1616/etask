package preview

import "github.com/ecodeclub/ginx"

const (
	systemErrorCode      = 509001
	invalidParameterCode = 509002
)

var systemErrorResult = ginx.Result{Code: systemErrorCode, Msg: "系统错误"}

func invalidParameterResult(err error) ginx.Result {
	return ginx.Result{Code: invalidParameterCode, Msg: err.Error()}
}
