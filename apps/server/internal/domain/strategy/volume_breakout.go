package strategy

import (
	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/indicator"
	"github.com/lifei6671/quantsage/apps/server/internal/domain/marketdata"
)

var (
	decimalZero          = decimal.Zero
	decimalOnePointEight = decimal.RequireFromString("1.8")
	decimalThree         = decimal.NewFromInt(3)
	decimalFive          = decimal.NewFromInt(5)
	decimalTwelvePct     = decimal.RequireFromString("1.12")
)

// EvaluateVolumeBreakout 评估放量突破策略。
func EvaluateVolumeBreakout(input MarketContext) (*SignalResult, bool, error) {
	breakoutPrice, ok := highestHighExcludeToday(input.RecentBars, 20)
	if !ok {
		return nil, false, nil
	}

	if input.CurrentFactor.MA20 == nil || input.CurrentFactor.MA10 == nil || input.CurrentFactor.VolumeMA20 == nil || input.CurrentFactor.UpperShadowRatio == nil {
		return nil, false, nil
	}

	if !input.CurrentBar.Close.GreaterThan(breakoutPrice) {
		return nil, false, nil
	}
	if !input.CurrentBar.Vol.GreaterThan(input.CurrentFactor.VolumeMA20.Mul(decimalOnePointEight)) {
		return nil, false, nil
	}
	if !input.CurrentBar.PctChg.GreaterThan(decimalThree) {
		return nil, false, nil
	}
	if !input.CurrentBar.Close.GreaterThan(*input.CurrentFactor.MA20) {
		return nil, false, nil
	}
	if !input.CurrentFactor.UpperShadowRatio.LessThan(decimal.RequireFromString("0.25")) {
		return nil, false, nil
	}

	score := decimal.NewFromInt(40)
	if allCloseAboveMAs(input.CurrentFactor) {
		score = score.Add(decimal.NewFromInt(15))
	}
	if maBullish(input.CurrentFactor) {
		score = score.Add(decimal.NewFromInt(15))
	}
	if input.CurrentBar.Vol.GreaterThan(input.CurrentFactor.VolumeMA20.Mul(decimal.RequireFromString("2.2"))) {
		score = score.Add(decimal.NewFromInt(10))
	}
	if input.CurrentBar.PctChg.GreaterThanOrEqual(decimalFive) {
		score = score.Add(decimal.NewFromInt(10))
	}
	if input.CurrentFactor.UpperShadowRatio.LessThan(decimal.RequireFromString("0.10")) {
		score = score.Add(decimal.NewFromInt(10))
	}

	stopLoss := minDecimal(breakoutPrice, *input.CurrentFactor.MA10).Round(4)
	takeProfit := input.CurrentBar.Close.Mul(decimalTwelvePct).Round(4)

	return &SignalResult{
		StrategyCode:          StrategyCodeVolumeBreakout,
		StrategyVersion:       StrategyVersionV1,
		TSCode:                input.CurrentBar.TSCode,
		TradeDate:             input.CurrentBar.TradeDate,
		SignalType:            "buy_signal",
		SignalStrength:        score.Round(4),
		SignalLevel:           SignalLevelFromScore(score),
		BuyPriceRef:           input.CurrentBar.Close.Round(4),
		StopLossRef:           stopLoss,
		TakeProfitRef:         takeProfit,
		InvalidationCondition: "close < ma20 or close < breakout_price",
		Reason:                "放量突破 20 日新高",
		InputSnapshot: map[string]any{
			"breakout_price":      breakoutPrice.String(),
			"close":               input.CurrentBar.Close.String(),
			"volume":              input.CurrentBar.Vol.String(),
			"volume_ma20":         input.CurrentFactor.VolumeMA20.String(),
			"pct_chg":             input.CurrentBar.PctChg.String(),
			"upper_shadow_ratio":  input.CurrentFactor.UpperShadowRatio.String(),
			"close_above_ma5":     boolPtrValue(input.CurrentFactor.CloseAboveMA5),
			"close_above_ma10":    boolPtrValue(input.CurrentFactor.CloseAboveMA10),
			"close_above_ma20":    boolPtrValue(input.CurrentFactor.CloseAboveMA20),
			"ma_bullish_arranged": maBullish(input.CurrentFactor),
		},
	}, true, nil
}

func highestHighExcludeToday(bars []marketdata.DailyBar, window int) (decimal.Decimal, bool) {
	if len(bars) < window+1 {
		return decimalZero, false
	}

	current := bars[len(bars)-1]
	start := len(bars) - window - 1
	highest := bars[start].High
	for i := start; i < len(bars)-1; i++ {
		if bars[i].TradeDate.After(current.TradeDate) {
			return decimalZero, false
		}
		if bars[i].High.GreaterThan(highest) {
			highest = bars[i].High
		}
	}

	return highest, true
}

func allCloseAboveMAs(factor indicator.DailyFactor) bool {
	return boolPtrValue(factor.CloseAboveMA5) && boolPtrValue(factor.CloseAboveMA10) && boolPtrValue(factor.CloseAboveMA20)
}

func maBullish(factor indicator.DailyFactor) bool {
	if factor.MA5 == nil || factor.MA10 == nil || factor.MA20 == nil {
		return false
	}

	return factor.MA5.GreaterThan(*factor.MA10) && factor.MA10.GreaterThan(*factor.MA20)
}

func boolPtrValue(value *bool) bool {
	return value != nil && *value
}

func minDecimal(left, right decimal.Decimal) decimal.Decimal {
	if left.LessThan(right) {
		return left
	}

	return right
}
