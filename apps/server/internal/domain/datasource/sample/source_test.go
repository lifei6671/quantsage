package sample

import (
	"context"
	"testing"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
)

func TestSourceListStocks(t *testing.T) {
	t.Parallel()

	source := New("../../../../testdata/sample")
	items, err := source.ListStocks(context.Background())
	if err != nil {
		t.Fatalf("ListStocks() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want %d", len(items), 3)
	}
	if items[0].TSCode != "000001.SZ" {
		t.Fatalf("items[0].TSCode = %q, want %q", items[0].TSCode, "000001.SZ")
	}
}

func TestSourceListTradeCalendar(t *testing.T) {
	t.Parallel()

	source := New("../../../../testdata/sample")
	startDate := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)
	items, err := source.ListTradeCalendar(context.Background(), "SSE", startDate, endDate)
	if err != nil {
		t.Fatalf("ListTradeCalendar() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want %d", len(items), 2)
	}
}

func TestSourceListDailyBars(t *testing.T) {
	t.Parallel()

	source := New("../../../../testdata/sample")
	startDate := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)
	items, err := source.ListDailyBars(context.Background(), startDate, endDate)
	if err != nil {
		t.Fatalf("ListDailyBars() error = %v", err)
	}
	if len(items) != 6 {
		t.Fatalf("len(items) = %d, want %d", len(items), 6)
	}
}

func TestSourceListDailyBarsUsesTradeDateGranularity(t *testing.T) {
	t.Parallel()

	source := New("../../../../testdata/sample")
	startDate := time.Date(2026, 4, 27, 18, 5, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 27, 18, 5, 0, 0, time.UTC)
	items, err := source.ListDailyBars(context.Background(), startDate, endDate)
	if err != nil {
		t.Fatalf("ListDailyBars() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want %d", len(items), 3)
	}
}

func TestSourceListTradeCalendarUsesTradeDateGranularity(t *testing.T) {
	t.Parallel()

	source := New("../../../../testdata/sample")
	startDate := time.Date(2026, 4, 27, 8, 5, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 27, 8, 5, 0, 0, time.UTC)
	items, err := source.ListTradeCalendar(context.Background(), "SSE", startDate, endDate)
	if err != nil {
		t.Fatalf("ListTradeCalendar() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
}

func TestSourceListKLinesSupportsDayInterval(t *testing.T) {
	t.Parallel()

	source := New("../../../../testdata/sample")
	endDate := time.Date(2026, 4, 16, 15, 0, 0, 0, time.UTC)

	items, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.IntervalDay,
		EndTime:  endDate,
		Limit:    2,
	})
	if err != nil {
		t.Fatalf("ListKLines() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want %d", len(items), 2)
	}
	if items[0].TSCode != "000001.SZ" {
		t.Fatalf("items[0].TSCode = %q, want %q", items[0].TSCode, "000001.SZ")
	}
	if !items[0].TradeTime.Before(items[1].TradeTime) {
		t.Fatalf("items not sorted ascending: %+v", items)
	}
}

func TestSourceListKLinesPreservesLocalCalendarDateRange(t *testing.T) {
	t.Parallel()

	source := New("../../../../testdata/sample")
	cst := time.FixedZone("CST", 8*3600)

	items, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:    "000001.SZ",
		Interval:  datasource.IntervalDay,
		StartTime: time.Date(2026, 4, 16, 0, 0, 0, 0, cst),
		EndTime:   time.Date(2026, 4, 16, 0, 0, 0, 0, cst),
	})
	if err != nil {
		t.Fatalf("ListKLines() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
	if got := items[0].TradeTime; !got.Equal(time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("items[0].TradeTime = %s, want %s", got, time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC))
	}
}

func TestSourceListKLinesRejectsUnsupportedInterval(t *testing.T) {
	t.Parallel()

	source := New("../../../../testdata/sample")

	_, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.Interval5Min,
		Limit:    10,
	})
	if err == nil {
		t.Fatal("ListKLines() error = nil, want non-nil")
	}
}

func TestSourceStreamKLinesReturnsUnsupported(t *testing.T) {
	t.Parallel()

	source := New("../../../../testdata/sample")
	_, err := source.StreamKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.IntervalDay,
		Limit:    1,
	})
	if err == nil {
		t.Fatal("StreamKLines() error = nil, want non-nil")
	}
}
