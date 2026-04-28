package indicator

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/marketdata"
)

var (
	errBarsNotSorted = errors.New("daily bars must be sorted by trade date ascending")
)

var (
	decimalZero = decimal.Zero
	decimalOne  = decimal.NewFromInt(1)
	decimalTwo  = decimal.NewFromInt(2)
)

// CalculateDailyFactors 根据升序日线数据计算每日因子。
func CalculateDailyFactors(bars []marketdata.DailyBar) ([]DailyFactor, error) {
	if err := validateBars(bars); err != nil {
		return nil, err
	}
	if len(bars) == 0 {
		return []DailyFactor{}, nil
	}

	closes := make([]decimal.Decimal, len(bars))
	volumes := make([]decimal.Decimal, len(bars))
	for i, bar := range bars {
		closes[i] = bar.Close
		volumes[i] = bar.Vol
	}

	ema12Series := calculateEMA(closes, 12)
	ema26Series := calculateEMA(closes, 26)
	difSeries := make([]decimal.Decimal, len(bars))
	for i := range bars {
		difSeries[i] = ema12Series[i].Sub(ema26Series[i])
	}
	deaSeries := calculateEMA(difSeries, 9)

	results := make([]DailyFactor, 0, len(bars))
	for i, bar := range bars {
		factor := DailyFactor{
			TSCode:    bar.TSCode,
			TradeDate: bar.TradeDate,
			EMA12:     decimalPtr(ema12Series[i].Round(6)),
			EMA26:     decimalPtr(ema26Series[i].Round(6)),
			MACDDIF:   decimalPtr(difSeries[i].Round(6)),
			MACDDEA:   decimalPtr(deaSeries[i].Round(6)),
			// MACD 柱值按 A 股常用口径使用 2 * (DIF - DEA)。
			MACDHist: decimalPtr(difSeries[i].Sub(deaSeries[i]).Mul(decimalTwo).Round(6)),
		}

		factor.MA5 = movingAverage(closes, i, 5, 4)
		factor.MA10 = movingAverage(closes, i, 10, 4)
		factor.MA20 = movingAverage(closes, i, 20, 4)
		factor.MA60 = movingAverage(closes, i, 60, 4)

		factor.RSI6 = calculateRSI(closes, i, 6)
		factor.RSI12 = calculateRSI(closes, i, 12)

		factor.VolumeMA5 = movingAverage(volumes, i, 5, 4)
		factor.VolumeMA20 = movingAverage(volumes, i, 20, 4)
		// V1 明确采用 5 日均量作为 volume_ratio 的分母。
		factor.VolumeRatio = divideDecimalPtr(bar.Vol, factor.VolumeMA5, 4)

		factor.UpperShadowRatio, factor.LowerShadowRatio = calculateShadowRatio(bar)
		factor.CloseAboveMA5 = compareCloseAbove(bar.Close, factor.MA5)
		factor.CloseAboveMA10 = compareCloseAbove(bar.Close, factor.MA10)
		factor.CloseAboveMA20 = compareCloseAbove(bar.Close, factor.MA20)

		results = append(results, factor)
	}

	return results, nil
}

func validateBars(bars []marketdata.DailyBar) error {
	for i := 1; i < len(bars); i++ {
		if bars[i].TradeDate.Before(bars[i-1].TradeDate) {
			return fmt.Errorf("validate bars order at index %d: %w", i, errBarsNotSorted)
		}
	}

	return nil
}

func movingAverage(values []decimal.Decimal, index, window int, scale int32) *decimal.Decimal {
	if window <= 0 || index < 0 || index+1 < window {
		return nil
	}

	sum := decimalZero
	start := index - window + 1
	for i := start; i <= index; i++ {
		sum = sum.Add(values[i])
	}

	avg := sum.Div(decimal.NewFromInt(int64(window))).Round(scale)
	return decimalPtr(avg)
}

func calculateEMA(values []decimal.Decimal, window int64) []decimal.Decimal {
	if len(values) == 0 {
		return []decimal.Decimal{}
	}

	alpha := decimalTwo.Div(decimal.NewFromInt(window + 1))
	base := decimalOne.Sub(alpha)

	result := make([]decimal.Decimal, len(values))
	result[0] = values[0]
	for i := 1; i < len(values); i++ {
		result[i] = values[i].Mul(alpha).Add(result[i-1].Mul(base))
	}

	return result
}

func calculateRSI(closes []decimal.Decimal, index, window int) *decimal.Decimal {
	if index < window || window <= 0 {
		return nil
	}

	gainSum := decimalZero
	lossSum := decimalZero
	for i := index - window + 1; i <= index; i++ {
		change := closes[i].Sub(closes[i-1])
		switch change.Cmp(decimalZero) {
		case 1:
			gainSum = gainSum.Add(change)
		case -1:
			lossSum = lossSum.Add(change.Abs())
		}
	}

	avgLoss := lossSum.Div(decimal.NewFromInt(int64(window)))
	if avgLoss.IsZero() {
		if gainSum.IsZero() {
			return nil
		}

		return decimalPtr(decimal.NewFromInt(100))
	}

	avgGain := gainSum.Div(decimal.NewFromInt(int64(window)))
	rs := avgGain.Div(avgLoss)
	rsi := decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimalOne.Add(rs))).Round(4)
	return decimalPtr(rsi)
}

func calculateShadowRatio(bar marketdata.DailyBar) (*decimal.Decimal, *decimal.Decimal) {
	amplitude := bar.High.Sub(bar.Low)
	if amplitude.IsZero() {
		return nil, nil
	}

	maxBody := maxDecimal(bar.Open, bar.Close)
	minBody := minDecimal(bar.Open, bar.Close)

	upper := bar.High.Sub(maxBody).Div(amplitude).Round(4)
	lower := minBody.Sub(bar.Low).Div(amplitude).Round(4)

	return decimalPtr(upper), decimalPtr(lower)
}

func compareCloseAbove(close decimal.Decimal, ma *decimal.Decimal) *bool {
	if ma == nil {
		return nil
	}

	value := close.GreaterThanOrEqual(*ma)
	return &value
}

func divideDecimalPtr(numerator decimal.Decimal, denominator *decimal.Decimal, scale int32) *decimal.Decimal {
	if denominator == nil || denominator.IsZero() {
		return nil
	}

	value := numerator.Div(*denominator).Round(scale)
	return decimalPtr(value)
}

func maxDecimal(left, right decimal.Decimal) decimal.Decimal {
	if left.GreaterThan(right) {
		return left
	}

	return right
}

func minDecimal(left, right decimal.Decimal) decimal.Decimal {
	if left.LessThan(right) {
		return left
	}

	return right
}

func decimalPtr(value decimal.Decimal) *decimal.Decimal {
	copyValue := value
	return &copyValue
}
