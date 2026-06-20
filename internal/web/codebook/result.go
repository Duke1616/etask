package codebook

import "github.com/ecodeclub/ginx"

const (
	SystemErrorCode      = 504001
	InvalidParameterCode = 504002
)

var (
	systemErrorResult      = ginx.Result{Code: SystemErrorCode, Msg: "系统错误"}
	invalidCodebookIDError = ginx.Result{Code: InvalidParameterCode, Msg: "脚本模板 ID 非法"}
	invalidProjectIDError  = ginx.Result{Code: InvalidParameterCode, Msg: "脚本项目 ID 非法"}
)

func invalidParameterResult(err error) ginx.Result {
	return ginx.Result{
		Code: InvalidParameterCode,
		Msg:  err.Error(),
	}
}
