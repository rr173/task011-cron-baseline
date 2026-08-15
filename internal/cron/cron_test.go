package cron

import (
	"testing"
	"time"
)

// TestProbe_StrictlyAfter 验证 Next 对恰在分钟整点的基准时间返回严格晚于的时刻。
// 表达式 "*/15 * * * *" 在 12:00:00 整点调用 Next，期望返回 12:15 而非 12:00。
func TestProbe_StrictlyAfter(t *testing.T) {
	e, err := Parse("*/15 * * * *")
	if err != nil {
		t.Fatalf("Parse err: %v", err)
	}
	from, _ := time.Parse(time.RFC3339, "2026-08-14T12:00:00Z")
	got, err := e.Next(from)
	if err != nil {
		t.Fatalf("Next err: %v", err)
	}
	want, _ := time.Parse(time.RFC3339, "2026-08-14T12:15:00Z")
	if !got.Equal(want) {
		t.Errorf("Next(%s) = %s, want %s", from.Format(time.RFC3339), got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

// TestProbe_Dow7MapsToSunday 验证星期字段 7 被正确映射为 0（周日）。
// 解析 "0 0 * * 7" 后 Dow.Values() 应为 [0]（只有周日）。
func TestProbe_Dow7MapsToSunday(t *testing.T) {
	e, err := Parse("0 0 * * 7")
	if err != nil {
		t.Fatalf("Parse err: %v", err)
	}
	vals := e.Dow.Values()
	if len(vals) != 1 || vals[0] != 0 {
		t.Errorf("Dow.Values() = %v, want [0]", vals)
	}
}

// TestProbe_EqualRangeValid 验证区间起止相同的表达式（如 5-5）是合法的。
// "5-5 * * * *" 应该成功解析且 Minute.Values() = [5]。
func TestProbe_EqualRangeValid(t *testing.T) {
	e, err := Parse("5-5 * * * *")
	if err != nil {
		t.Errorf("Parse(\"5-5 * * * *\") 应成功但报错: %v", err)
		return
	}
	vals := e.Minute.Values()
	if len(vals) != 1 || vals[0] != 5 {
		t.Errorf("Minute.Values() = %v, want [5]", vals)
	}
}

// TestProbe_CrossMonthNext 验证 Next 能正确计算跨月的下次执行时间。
// "0 0 1 * *" 在 8 月 14 日调用 Next，期望返回 9 月 1 日。
func TestProbe_CrossMonthNext(t *testing.T) {
	e, err := Parse("0 0 1 * *")
	if err != nil {
		t.Fatalf("Parse err: %v", err)
	}
	from, _ := time.Parse(time.RFC3339, "2026-08-14T12:00:00Z")
	got, err := e.Next(from)
	if err != nil {
		t.Fatalf("Next err: %v", err)
	}
	want, _ := time.Parse(time.RFC3339, "2026-09-01T00:00:00Z")
	if !got.Equal(want) {
		t.Errorf("Next(%s) = %s, want %s", from.Format(time.RFC3339), got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}
