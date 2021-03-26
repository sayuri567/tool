package timeutil

import (
	"time"
)

// GetLastMonthSameDay 获取上个月同一天，最大值不会超过上个月最后一天，时分秒取参数时间
func GetLastMonthSameDay(t time.Time) time.Time {
	thisMonth := GetStartOfMonth(t)
	day := int((t.UnixNano() - thisMonth.UnixNano()) / (int64(time.Hour) * 24))
	lastMonth := GetStartOfLastMonth(t).AddDate(0, 0, day)
	if lastMonth.After(thisMonth) {
		lastMonth = thisMonth.Add(-1)
	}
	return time.Date(lastMonth.Year(), lastMonth.Month(), lastMonth.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// GetLastQuarterSameDay 获取上个季度同一天，最大值不会超过上个季度最后一天，时分秒取参数时间
func GetLastQuarterSameDay(t time.Time) time.Time {
	startQuarter := GetStartOfQuarter(t)
	day := int((t.UnixNano() - startQuarter.UnixNano()) / (int64(time.Hour) * 24))
	lastQuarter := GetStartOfLastQuarter(t).AddDate(0, 0, day)
	if lastQuarter.After(startQuarter) {
		lastQuarter = startQuarter.Add(-1)
	}
	return time.Date(lastQuarter.Year(), lastQuarter.Month(), lastQuarter.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// GetLastDaySameTime 获取昨日同一时间
func GetLastDaySameTime(t time.Time) time.Time {
	return t.AddDate(0, 0, -1)
}

// GetStartOfMonth 获取本月开始时间
func GetStartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// GetStartOfLastMonth 获取上月开始时间
func GetStartOfLastMonth(t time.Time) time.Time {
	return GetStartOfMonth(t).AddDate(0, -1, 0)
}

// GetStartOfQuarter 获取本季度开始时间，季度开始时间分别为1，4，7，10月
func GetStartOfQuarter(t time.Time) time.Time {
	// 月份减1，整除3，余数为距离季度开始的月份
	mc := int((t.Month() - 1) % 3)
	return GetStartOfMonth(t).AddDate(0, -mc, 0)
}

// GetStartOfLastQuarter 获取上个季度开始时间
func GetStartOfLastQuarter(t time.Time) time.Time {
	return GetStartOfQuarter(t).AddDate(0, -3, 0)
}

// GetStartOfDay 获取今日开始时间
func GetStartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// GetStartOfDay 获取今日开始时间
func GetStartOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

// GetStartOfWeek GetStartOfWeek
func GetStartOfWeek(t time.Time) time.Time {
	offset := int(time.Monday - t.Weekday())
	if offset > 0 {
		offset = -6
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).AddDate(0, 0, offset)
}
