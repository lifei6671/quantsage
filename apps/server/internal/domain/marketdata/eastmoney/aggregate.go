package eastmoney

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// AggregateKLines 将更细粒度 K 线按固定窗口聚合。
func AggregateKLines(items []KLine, size int) ([]KLine, error) {
	if size <= 0 {
		return nil, fmt.Errorf("aggregate size must be positive")
	}
	if len(items) == 0 {
		return []KLine{}, nil
	}

	result := make([]KLine, 0, (len(items)+size-1)/size)
	for start := 0; start < len(items); start += size {
		end := start + size
		if end > len(items) {
			end = len(items)
		}

		chunk := items[start:end]
		aggregated := chunk[0]
		aggregated.High = chunk[0].High
		aggregated.Low = chunk[0].Low
		aggregated.Vol = decimal.Zero
		aggregated.Amount = decimal.Zero
		aggregated.TurnoverRate = decimal.Zero
		for _, item := range chunk {
			if item.High.GreaterThan(aggregated.High) {
				aggregated.High = item.High
			}
			if item.Low.LessThan(aggregated.Low) {
				aggregated.Low = item.Low
			}
			aggregated.Close = item.Close
			aggregated.TradeTime = item.TradeTime
			aggregated.Vol = aggregated.Vol.Add(item.Vol)
			aggregated.Amount = aggregated.Amount.Add(item.Amount)
			aggregated.TurnoverRate = aggregated.TurnoverRate.Add(item.TurnoverRate)
		}
		aggregated.Change = aggregated.Close.Sub(aggregated.PreClose)
		if !aggregated.PreClose.IsZero() {
			aggregated.PctChg = aggregated.Change.Div(aggregated.PreClose).Mul(decimal.NewFromInt(100))
		} else {
			aggregated.PctChg = decimal.Zero
		}

		result = append(result, aggregated)
	}

	return result, nil
}
