package job

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/indicator"
	"github.com/lifei6671/quantsage/apps/server/internal/domain/marketdata"
)

type fakeDailyBarReader struct {
	items []marketdata.DailyBar
	err   error
}

func (r *fakeDailyBarReader) ListStockDaily(ctx context.Context, startDate, endDate time.Time) ([]marketdata.DailyBar, error) {
	if r.err != nil {
		return nil, r.err
	}

	return r.items, nil
}

type fakeDailyFactorWriter struct {
	items []indicator.DailyFactor
	err   error
}

func (w *fakeDailyFactorWriter) UpsertDailyFactors(ctx context.Context, items []indicator.DailyFactor) error {
	w.items = items
	return w.err
}

func TestCalcDailyFactor(t *testing.T) {
	t.Parallel()

	recorder := &fakeRecorder{}
	writer := &fakeDailyFactorWriter{}
	reader := &fakeDailyBarReader{
		items: []marketdata.DailyBar{
			buildMarketBar("000002.SZ", "2026-04-28", "22", "220"),
			buildMarketBar("000001.SZ", "2026-04-27", "10", "100"),
			buildMarketBar("000001.SZ", "2026-04-28", "11", "110"),
			buildMarketBar("000002.SZ", "2026-04-27", "21", "210"),
		},
	}

	err := CalcDailyFactor(
		context.Background(),
		recorder,
		reader,
		writer,
		time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("CalcDailyFactor() error = %v", err)
	}
	if len(writer.items) != 4 {
		t.Fatalf("len(writer.items) = %d, want %d", len(writer.items), 4)
	}
	if writer.items[0].TSCode != "000001.SZ" || writer.items[1].TSCode != "000001.SZ" {
		t.Fatalf("writer.items[:2] ts_code = %q, %q, want grouped 000001.SZ", writer.items[0].TSCode, writer.items[1].TSCode)
	}
	if len(recorder.calls) != 2 {
		t.Fatalf("len(recorder.calls) = %d, want %d", len(recorder.calls), 2)
	}
	if recorder.calls[1].name != "success:calc_daily_factor" {
		t.Fatalf("recorder.calls[1].name = %q, want %q", recorder.calls[1].name, "success:calc_daily_factor")
	}
}

func TestCalcDailyFactorFailure(t *testing.T) {
	t.Parallel()

	recorder := &fakeRecorder{}
	writer := &fakeDailyFactorWriter{}
	reader := &fakeDailyBarReader{err: errors.New("reader down")}

	err := CalcDailyFactor(
		context.Background(),
		recorder,
		reader,
		writer,
		time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("CalcDailyFactor() error = nil, want non-nil")
	}
	if len(recorder.calls) != 2 {
		t.Fatalf("len(recorder.calls) = %d, want %d", len(recorder.calls), 2)
	}
	if recorder.calls[1].name != "fail:calc_daily_factor" {
		t.Fatalf("recorder.calls[1].name = %q, want %q", recorder.calls[1].name, "fail:calc_daily_factor")
	}
}

func buildMarketBar(tsCode, tradeDate, closePrice, volume string) marketdata.DailyBar {
	dateValue, err := time.Parse("2006-01-02", tradeDate)
	if err != nil {
		panic(err)
	}

	closeValue := decimal.RequireFromString(closePrice)
	return marketdata.DailyBar{
		TSCode:    tsCode,
		TradeDate: dateValue,
		Open:      closeValue.Sub(decimal.RequireFromString("0.2")),
		High:      closeValue.Add(decimal.RequireFromString("0.5")),
		Low:       closeValue.Sub(decimal.RequireFromString("0.5")),
		Close:     closeValue,
		PreClose:  closeValue.Sub(decimal.RequireFromString("0.1")),
		Change:    decimal.RequireFromString("0.1"),
		PctChg:    decimal.RequireFromString("1"),
		Vol:       decimal.RequireFromString(volume),
		Amount:    decimal.RequireFromString("1000000"),
		Source:    "sample",
	}
}
