package indicator

import (
	"time"

	"github.com/shopspring/decimal"
)

// DailyFactor 表示单只股票在单个交易日上的衍生因子结果。
type DailyFactor struct {
	TSCode string

	TradeDate time.Time

	MA5  *decimal.Decimal
	MA10 *decimal.Decimal
	MA20 *decimal.Decimal
	MA60 *decimal.Decimal

	EMA12 *decimal.Decimal
	EMA26 *decimal.Decimal

	MACDDIF  *decimal.Decimal
	MACDDEA  *decimal.Decimal
	MACDHist *decimal.Decimal

	RSI6  *decimal.Decimal
	RSI12 *decimal.Decimal

	VolumeMA5   *decimal.Decimal
	VolumeMA20  *decimal.Decimal
	VolumeRatio *decimal.Decimal

	UpperShadowRatio *decimal.Decimal
	LowerShadowRatio *decimal.Decimal

	CloseAboveMA5  *bool
	CloseAboveMA10 *bool
	CloseAboveMA20 *bool
}
