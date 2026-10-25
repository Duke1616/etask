package calendar

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var (
	mu         sync.RWMutex
	lastLoaded = make(map[int]time.Time) // 记录各年份最后成功拉取的时间戳
	holidays   = make(map[string]bool)    // 缓存法定放假日期：map["2026-10-01"] = true
	httpClient = &http.Client{Timeout: 5 * time.Second}
)

type remoteHolidaySchema struct {
	Year int `json:"year"`
	Days []struct {
		Name     string `json:"name"`
		Date     string `json:"date"`
		IsOffDay bool   `json:"isOffDay"`
	} `json:"days"`
}

// IsPublicHoliday 判定指定时间对应日期是否为法定节假日放假。
// 数据按自然日缓存（每日最多拉取一次），其余时间纯内存读锁 O(1) 查表，绝不重复计算与发起网络 IO。
func IsPublicHoliday(t time.Time) bool {
	year := t.Year()
	ensureYearLoaded(year)

	dateStr := t.Format("2006-01-02")

	mu.RLock()
	defer mu.RUnlock()

	return holidays[dateStr]
}

// isCacheValid 检查指定年份在今天是否已成功拉取过
func isCacheValid(lastTime time.Time, now time.Time) bool {
	if lastTime.IsZero() {
		return false
	}
	y1, m1, d1 := lastTime.Date()
	y2, m2, d2 := now.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// ensureYearLoaded 确保指定年份在当天已有最新数据，否则触发每日一次的外部数据更新
func ensureYearLoaded(year int) {
	now := time.Now()

	// 1. 快速读锁路径：若当天已经拉取过，直接返回，走纯内存查表
	mu.RLock()
	if isCacheValid(lastLoaded[year], now) {
		mu.RUnlock()
		return
	}
	mu.RUnlock()

	// 2. 双重检查锁定，防止跨天时多并发重复拉取
	mu.Lock()
	defer mu.Unlock()

	if isCacheValid(lastLoaded[year], now) {
		return
	}

	// 权威开源数据源：NateScarlet/holiday-cn（自动同步国务院办公厅公报）
	url := fmt.Sprintf("https://cdn.jsdelivr.net/gh/NateScarlet/holiday-cn@master/%d.json", year)
	resp, err := httpClient.Get(url)
	if err != nil {
		// 遭遇网络波动时，设置 1 小时后允许重试，同时保留内存中的旧数据兜底
		lastLoaded[year] = now.Add(-23 * time.Hour)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		lastLoaded[year] = now.Add(-23 * time.Hour)
		return
	}

	var data remoteHolidaySchema
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		lastLoaded[year] = now.Add(-23 * time.Hour)
		return
	}

	// 成功获取，更新内存放假字典
	for _, d := range data.Days {
		if d.IsOffDay {
			holidays[d.Date] = true
		} else {
			delete(holidays, d.Date)
		}
	}

	// 记录当天拉取成功
	lastLoaded[year] = now
}

// SetHTTPClient 支持单测注入或自定义超时客户端
func SetHTTPClient(client *http.Client) {
	mu.Lock()
	defer mu.Unlock()
	httpClient = client
}

// ResetCache 清空缓存，主要供单元测试使用
func ResetCache() {
	mu.Lock()
	defer mu.Unlock()
	lastLoaded = make(map[int]time.Time)
	holidays = make(map[string]bool)
}
