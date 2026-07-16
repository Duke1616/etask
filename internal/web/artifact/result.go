package artifact

import "github.com/ecodeclub/ginx"

const (
	SystemErrorCode      = 508001
	InvalidParameterCode = 508002
)

var systemErrorResult = ginx.Result{Code: SystemErrorCode, Msg: "系统错误"}

func invalidParameterResult(err error) ginx.Result {
	return ginx.Result{Code: InvalidParameterCode, Msg: err.Error()}
}
