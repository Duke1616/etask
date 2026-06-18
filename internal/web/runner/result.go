package runner

import "github.com/ecodeclub/ginx"

const (
	SystemErrorCode      = 505001
	InvalidParameterCode = 505002
)

var (
	systemErrorResult = ginx.Result{Code: SystemErrorCode, Msg: "系统错误"}
	invalidIDResult   = ginx.Result{Code: InvalidParameterCode, Msg: "执行器 ID 非法"}
)

func invalidParameterResult(err error) ginx.Result {
	return ginx.Result{
		Code: InvalidParameterCode,
		Msg:  err.Error(),
	}
}
