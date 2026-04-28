package strategy

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/indicator"
	"github.com/lifei6671/quantsage/apps/server/internal/domain/marketdata"
)

const (
	StrategyCodeVolumeBreakout = "volume_breakout_v1"
	StrategyCodeTrendBreak     = "trend_break_v1"
	StrategyVersionV1          = "v1"
)

// MarketContext 表示策略评估时所需的市场上下文。
type MarketContext struct {
	CurrentBar    marketdata.DailyBar
	CurrentFactor indicator.DailyFactor
	RecentBars    []marketdata.DailyBar
}

// SignalResult 表示单条策略信号结果。
type SignalResult struct {
	StrategyCode          string
	StrategyVersion       string
	TSCode                string
	TradeDate             time.Time
	SignalType            string
	SignalStrength        decimal.Decimal
	SignalLevel           string
	BuyPriceRef           decimal.Decimal
	StopLossRef           decimal.Decimal
	TakeProfitRef         decimal.Decimal
	InvalidationCondition string
	Reason                string
	InputSnapshot         map[string]any
}
