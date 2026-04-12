package uDate

import (
	"time"
)

// 时间格式常量
const (
	// TIME_FORMAT 标准日期时间格式
	TIME_FORMAT = "2006-01-02 15:04:05"

	// DATE_FORMAT 标准日期格式
	DATE_FORMAT = "2006-01-02"

	// DATE_COMPACT 紧凑日期格式，不含分隔符
	DATE_COMPACT = "20060102"
)

// UnixMillis 获取当前时间的毫秒时间戳
//
// 使用示例：
//
//	ms := uDate.UnixMillis()
//	// ms = 1706745600000
func UnixMillis() int64 {
	return time.Now().UnixMilli()
}

// CurrentDate 获取当前日期字符串，格式 20060102
//
// 使用示例：
//
//	d := uDate.CurrentDate()
//	// d = "20240201"
func CurrentDate() string {
	return time.Now().Format(DATE_COMPACT)
}

// BeforeDayDate 获取昨天日期字符串，格式 20060102
//
// 使用示例：
//
//	d := uDate.BeforeDayDate()
//	// d = "20240131"
func BeforeDayDate() string {
	return time.Now().AddDate(0, 0, -1).Format(DATE_COMPACT)
}

// AfterDayDateUnix 获取明天凌晨 00:00:00 的 Unix 时间戳
//
// 使用示例：
//
//	ts := uDate.AfterDayDateUnix()
//	// ts = 1706745600
func AfterDayDateUnix() int64 {
	aTime, err := time.ParseInLocation("2006-01-02 15:04:05", time.Now().Format("2006-01-02")+" 23:59:59", time.Local)
	if err != nil {
		// 如果出错，直接偏移到第二天
		return time.Now().AddDate(0, 0, 1).Unix()
	}

	return aTime.Unix() + 1
}

// AfterDayToNowUnixDiff 获取距离今天凌晨 00:00:00 的剩余秒数，常用于 Redis key TTL
//
// 使用示例：
//
//	ttl := uDate.AfterDayToNowUnixDiff()
//	rdb.Expire(ctx, key, time.Duration(ttl)*time.Second)
func AfterDayToNowUnixDiff() int64 {
	return AfterDayDateUnix() - time.Now().Unix()
}

// Format 将 time.Time 格式化为字符串，layout 缺省使用 TIME_FORMAT
//
// 使用示例：
//
//	s := uDate.Format(time.Now())
//	// s = "2024-02-01 12:00:00"
//
//	s := uDate.Format(time.Now(), uDate.DATE_FORMAT)
//	// s = "2024-02-01"
func Format(t time.Time, layout ...string) string {
	l := TIME_FORMAT
	if len(layout) > 0 && layout[0] != "" {
		l = layout[0]
	}
	return t.Format(l)
}

// Parse 将字符串解析为 time.Time（本地时区），layout 缺省使用 TIME_FORMAT
//
// 使用示例：
//
//	t, err := uDate.Parse("2024-02-01 12:00:00")
//
//	t, err := uDate.Parse("2024-02-01", uDate.DATE_FORMAT)
func Parse(s string, layout ...string) (time.Time, error) {
	l := TIME_FORMAT
	if len(layout) > 0 && layout[0] != "" {
		l = layout[0]
	}
	return time.ParseInLocation(l, s, time.Local)
}

// UnixToTime 将 Unix 时间戳（秒）转为 time.Time（本地时区）
//
// 使用示例：
//
//	t := uDate.UnixToTime(1706745600)
//	// t = 2024-02-01 08:00:00 +0800 CST
func UnixToTime(unix int64) time.Time {
	return time.Unix(unix, 0)
}

// DayStart 获取指定时间所在天的 00:00:00
//
// 使用示例：
//
//	start := uDate.DayStart(time.Now())
//	// start = 2024-02-01 00:00:00 +0800 CST
func DayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// DayEnd 获取指定时间所在天的 23:59:59
//
// 使用示例：
//
//	end := uDate.DayEnd(time.Now())
//	// end = 2024-02-01 23:59:59 +0800 CST
func DayEnd(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 23, 59, 59, 0, t.Location())
}

// WeekStart 获取指定时间所在周的周一 00:00:00（ISO 8601，周一为第一天）
//
// 使用示例：
//
//	start := uDate.WeekStart(time.Now())
//	// start = 2024-01-29 00:00:00 +0800 CST（本周一）
func WeekStart(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7 // 周日算第 7 天
	}
	return DayStart(t.AddDate(0, 0, 1-wd))
}

// WeekEnd 获取指定时间所在周的周日 23:59:59（ISO 8601，周日为最后一天）
//
// 使用示例：
//
//	end := uDate.WeekEnd(time.Now())
//	// end = 2024-02-04 23:59:59 +0800 CST（本周日）
func WeekEnd(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return DayEnd(t.AddDate(0, 0, 7-wd))
}

// MonthStart 获取指定时间所在月的第一天 00:00:00
//
// 使用示例：
//
//	start := uDate.MonthStart(time.Now())
//	// start = 2024-02-01 00:00:00 +0800 CST
func MonthStart(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
}

// MonthEnd 获取指定时间所在月的最后一天 23:59:59
//
// 使用示例：
//
//	end := uDate.MonthEnd(time.Now())
//	// end = 2024-02-29 23:59:59 +0800 CST（2024 为闰年）
func MonthEnd(t time.Time) time.Time {
	y, m, _ := t.Date()
	// 下个月第一天减一天即为本月最后一天
	lastDay := time.Date(y, m+1, 1, 0, 0, 0, 0, t.Location()).AddDate(0, 0, -1)
	return DayEnd(lastDay)
}

// IsToday 判断指定时间是否为今天
//
// 使用示例：
//
//	uDate.IsToday(time.Now())            // true
//	uDate.IsToday(time.Now().AddDate(0, 0, -1)) // false
func IsToday(t time.Time) bool {
	ny, nm, nd := time.Now().Date()
	ty, tm, td := t.Date()
	return ny == ty && nm == tm && nd == td
}

// DiffDays 计算两个时间相差的天数（绝对值，不含时分秒）
//
// 使用示例：
//
//	days := uDate.DiffDays(time.Now(), time.Now().AddDate(0, 0, -5))
//	// days = 5
func DiffDays(t1, t2 time.Time) int {
	d := DayStart(t1).Sub(DayStart(t2))
	days := int(d.Hours() / 24)
	if days < 0 {
		return -days
	}
	return days
}
