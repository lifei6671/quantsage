package job

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/indicator"
	"github.com/lifei6671/quantsage/apps/server/internal/domain/marketdata"
	"github.com/lifei6671/quantsage/apps/server/internal/domain/strategy"
)

type fakeStrategySignalReader struct {
	items []strategy.MarketContext
	err   error
}

func (r *fakeStrategySignalReader) ListStrategyContexts(ctx context.Context, startDate, endDate time.Time) ([]strategy.MarketContext, error) {
	if r.err != nil {
		return nil, r.err
	}

	return r.items, nil
}

type fakeStrategySignalWriter struct {
	items []strategy.SignalResult
	err   error
}

func (w *fakeStrategySignalWriter) UpsertStrategySignals(ctx context.Context, items []strategy.SignalResult) error {
	w.items = items
	return w.err
}

type fakeStrategySignalReplaceWriter struct {
	items     []strategy.SignalResult
	startDate time.Time
	endDate   time.Time
	err       error
}

func (w *fakeStrategySignalReplaceWriter) UpsertStrategySignals(ctx context.Context, items []strategy.SignalResult) error {
	w.items = items
	return nil
}

func (w *fakeStrategySignalReplaceWriter) ReplaceStrategySignals(ctx context.Context, startDate, endDate time.Time, items []strategy.SignalResult) error {
	w.startDate = startDate
	w.endDate = endDate
	w.items = items
	return w.err
}

func TestGenerateStrategySignals(t *testing.T) {
	t.Parallel()

	recorder := &fakeRecorder{}
	writer := &fakeStrategySignalWriter{}
	reader := &fakeStrategySignalReader{
		items: []strategy.MarketContext{
			buildStrategyContext("000001.SZ", "23", "420000", "5.2", "18", "16", "15", "180000", "0.05"),
			buildStrategyContext("000002.SZ", "14.5", "260000", "-4.5", "18", "16", "15", "180000", "0.10"),
		},
	}

	err := GenerateStrategySignals(
		context.Background(),
		recorder,
		reader,
		writer,
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("GenerateStrategySignals() error = %v", err)
	}
	if len(writer.items) != 2 {
		t.Fatalf("len(writer.items) = %d, want %d", len(writer.items), 2)
	}
	if writer.items[0].StrategyCode != strategy.StrategyCodeVolumeBreakout {
		t.Fatalf("writer.items[0].StrategyCode = %q, want %q", writer.items[0].StrategyCode, strategy.StrategyCodeVolumeBreakout)
	}
	if writer.items[1].StrategyCode != strategy.StrategyCodeTrendBreak {
		t.Fatalf("writer.items[1].StrategyCode = %q, want %q", writer.items[1].StrategyCode, strategy.StrategyCodeTrendBreak)
	}
}

func TestGenerateStrategySignalsFailure(t *testing.T) {
	t.Parallel()

	recorder := &fakeRecorder{}
	writer := &fakeStrategySignalWriter{}
	reader := &fakeStrategySignalReader{err: errors.New("reader down")}

	err := GenerateStrategySignals(
		context.Background(),
		recorder,
		reader,
		writer,
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("GenerateStrategySignals() error = nil, want non-nil")
	}
	if len(recorder.calls) != 2 {
		t.Fatalf("len(recorder.calls) = %d, want %d", len(recorder.calls), 2)
	}
	if recorder.calls[1].name != "fail:generate_strategy_signals" {
		t.Fatalf("recorder.calls[1].name = %q, want %q", recorder.calls[1].name, "fail:generate_strategy_signals")
	}
}

func TestGenerateStrategySignalsUsesReplaceWriter(t *testing.T) {
	t.Parallel()

	recorder := &fakeRecorder{}
	writer := &fakeStrategySignalReplaceWriter{}
	reader := &fakeStrategySignalReader{}

	startDate := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	bizDate := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)

	err := GenerateStrategySignals(context.Background(), recorder, reader, writer, startDate, endDate, bizDate)
	if err != nil {
		t.Fatalf("GenerateStrategySignals() error = %v", err)
	}
	if !writer.startDate.Equal(startDate) {
		t.Fatalf("writer.startDate = %s, want %s", writer.startDate, startDate)
	}
	if !writer.endDate.Equal(endDate) {
		t.Fatalf("writer.endDate = %s, want %s", writer.endDate, endDate)
	}
	if len(writer.items) != 0 {
		t.Fatalf("len(writer.items) = %d, want %d", len(writer.items), 0)
	}
}

func buildStrategyContext(tsCode, closePrice, volume, pctChg, ma20, ma10, ma5, volumeMA20, upperShadow string) strategy.MarketContext {
	recentBars := make([]marketdata.DailyBar, 0, 21)
	for i := 0; i < 20; i++ {
		recentBars = append(recentBars, marketdata.DailyBar{
			TSCode:    tsCode,
			TradeDate: time.Date(2026, 4, 1+i, 0, 0, 0, 0, time.UTC),
			High:      decimal.NewFromInt(int64(10 + i/2)),
			Close:     decimal.NewFromInt(int64(9 + i/2)),
			Vol:       decimal.RequireFromString("100000"),
		})
	}

	currentClose := decimal.RequireFromString(closePrice)
	currentVolume := decimal.RequireFromString(volume)
	currentPctChg := decimal.RequireFromString(pctChg)
	recentBars = append(recentBars, marketdata.DailyBar{
		TSCode:    tsCode,
		TradeDate: time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC),
		Open:      currentClose.Sub(decimal.RequireFromString("0.2")),
		High:      currentClose.Add(decimal.RequireFromString("0.5")),
		Low:       currentClose.Sub(decimal.RequireFromString("0.5")),
		Close:     currentClose,
		PctChg:    currentPctChg,
		Vol:       currentVolume,
	})

	ma20Value := decimal.RequireFromString(ma20)
	ma10Value := decimal.RequireFromString(ma10)
	ma5Value := decimal.RequireFromString(ma5)
	volumeMA20Value := decimal.RequireFromString(volumeMA20)
	upperShadowValue := decimal.RequireFromString(upperShadow)
	closeAboveMA5 := currentClose.GreaterThan(ma5Value)
	closeAboveMA10 := currentClose.GreaterThan(ma10Value)
	closeAboveMA20 := currentClose.GreaterThan(ma20Value)

	return strategy.MarketContext{
		CurrentBar: recentBars[len(recentBars)-1],
		CurrentFactor: indicator.DailyFactor{
			TSCode:           tsCode,
			TradeDate:        recentBars[len(recentBars)-1].TradeDate,
			MA5:              &ma5Value,
			MA10:             &ma10Value,
			MA20:             &ma20Value,
			VolumeMA20:       &volumeMA20Value,
			UpperShadowRatio: &upperShadowValue,
			CloseAboveMA5:    &closeAboveMA5,
			CloseAboveMA10:   &closeAboveMA10,
			CloseAboveMA20:   &closeAboveMA20,
		},
		RecentBars: recentBars,
	}
}
