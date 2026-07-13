package pool

import "github.com/ecodeclub/ginx"

const (
	SystemErrorCode      = 507001
	InvalidParameterCode = 507002
	NotFoundCode         = 507003
)

var (
	systemErrorResult = ginx.Result{Code: SystemErrorCode, Msg: "系统错误"}
)

func successResult() ginx.Result {
	return ginx.Result{Msg: "success"}
}

func invalidParameterResult(err error) ginx.Result {
	return ginx.Result{
		Code: InvalidParameterCode,
		Msg:  err.Error(),
	}
}

func notFoundResult(err error) ginx.Result {
	return ginx.Result{
		Code: NotFoundCode,
		Msg:  err.Error(),
	}
}
