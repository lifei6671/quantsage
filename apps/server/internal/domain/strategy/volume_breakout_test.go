package strategy

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/indicator"
	"github.com/lifei6671/quantsage/apps/server/internal/domain/marketdata"
)

func TestEvaluateVolumeBreakoutHit(t *testing.T) {
	t.Parallel()

	input := buildVolumeBreakoutContext(
		"23.0",
		"420000",
		"5.2",
		"15.0",
		"16.0",
		"18.0",
		"180000",
		"0.05",
	)

	result, hit, err := EvaluateVolumeBreakout(input)
	if err != nil {
		t.Fatalf("EvaluateVolumeBreakout() error = %v", err)
	}
	if !hit {
		t.Fatal("EvaluateVolumeBreakout() hit = false, want true")
	}
	if result.SignalLevel != "A" {
		t.Fatalf("result.SignalLevel = %q, want %q", result.SignalLevel, "A")
	}
	if result.SignalStrength.String() != "100" {
		t.Fatalf("result.SignalStrength = %s, want %s", result.SignalStrength.String(), "100")
	}
	if result.StrategyCode != StrategyCodeVolumeBreakout {
		t.Fatalf("result.StrategyCode = %q, want %q", result.StrategyCode, StrategyCodeVolumeBreakout)
	}
}

func TestEvaluateVolumeBreakoutMiss(t *testing.T) {
	t.Parallel()

	input := buildVolumeBreakoutContext(
		"23.0",
		"200000",
		"5.2",
		"18.0",
		"16.0",
		"15.0",
		"180000",
		"0.05",
	)

	result, hit, err := EvaluateVolumeBreakout(input)
	if err != nil {
		t.Fatalf("EvaluateVolumeBreakout() error = %v", err)
	}
	if hit {
		t.Fatalf("EvaluateVolumeBreakout() hit = true, want false, result = %+v", result)
	}
}

func TestEvaluateVolumeBreakoutHistoryInsufficient(t *testing.T) {
	t.Parallel()

	input := buildVolumeBreakoutContext(
		"23.0",
		"420000",
		"5.2",
		"18.0",
		"16.0",
		"15.0",
		"180000",
		"0.05",
	)
	input.RecentBars = input.RecentBars[len(input.RecentBars)-20:]

	result, hit, err := EvaluateVolumeBreakout(input)
	if err != nil {
		t.Fatalf("EvaluateVolumeBreakout() error = %v", err)
	}
	if hit {
		t.Fatalf("EvaluateVolumeBreakout() hit = true, want false, result = %+v", result)
	}
}

func TestSignalLevelFromScoreBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		score string
		want  string
	}{
		{score: "80", want: "A"},
		{score: "79.99", want: "B"},
		{score: "60", want: "B"},
		{score: "59.99", want: "C"},
		{score: "40", want: "C"},
		{score: "39.99", want: "D"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.score, func(t *testing.T) {
			t.Parallel()
			score := decimal.RequireFromString(tc.score)
			if got := SignalLevelFromScore(score); got != tc.want {
				t.Fatalf("SignalLevelFromScore(%s) = %q, want %q", score.String(), got, tc.want)
			}
		})
	}
}

func TestEvaluateTrendBreakHit(t *testing.T) {
	t.Parallel()

	input := buildVolumeBreakoutContext(
		"14.5",
		"260000",
		"-4.5",
		"18.0",
		"16.0",
		"15.0",
		"180000",
		"0.10",
	)

	result, hit, err := EvaluateTrendBreak(input)
	if err != nil {
		t.Fatalf("EvaluateTrendBreak() error = %v", err)
	}
	if !hit {
		t.Fatal("EvaluateTrendBreak() hit = false, want true")
	}
	if result.SignalLevel != "A" {
		t.Fatalf("result.SignalLevel = %q, want %q", result.SignalLevel, "A")
	}
}

func buildVolumeBreakoutContext(closePrice, volume, pctChg, ma20, ma10, ma5, volumeMA20, upperShadow string) MarketContext {
	recentBars := make([]marketdata.DailyBar, 0, 21)
	for i := 0; i < 20; i++ {
		recentBars = append(recentBars, marketdata.DailyBar{
			TSCode:    "000001.SZ",
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
		TSCode:    "000001.SZ",
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
	closeAboveMA5 := true
	closeAboveMA10 := true
	closeAboveMA20 := currentClose.GreaterThan(ma20Value)

	return MarketContext{
		CurrentBar: recentBars[len(recentBars)-1],
		CurrentFactor: indicator.DailyFactor{
			TSCode:           "000001.SZ",
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
