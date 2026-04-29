package eastmoney

import "github.com/shopspring/decimal"

// AttachSimpleMovingAverages 为每条 K 线附加简单移动均线值。
func AttachSimpleMovingAverages(items []KLine, periods []int) []KLineWithMA {
	result := make([]KLineWithMA, 0, len(items))
	for index, item := range items {
		enriched := KLineWithMA{KLine: item}
		for _, period := range periods {
			if period <= 0 || index+1 < period {
				continue
			}

			sum := decimal.Zero
			for offset := index + 1 - period; offset <= index; offset++ {
				sum = sum.Add(items[offset].Close)
			}
			enriched.MovingAverages = append(enriched.MovingAverages, MAValue{
				Period: period,
				Value:  sum.Div(decimal.NewFromInt(int64(period))),
			})
		}
		result = append(result, enriched)
	}

	return result
}
