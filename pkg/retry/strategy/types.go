package strategy

import (
	"time"
)

type Strategy interface {
	// NextWithRetries 根据当前重试次数计算下一次重试间隔，如果不需要继续重试，那么第二参数返回 false
	NextWithRetries(retries int32) (time.Duration, bool)
	// Next 返回下一次重试的间隔，如果不需要继续重试，那么第二参数返回 false
	Next() (time.Duration, bool)
	// Report 记录本次错误并返回后续使用的策略状态。
	Report(err error) Strategy
}
