package job

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
)

type recorderCall struct {
	name string
}

type fakeRecorder struct {
	calls []recorderCall
}

func (r *fakeRecorder) Start(ctx context.Context, jobName string, bizDate time.Time) error {
	r.calls = append(r.calls, recorderCall{name: "start:" + jobName})
	return nil
}

func (r *fakeRecorder) Success(ctx context.Context, jobName string, bizDate time.Time) error {
	r.calls = append(r.calls, recorderCall{name: "success:" + jobName})
	return nil
}

func (r *fakeRecorder) Fail(ctx context.Context, jobName string, bizDate time.Time, err error) error {
	r.calls = append(r.calls, recorderCall{name: "fail:" + jobName})
	return nil
}

type fakeSource struct {
	stocks   []datasource.StockBasic
	days     []datasource.TradeDay
	bars     []datasource.DailyBar
	stockErr error
	dayErr   error
	barErr   error
}

func (s *fakeSource) ListStocks(ctx context.Context) ([]datasource.StockBasic, error) {
	if s.stockErr != nil {
		return nil, s.stockErr
	}
	return s.stocks, nil
}

func (s *fakeSource) ListTradeCalendar(ctx context.Context, exchange string, startDate, endDate time.Time) ([]datasource.TradeDay, error) {
	if s.dayErr != nil {
		return nil, s.dayErr
	}
	return s.days, nil
}

func (s *fakeSource) ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]datasource.DailyBar, error) {
	if s.barErr != nil {
		return nil, s.barErr
	}
	return s.bars, nil
}

type fakeStockBasicWriter struct {
	items []datasource.StockBasic
	err   error
}

func (w *fakeStockBasicWriter) UpsertStockBasics(ctx context.Context, items []datasource.StockBasic) error {
	w.items = items
	return w.err
}

type fakeTradeCalendarWriter struct {
	items []datasource.TradeDay
	err   error
}

func (w *fakeTradeCalendarWriter) UpsertTradeCalendar(ctx context.Context, items []datasource.TradeDay) error {
	w.items = items
	return w.err
}

type fakeStockDailyWriter struct {
	items []datasource.DailyBar
	err   error
}

func (w *fakeStockDailyWriter) UpsertStockDaily(ctx context.Context, items []datasource.DailyBar) error {
	w.items = items
	return w.err
}

func TestImportStockBasic(t *testing.T) {
	t.Parallel()

	recorder := &fakeRecorder{}
	writer := &fakeStockBasicWriter{}
	source := &fakeSource{
		stocks: []datasource.StockBasic{{TSCode: "000001.SZ", Name: "平安银行"}},
	}

	if err := ImportStockBasic(context.Background(), recorder, writer, source, time.Now()); err != nil {
		t.Fatalf("ImportStockBasic() error = %v", err)
	}
	if len(writer.items) != 1 {
		t.Fatalf("len(writer.items) = %d, want %d", len(writer.items), 1)
	}
	if len(recorder.calls) != 2 {
		t.Fatalf("len(recorder.calls) = %d, want %d", len(recorder.calls), 2)
	}
}

func TestImportTradeCalendarFailure(t *testing.T) {
	t.Parallel()

	recorder := &fakeRecorder{}
	writer := &fakeTradeCalendarWriter{}
	source := &fakeSource{
		dayErr: errors.New("calendar down"),
	}

	err := ImportTradeCalendar(context.Background(), recorder, writer, source, "SSE", time.Now(), time.Now(), time.Now())
	if err == nil {
		t.Fatal("ImportTradeCalendar() error = nil, want non-nil")
	}
	if len(recorder.calls) != 2 {
		t.Fatalf("len(recorder.calls) = %d, want %d", len(recorder.calls), 2)
	}
	if recorder.calls[1].name != "fail:sync_trade_calendar" {
		t.Fatalf("recorder.calls[1].name = %q, want %q", recorder.calls[1].name, "fail:sync_trade_calendar")
	}
}

func TestImportStockDaily(t *testing.T) {
	t.Parallel()

	recorder := &fakeRecorder{}
	writer := &fakeStockDailyWriter{}
	source := &fakeSource{
		bars: []datasource.DailyBar{{
			TSCode:    "000001.SZ",
			TradeDate: time.Now(),
			Open:      decimal.RequireFromString("10.10"),
			High:      decimal.RequireFromString("10.50"),
			Low:       decimal.RequireFromString("10.00"),
			Close:     decimal.RequireFromString("10.40"),
			PreClose:  decimal.RequireFromString("10.00"),
			Change:    decimal.RequireFromString("0.40"),
			PctChg:    decimal.RequireFromString("4.00"),
			Vol:       decimal.RequireFromString("100000"),
			Amount:    decimal.RequireFromString("1000000"),
			Source:    "sample",
		}},
	}

	if err := ImportStockDaily(context.Background(), recorder, writer, source, time.Now(), time.Now(), time.Now()); err != nil {
		t.Fatalf("ImportStockDaily() error = %v", err)
	}
	if len(writer.items) != 1 {
		t.Fatalf("len(writer.items) = %d, want %d", len(writer.items), 1)
	}
	if len(recorder.calls) != 2 {
		t.Fatalf("len(recorder.calls) = %d, want %d", len(recorder.calls), 2)
	}
}
