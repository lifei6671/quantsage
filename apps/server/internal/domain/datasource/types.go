package datasource

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

var chinaMarketTimeZone = time.FixedZone("Asia/Shanghai", 8*3600)

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

// Interval 定义统一的 K 线周期枚举。
type Interval string

const (
	Interval1Min    Interval = "1m"
	Interval5Min    Interval = "5m"
	Interval15Min   Interval = "15m"
	Interval30Min   Interval = "30m"
	Interval60Min   Interval = "60m"
	IntervalDay     Interval = "1d"
	IntervalWeek    Interval = "1w"
	IntervalMonth   Interval = "1mo"
	IntervalQuarter Interval = "1q"
	IntervalYear    Interval = "1y"
)

// KLine 表示统一的单票 K 线结构。
type KLine struct {
	TSCode    string
	TradeTime time.Time
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

// KLineQuery 表示统一的单票 K 线查询参数。
type KLineQuery struct {
	TSCode    string
	Interval  Interval
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

// KLineStreamItem 表示流式 K 线接口的单次推送结果。
// 一个流式事件可以携带一批 K 线，便于页面驱动型数据源按响应批次回推数据。
type KLineStreamItem struct {
	Items []KLine
	Err   error
}

// Source 定义 QuantSage 对外部数据源的最小读取契约。
type Source interface {
	ListStocks(ctx context.Context) ([]StockBasic, error)
	ListTradeCalendar(ctx context.Context, exchange string, startDate, endDate time.Time) ([]TradeDay, error)
	ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]DailyBar, error)
	ListKLines(ctx context.Context, query KLineQuery) ([]KLine, error)
	StreamKLines(ctx context.Context, query KLineQuery) (<-chan KLineStreamItem, error)
}

// NormalizeKLineQuery 统一整理公共 K 线查询参数。
func NormalizeKLineQuery(query KLineQuery, now func() time.Time) (KLineQuery, error) {
	query.TSCode = strings.ToUpper(strings.TrimSpace(query.TSCode))
	if query.TSCode == "" {
		return KLineQuery{}, apperror.New(apperror.CodeBadRequest, errors.New("ts_code is required"))
	}
	if query.Interval == "" {
		return KLineQuery{}, apperror.New(apperror.CodeBadRequest, errors.New("interval is required"))
	}

	if now == nil {
		now = time.Now
	}

	if query.Limit > 0 {
		if query.EndTime.IsZero() {
			query.EndTime = currentMarketKLineBoundary(query.Interval, now())
		} else {
			query.EndTime = normalizeKLineBoundary(query.Interval, query.EndTime)
		}
		query.StartTime = time.Time{}
		return query, nil
	}

	if query.StartTime.IsZero() || query.EndTime.IsZero() {
		return KLineQuery{}, apperror.New(
			apperror.CodeBadRequest,
			errors.New("start_time and end_time are required when limit <= 0"),
		)
	}
	query.StartTime = normalizeKLineBoundary(query.Interval, query.StartTime)
	query.EndTime = normalizeKLineBoundary(query.Interval, query.EndTime)
	if query.StartTime.After(query.EndTime) {
		return KLineQuery{}, apperror.New(apperror.CodeBadRequest, errors.New("start_time must be before or equal to end_time"))
	}

	return query, nil
}

func normalizeKLineBoundary(interval Interval, value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	if usesCalendarBoundary(interval) {
		year, month, day := value.Date()
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}

	return value.UTC()
}

func currentMarketKLineBoundary(interval Interval, value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	if usesCalendarBoundary(interval) {
		year, month, day := value.In(chinaMarketTimeZone).Date()
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}

	return value.UTC()
}

func usesCalendarBoundary(interval Interval) bool {
	switch interval {
	case IntervalDay, IntervalWeek, IntervalMonth, IntervalQuarter, IntervalYear:
		return true
	default:
		return false
	}
}

// TrimKLinesByLimit 在保持升序排列的前提下，裁剪到最近 N 条。
func TrimKLinesByLimit(items []KLine, limit int) []KLine {
	if limit <= 0 || len(items) <= limit {
		return slices.Clone(items)
	}

	return slices.Clone(items[len(items)-limit:])
}

// UnsupportedStreamError 返回统一的“不支持流式 K 线”错误。
func UnsupportedStreamError(sourceName string) error {
	sourceName = strings.TrimSpace(sourceName)
	if sourceName == "" {
		sourceName = "current"
	}

	return apperror.New(
		apperror.CodeDatasourceUnavailable,
		fmt.Errorf("%s datasource does not support streaming kline", sourceName),
	)
}
