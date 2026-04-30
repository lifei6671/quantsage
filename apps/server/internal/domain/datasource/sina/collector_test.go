package sina

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
)

func TestCollectorFinalizesLatestUniqueItemsByLimit(t *testing.T) {
	t.Parallel()

	collector := newCollector(datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.Interval5Min,
		EndTime:  time.Date(2026, 4, 29, 15, 0, 0, 0, time.UTC),
		Limit:    2,
	})

	collector.Add([]datasource.KLine{
		{TSCode: "000001.SZ", TradeTime: time.Date(2026, 4, 29, 9, 30, 0, 0, time.UTC)},
		{TSCode: "000001.SZ", TradeTime: time.Date(2026, 4, 29, 9, 35, 0, 0, time.UTC)},
		{TSCode: "000001.SZ", TradeTime: time.Date(2026, 4, 29, 9, 35, 0, 0, time.UTC)},
		{TSCode: "000001.SZ", TradeTime: time.Date(2026, 4, 29, 9, 40, 0, 0, time.UTC)},
	})

	items := collector.Finalize()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want %d", len(items), 2)
	}
	if !items[0].TradeTime.Before(items[1].TradeTime) {
		t.Fatalf("items not sorted ascending: %+v", items)
	}
	if got := items[0].TradeTime; !got.Equal(time.Date(2026, 4, 29, 9, 35, 0, 0, time.UTC)) {
		t.Fatalf("items[0].TradeTime = %s, want %s", got, time.Date(2026, 4, 29, 9, 35, 0, 0, time.UTC))
	}
}

func TestCollectorFinalizesTimeWindow(t *testing.T) {
	t.Parallel()

	collector := newCollector(datasource.KLineQuery{
		TSCode:    "000001.SZ",
		Interval:  datasource.Interval5Min,
		StartTime: time.Date(2026, 4, 29, 9, 35, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 4, 29, 9, 40, 0, 0, time.UTC),
	})

	collector.Add([]datasource.KLine{
		{TSCode: "000001.SZ", TradeTime: time.Date(2026, 4, 29, 9, 30, 0, 0, time.UTC)},
		{TSCode: "000001.SZ", TradeTime: time.Date(2026, 4, 29, 9, 35, 0, 0, time.UTC)},
		{TSCode: "000001.SZ", TradeTime: time.Date(2026, 4, 29, 9, 40, 0, 0, time.UTC)},
	})

	items := collector.Finalize()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want %d", len(items), 2)
	}
	if got := items[0].TradeTime; !got.Equal(time.Date(2026, 4, 29, 9, 35, 0, 0, time.UTC)) {
		t.Fatalf("items[0].TradeTime = %s, want %s", got, time.Date(2026, 4, 29, 9, 35, 0, 0, time.UTC))
	}
}

func TestCollectorPrefersLaterSnapshotForSameTradeTime(t *testing.T) {
	t.Parallel()

	collector := newCollector(datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.Interval5Min,
		EndTime:  time.Date(2026, 4, 29, 15, 0, 0, 0, time.UTC),
		Limit:    1,
	})

	tradeTime := time.Date(2026, 4, 29, 9, 35, 0, 0, time.UTC)
	collector.Add([]datasource.KLine{
		{TSCode: "000001.SZ", TradeTime: tradeTime},
	})
	collector.Add([]datasource.KLine{
		{
			TSCode:    "000001.SZ",
			TradeTime: tradeTime,
			Close:     mustDecimal("10.30"),
			Vol:       mustDecimal("1200"),
		},
	})

	items := collector.Finalize()
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
	if !items[0].Close.Equal(mustDecimal("10.30")) {
		t.Fatalf("items[0].Close = %s, want 10.30", items[0].Close)
	}
	if !items[0].Vol.Equal(mustDecimal("1200")) {
		t.Fatalf("items[0].Vol = %s, want 1200", items[0].Vol)
	}
}

func mustDecimal(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}
