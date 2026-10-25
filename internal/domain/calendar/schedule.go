package calendar

import (
	"time"

	"github.com/robfig/cron/v3"
)

// skipHolidaySchedule 是实现了 cron.Schedule 接口的轻量装饰器
// 在标准 cron 表达式推进的基础上，自动向前顺延跳过落在国家法定节假日放假的日期
type skipHolidaySchedule struct {
	inner cron.Schedule
}

// NewSkipHolidaySchedule 创建跳过法定休假的调度装饰器
func NewSkipHolidaySchedule(inner cron.Schedule) cron.Schedule {
	return &skipHolidaySchedule{inner: inner}
}

// Next 满足 cron.Schedule 接口，返回基准时间之后满足 cron 且不落在法定节假日放假日的下一个时刻
func (s *skipHolidaySchedule) Next(t time.Time) time.Time {
	next := s.inner.Next(t)

	// 限制最多连续跨越 366 天的超长假期（每天只跳 1 步，彻底杜绝高频任务循环溢出）
	for range 366 {
		if next.IsZero() {
			return time.Time{}
		}

		// 若当前算出的日期不是法定节假日放假，即为合法的下一次触发点
		if !IsPublicHoliday(next) {
			return next
		}

		// 核心优化（整天快进）：当天属于法定放假，直接快进到当天的 23:59:59
		// 这样下一跳自动进入次日首个触发时刻，一步跳过当天剩下的成百上千次高频触发点
		endOfDay := time.Date(next.Year(), next.Month(), next.Day(), 23, 59, 59, 0, next.Location())
		next = s.inner.Next(endOfDay)
	}

	return time.Time{}
}
