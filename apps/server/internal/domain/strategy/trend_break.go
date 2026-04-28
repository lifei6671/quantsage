package strategy

import "github.com/shopspring/decimal"

var decimalOnePointTwo = decimal.RequireFromString("1.2")

// EvaluateTrendBreak 评估趋势破位策略。
func EvaluateTrendBreak(input MarketContext) (*SignalResult, bool, error) {
	if input.CurrentFactor.MA20 == nil || input.CurrentFactor.VolumeMA20 == nil {
		return nil, false, nil
	}

	if !input.CurrentBar.Close.LessThan(*input.CurrentFactor.MA20) {
		return nil, false, nil
	}
	if !input.CurrentBar.Vol.GreaterThan(input.CurrentFactor.VolumeMA20.Mul(decimalOnePointTwo)) {
		return nil, false, nil
	}

	score := decimal.NewFromInt(60)
	if input.CurrentFactor.MA10 != nil && input.CurrentBar.Close.LessThan(*input.CurrentFactor.MA10) {
		score = score.Add(decimal.NewFromInt(20))
	}
	if input.CurrentBar.PctChg.LessThan(decimal.NewFromInt(-3)) {
		score = score.Add(decimal.NewFromInt(20))
	}

	return &SignalResult{
		StrategyCode:          StrategyCodeTrendBreak,
		StrategyVersion:       StrategyVersionV1,
		TSCode:                input.CurrentBar.TSCode,
		TradeDate:             input.CurrentBar.TradeDate,
		SignalType:            "sell_signal",
		SignalStrength:        score.Round(4),
		SignalLevel:           SignalLevelFromScore(score),
		BuyPriceRef:           input.CurrentBar.Close.Round(4),
		StopLossRef:           input.CurrentFactor.MA20.Round(4),
		TakeProfitRef:         input.CurrentBar.Close.Mul(decimal.RequireFromString("0.90")).Round(4),
		InvalidationCondition: "close >= ma20",
		Reason:                "跌破 20 日线且放量",
		InputSnapshot: map[string]any{
			"close":       input.CurrentBar.Close.String(),
			"ma20":        input.CurrentFactor.MA20.String(),
			"volume":      input.CurrentBar.Vol.String(),
			"volume_ma20": input.CurrentFactor.VolumeMA20.String(),
			"pct_chg":     input.CurrentBar.PctChg.String(),
		},
	}, true, nil
}
