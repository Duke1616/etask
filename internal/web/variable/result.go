package variable

import "github.com/ecodeclub/ginx"

const (
	SystemErrorCode      = 506001
	InvalidParameterCode = 506002
)

var (
	systemErrorResult      = ginx.Result{Code: SystemErrorCode, Msg: "系统错误"}
	invalidVariableIDError = ginx.Result{Code: InvalidParameterCode, Msg: "变量 ID 非法"}
)

func invalidParameterResult(err error) ginx.Result {
	return ginx.Result{
		Code: InvalidParameterCode,
		Msg:  err.Error(),
	}
}
