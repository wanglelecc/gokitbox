package uDate

import (
	"testing"
	"time"
)

func TestUnixMillis(t *testing.T) {
	got := UnixMillis()
	// 验证返回的是合理的时间戳（2020年之后）
	if got < 1577836800000 {
		t.Errorf("UnixMillis() = %d, seems too old", got)
	}
	// 验证是毫秒时间戳（比秒时间戳大1000倍）
	now := time.Now().UnixMilli()
	if got < now-1000 || got > now+1000 {
		t.Errorf("UnixMillis() = %d, not in expected range", got)
	}
}

func TestCurrentDate(t *testing.T) {
	got := CurrentDate()
	// 验证格式是 20060102（8位数字）
	if len(got) != 8 {
		t.Errorf("CurrentDate() length = %d, want 8", len(got))
	}
	// 验证可以解析
	_, err := time.Parse(DATE_COMPACT, got)
	if err != nil {
		t.Errorf("CurrentDate() = %s, not valid format: %v", got, err)
	}
}

func TestBeforeDayDate(t *testing.T) {
	got := BeforeDayDate()
	yesterday := time.Now().AddDate(0, 0, -1).Format(DATE_COMPACT)
	if got != yesterday {
		t.Errorf("BeforeDayDate() = %s, want %s", got, yesterday)
	}
}

func TestAfterDayDateUnix(t *testing.T) {
	got := AfterDayDateUnix()
	// 应该是明天凌晨的时间戳
	tomorrow := time.Now().AddDate(0, 0, 1)
	tomorrowStart := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.Local)
	expected := tomorrowStart.Unix()
	// 允许 ±60 秒的误差
	if got < expected-60 || got > expected+60 {
		t.Errorf("AfterDayDateUnix() = %d, want around %d", got, expected)
	}
}

func TestAfterDayToNowUnixDiff(t *testing.T) {
	got := AfterDayToNowUnixDiff()
	// 应该是一个正数（到明天凌晨的秒数）
	if got <= 0 || got > 86400 {
		t.Errorf("AfterDayToNowUnixDiff() = %d, should be between 1 and 86400", got)
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name     string
		t        time.Time
		layout   []string
		expected string
	}{
		{"默认格式", time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local), nil, "2024-01-15 10:30:00"},
		{"日期格式", time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local), []string{DATE_FORMAT}, "2024-01-15"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Format(tt.t, tt.layout...)
			if got != tt.expected {
				t.Errorf("Format() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		layout  []string
		wantErr bool
	}{
		{"默认格式", "2024-01-15 10:30:00", nil, false},
		{"日期格式", "2024-01-15", []string{DATE_FORMAT}, false},
		{"无效格式", "invalid", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.s, tt.layout...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// 验证解析正确
				if got.IsZero() {
					t.Errorf("Parse() returned zero time")
				}
			}
		})
	}
}

func TestUnixToTime(t *testing.T) {
	ts := int64(1705315800) // 2024-01-15 15:30:00
	got := UnixToTime(ts)
	if got.Unix() != ts {
		t.Errorf("UnixToTime(%d).Unix() = %d", ts, got.Unix())
	}
}

func TestDayStart(t *testing.T) {
	input := time.Date(2024, 1, 15, 10, 30, 45, 0, time.Local)
	got := DayStart(input)
	expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	if got != expected {
		t.Errorf("DayStart() = %v, want %v", got, expected)
	}
}

func TestDayEnd(t *testing.T) {
	input := time.Date(2024, 1, 15, 10, 30, 45, 0, time.Local)
	got := DayEnd(input)
	expected := time.Date(2024, 1, 15, 23, 59, 59, 0, time.Local)
	if got != expected {
		t.Errorf("DayEnd() = %v, want %v", got, expected)
	}
}

func TestWeekStart(t *testing.T) {
	// 2024-01-15 是周一
	input := time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local)
	got := WeekStart(input)
	expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	if got != expected {
		t.Errorf("WeekStart() = %v, want %v", got, expected)
	}

	// 2024-01-17 是周三，应该返回周一
	input = time.Date(2024, 1, 17, 10, 30, 0, 0, time.Local)
	got = WeekStart(input)
	if got != expected {
		t.Errorf("WeekStart() = %v, want %v", got, expected)
	}
}

func TestWeekEnd(t *testing.T) {
	// 2024-01-15 是周一，周日是 2024-01-21
	input := time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local)
	got := WeekEnd(input)
	expected := time.Date(2024, 1, 21, 23, 59, 59, 0, time.Local)
	if got != expected {
		t.Errorf("WeekEnd() = %v, want %v", got, expected)
	}
}

func TestMonthStart(t *testing.T) {
	input := time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local)
	got := MonthStart(input)
	expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	if got != expected {
		t.Errorf("MonthStart() = %v, want %v", got, expected)
	}
}

func TestMonthEnd(t *testing.T) {
	input := time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local)
	got := MonthEnd(input)
	expected := time.Date(2024, 1, 31, 23, 59, 59, 0, time.Local)
	if got != expected {
		t.Errorf("MonthEnd() = %v, want %v", got, expected)
	}
}

func TestIsToday(t *testing.T) {
	// 现在应该是今天
	if !IsToday(time.Now()) {
		t.Error("IsToday(time.Now()) = false, want true")
	}
	// 昨天不应该是今天
	if IsToday(time.Now().AddDate(0, 0, -1)) {
		t.Error("IsToday(yesterday) = true, want false")
	}
}

func TestDiffDays(t *testing.T) {
	t1 := time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local)
	t2 := time.Date(2024, 1, 10, 8, 0, 0, 0, time.Local)
	got := DiffDays(t1, t2)
	if got != 5 {
		t.Errorf("DiffDays() = %d, want 5", got)
	}
	// 顺序不影响结果
	got = DiffDays(t2, t1)
	if got != 5 {
		t.Errorf("DiffDays() = %d, want 5", got)
	}
}
