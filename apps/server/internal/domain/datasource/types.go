package datasource

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// StockBasic 表示外部数据源返回的股票基础信息。
type StockBasic struct {
	TSCode   string
	Symbol   string
	Name     string
	Area     string
	Industry string
	Market   string
	Exchange string
	ListDate time.Time
	Source   string
}

// TradeDay 表示交易日历数据。
type TradeDay struct {
	Exchange     string
	CalDate      time.Time
	IsOpen       bool
	PretradeDate time.Time
	Source       string
}

// DailyBar 表示日线行情数据。
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

// Source 定义 QuantSage 对外部数据源的最小读取契约。
type Source interface {
	ListStocks(ctx context.Context) ([]StockBasic, error)
	ListTradeCalendar(ctx context.Context, exchange string, startDate, endDate time.Time) ([]TradeDay, error)
	ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]DailyBar, error)
}
