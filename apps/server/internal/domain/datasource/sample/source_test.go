package sample

import (
	"context"
	"testing"
	"time"
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
