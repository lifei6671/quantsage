package strategy

import "github.com/shopspring/decimal"

var (
	scoreAThreshold = decimal.NewFromInt(80)
	scoreBThreshold = decimal.NewFromInt(60)
	scoreCThreshold = decimal.NewFromInt(40)
)

// SignalLevelFromScore 根据分数映射信号等级。
func SignalLevelFromScore(score decimal.Decimal) string {
	switch {
	case score.GreaterThanOrEqual(scoreAThreshold):
		return "A"
	case score.GreaterThanOrEqual(scoreBThreshold):
		return "B"
	case score.GreaterThanOrEqual(scoreCThreshold):
		return "C"
	default:
		return "D"
	}
}
