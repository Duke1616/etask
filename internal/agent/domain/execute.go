package domain

type Status uint8

func (s Status) ToUint8() uint8 {
	return uint8(s)
}

const (
	// SUCCESS 成功
	SUCCESS Status = 1
	// FAILED 失败
	FAILED Status = 2
)

type Agent struct {
	Name   string
	Desc   string
	Topic  string
	Status Status
}

type ExecuteReceive struct {
	TaskId    int64  // 任务ID
	Language  string // 使用语言
	Handler   string // 调用方法
	Code      string // 代码
	Args      string // 参数
	Variables string // 变量
}

type Variable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
