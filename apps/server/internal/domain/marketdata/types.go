package marketdata

import (
	"time"

	"github.com/shopspring/decimal"
)

// DailyBar 表示进入指标计算流程的标准日线数据。
type DailyBar struct {
	TSCode    string
	TradeDate time.Time
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	PreClose  decimal.Decimal
	Change    decimal.Decimal
	PctChg    decimal.Decimal
	Vol       decimal.Decimal
	Amount    decimal.Decimal
	Source    string
}
