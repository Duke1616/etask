package calendar

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPublicHoliday_FromRemoteCDN(t *testing.T) {
	// 真实验证从 https://cdn.jsdelivr.net/gh/NateScarlet/holiday-cn@master/2026.json 拉取解析
	testCases := []struct {
		name     string
		date     time.Time
		expected bool
	}{
		{
			name:     "2026年元旦当天（01-01）是法定放假",
			date:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local),
			expected: true,
		},
		{
			name:     "2026年3月18日（周三）普通工作日不是法定放假",
			date:     time.Date(2026, 3, 18, 0, 0, 0, 0, time.Local),
			expected: false,
		},
		{
			name:     "2026年国庆节当天（10-01）是法定放假",
			date:     time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local),
			expected: true,
		},
		{
			name:     "2026年国庆长假后首个工作日（10-08周四）不是法定放假",
			date:     time.Date(2026, 10, 8, 0, 0, 0, 0, time.Local),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, IsPublicHoliday(tc.date))
		})
	}
}

func TestSkipHolidaySchedule_Next(t *testing.T) {
	parser := cron.NewParser(
		cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
	)
	// 周一至周五，每天上午 10:00:00 触发
	baseSchedule, err := parser.Parse("0 0 10 * * 1-5")
	require.NoError(t, err)

	schedule := NewSkipHolidaySchedule(baseSchedule)

	testCases := []struct {
		name     string
		baseTime time.Time
		expected string
	}{
		{
			name:     "周一上午推算 -> 下一次为周二",
			baseTime: time.Date(2026, 9, 21, 10, 0, 0, 0, time.Local), // 周一
			expected: "2026-09-22 10:00:00",
		},
		{
			name:     "普通周五上午推算 -> 自动跳过周六周日，下一次为周一",
			baseTime: time.Date(2026, 9, 18, 10, 0, 0, 0, time.Local), // 周五
			expected: "2026-09-21 10:00:00",                            // 周一
		},
		{
			name:     "国庆长假跳过测试 -> 9月30日（周三）执行完毕后，根据外部权威数据自动跨越 10-01 至 10-07 假期，命中 10-08（周四）",
			baseTime: time.Date(2026, 9, 30, 10, 0, 0, 0, time.Local), // 9月30日周三
			expected: "2026-10-08 10:00:00",                            // 10月8日周四
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := schedule.Next(tc.baseTime)
			assert.Equal(t, tc.expected, actual.Format("2006-01-02 15:04:05"))
		})
	}
}

func TestSkipHolidaySchedule_HighFrequency(t *testing.T) {
	parser := cron.NewParser(
		cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
	)
	// 每 5 分钟执行一次：0 */5 * * * *
	// 7 天长假正常情况下会有 7 * 288 = 2016 个触发点
	// 若用传统逐步探测，在 366 步限制下会提前耗尽导致返回零值任务挂起；
	// 使用整天快进机制后，每天只需 1 步快进，共消耗 7 步即可准确命中假期后首个触发点。
	baseSchedule, err := parser.Parse("0 */5 * * * *")
	require.NoError(t, err)

	schedule := NewSkipHolidaySchedule(baseSchedule)

	// 9月30日 23:55:00 执行完毕
	baseTime := time.Date(2026, 9, 30, 23, 55, 0, 0, time.Local)
	actual := schedule.Next(baseTime)

	// 应直接命中 10月8日 00:00:00，绝不返回零值
	assert.False(t, actual.IsZero())
	assert.Equal(t, "2026-10-08 00:00:00", actual.Format("2006-01-02 15:04:05"))
}

func TestDailyCacheValid(t *testing.T) {
	testCases := []struct {
		name     string
		lastTime time.Time
		nowTime  time.Time
		expected bool
	}{
		{
			name:     "零值时间必定失效",
			lastTime: time.Time{},
			nowTime:  time.Date(2026, 10, 25, 12, 0, 0, 0, time.Local),
			expected: false,
		},
		{
			name:     "同一天不同时刻：缓存有效（纯内存查表，不重新拉取）",
			lastTime: time.Date(2026, 10, 25, 0, 1, 0, 0, time.Local),
			nowTime:  time.Date(2026, 10, 25, 18, 30, 0, 0, time.Local),
			expected: true,
		},
		{
			name:     "跨天后（次日凌晨）：缓存失效（触发每日重新拉取）",
			lastTime: time.Date(2026, 10, 25, 23, 59, 0, 0, time.Local),
			nowTime:  time.Date(2026, 10, 26, 0, 1, 0, 0, time.Local),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isCacheValid(tc.lastTime, tc.nowTime))
		})
	}
}
