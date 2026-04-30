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
	// ListStocks 返回当前数据源可枚举的股票基础信息。
	// 这个方法主要服务于“全市场同步”类任务，因此实现方应尽量返回稳定、可重复抓取的全集结果，
	// 而不是带页面上下文或临时过滤条件的局部结果。
	// 如果某个数据源天然不具备批量股票列表能力，应返回明确的“不支持”错误，而不是返回空切片伪装成功。
	ListStocks(ctx context.Context) ([]StockBasic, error)

	// ListTradeCalendar 返回指定交易所、指定日期区间内的交易日历。
	// 调用方依赖它判断某天是否开市、以及补数/对账任务的日期边界，因此实现方需要保证：
	// 1. 返回结果只包含 [startDate, endDate] 区间内的数据；
	// 2. 日期语义稳定，避免把时分秒或本地时区细节泄漏给上层；
	// 3. exchange 不支持时返回明确错误，便于调用方降级或停止任务。
	ListTradeCalendar(ctx context.Context, exchange string, startDate, endDate time.Time) ([]TradeDay, error)

	// ListDailyBars 返回一个日期区间内的日线行情集合。
	// 该接口面向“批量导入”而不是单票查询：实现方可以按自身能力抓取全市场、分批汇总后统一返回，
	// 但最终结果必须是标准化后的 DailyBar 切片，不能把分页状态、流式事件或浏览器细节暴露到接口外。
	// 后续如果单票、多周期需求变复杂，优先放到 ListKLines，而不是继续扩展这个日线批量接口。
	ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]DailyBar, error)

	// ListKLines 返回单只股票在指定周期下的标准化 K 线结果。
	// 它面向“查询能力”而非“全量导入能力”，维护时需要重点保持以下约束：
	// 1. 入参语义遵循 KLineQuery / NormalizeKLineQuery 的统一规则；
	// 2. 返回结果按 TradeTime 升序排列，便于上层直接做裁剪、指标计算和展示；
	// 3. 默认返回原始行情，不在这里隐式加入复权、缓存命中来源或页面采集细节；
	// 4. 某个周期不支持时返回显式错误，避免静默降级到别的周期或空结果。
	ListKLines(ctx context.Context, query KLineQuery) ([]KLine, error)

	// StreamKLines 返回流式 K 线事件通道，适合页面驱动型或实时推送型数据源。
	// 实现方应把底层多次响应整理成 KLineStreamItem 批次后再向外输出，并在通道关闭前保证错误可观测。
	// 对于只支持一次性查询的数据源，允许直接返回 UnsupportedStreamError；
	// 但一旦声明支持流式语义，就不应把它退化成“内部先收完再一次性吐出”的假流式实现。
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
