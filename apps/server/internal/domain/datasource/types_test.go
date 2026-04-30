package datasource

import (
	"testing"
	"time"
)

func TestNormalizeKLineQueryPreservesCallerCalendarDateForDayInterval(t *testing.T) {
	t.Parallel()

	cst := time.FixedZone("CST", 8*3600)
	got, err := NormalizeKLineQuery(KLineQuery{
		TSCode:    "000001.SZ",
		Interval:  IntervalDay,
		StartTime: time.Date(2026, 4, 29, 0, 0, 0, 0, cst),
		EndTime:   time.Date(2026, 4, 29, 0, 0, 0, 0, cst),
	}, nil)
	if err != nil {
		t.Fatalf("NormalizeKLineQuery() error = %v", err)
	}

	want := time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC)
	if !got.StartTime.Equal(want) {
		t.Fatalf("StartTime = %s, want %s", got.StartTime, want)
	}
	if !got.EndTime.Equal(want) {
		t.Fatalf("EndTime = %s, want %s", got.EndTime, want)
	}
}

func TestNormalizeKLineQueryKeepsMinuteBoundaryAsInstant(t *testing.T) {
	t.Parallel()

	cst := time.FixedZone("CST", 8*3600)
	got, err := NormalizeKLineQuery(KLineQuery{
		TSCode:   "000001.SZ",
		Interval: Interval5Min,
		EndTime:  time.Date(2026, 4, 29, 9, 35, 0, 0, cst),
		Limit:    1,
	}, nil)
	if err != nil {
		t.Fatalf("NormalizeKLineQuery() error = %v", err)
	}

	want := time.Date(2026, 4, 29, 1, 35, 0, 0, time.UTC)
	if !got.EndTime.Equal(want) {
		t.Fatalf("EndTime = %s, want %s", got.EndTime, want)
	}
}

func TestNormalizeKLineQueryUsesNowCalendarDateForLatestDayInterval(t *testing.T) {
	t.Parallel()

	cst := time.FixedZone("CST", 8*3600)
	got, err := NormalizeKLineQuery(KLineQuery{
		TSCode:   "000001.SZ",
		Interval: IntervalDay,
		Limit:    1,
	}, func() time.Time {
		return time.Date(2026, 4, 29, 15, 30, 0, 0, cst)
	})
	if err != nil {
		t.Fatalf("NormalizeKLineQuery() error = %v", err)
	}

	want := time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC)
	if !got.EndTime.Equal(want) {
		t.Fatalf("EndTime = %s, want %s", got.EndTime, want)
	}
}

func TestNormalizeKLineQueryUsesChinaMarketDateForLatestDayInterval(t *testing.T) {
	t.Parallel()

	got, err := NormalizeKLineQuery(KLineQuery{
		TSCode:   "000001.SZ",
		Interval: IntervalDay,
		Limit:    1,
	}, func() time.Time {
		return time.Date(2026, 4, 28, 20, 30, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("NormalizeKLineQuery() error = %v", err)
	}

	want := time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC)
	if !got.EndTime.Equal(want) {
		t.Fatalf("EndTime = %s, want %s", got.EndTime, want)
	}
}
