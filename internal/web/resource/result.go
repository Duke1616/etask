package resource

import "github.com/ecodeclub/ginx"

var systemErrorResult = ginx.Result{
	Code: 5,
	Msg:  "系统错误",
}
