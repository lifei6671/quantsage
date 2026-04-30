package sample

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

type stockBasicRecord struct {
	TSCode   string `json:"ts_code"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Area     string `json:"area"`
	Industry string `json:"industry"`
	Market   string `json:"market"`
	Exchange string `json:"exchange"`
	ListDate string `json:"list_date"`
	Source   string `json:"source"`
}

type tradeCalendarRecord struct {
	Exchange     string `json:"exchange"`
	CalDate      string `json:"cal_date"`
	IsOpen       bool   `json:"is_open"`
	PretradeDate string `json:"pretrade_date"`
	Source       string `json:"source"`
}

type dailyBarRecord struct {
	TSCode    string `json:"ts_code"`
	TradeDate string `json:"trade_date"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	PreClose  string `json:"pre_close"`
	Change    string `json:"change"`
	PctChg    string `json:"pct_chg"`
	Vol       string `json:"vol"`
	Amount    string `json:"amount"`
	Source    string `json:"source"`
}

// Source 从本地 JSON 样例文件读取固定数据。
type Source struct {
	baseDir string
}

// New 创建一个绑定到指定目录的样例数据源。
func New(baseDir string) *Source {
	return &Source{baseDir: baseDir}
}

// ListStocks 返回确定性的样例股票基础信息。
func (s *Source) ListStocks(ctx context.Context) ([]datasource.StockBasic, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("list sample stocks: %w", ctx.Err())
	default:
	}

	var records []stockBasicRecord
	if err := readJSON(filepath.Join(s.baseDir, "stock_basic.json"), &records); err != nil {
		return nil, fmt.Errorf("read sample stock_basic: %w", err)
	}

	items := make([]datasource.StockBasic, 0, len(records))
	for _, record := range records {
		listDate, err := time.Parse("2006-01-02", record.ListDate)
		if err != nil {
			return nil, fmt.Errorf("parse stock list_date: %w", err)
		}
		items = append(items, datasource.StockBasic{
			TSCode:   record.TSCode,
			Symbol:   record.Symbol,
			Name:     record.Name,
			Area:     record.Area,
			Industry: record.Industry,
			Market:   record.Market,
			Exchange: record.Exchange,
			ListDate: listDate,
			Source:   record.Source,
		})
	}

	return items, nil
}

// ListTradeCalendar 返回确定性的样例交易日历数据。
func (s *Source) ListTradeCalendar(ctx context.Context, exchange string, startDate, endDate time.Time) ([]datasource.TradeDay, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("list sample trade calendar: %w", ctx.Err())
	default:
	}
	startDate, endDate = normalizeDateRange(startDate, endDate)

	var records []tradeCalendarRecord
	if err := readJSON(filepath.Join(s.baseDir, "trade_calendar.json"), &records); err != nil {
		return nil, fmt.Errorf("read sample trade_calendar: %w", err)
	}

	items := make([]datasource.TradeDay, 0, len(records))
	for _, record := range records {
		calDate, err := time.Parse("2006-01-02", record.CalDate)
		if err != nil {
			return nil, fmt.Errorf("parse trade cal_date: %w", err)
		}
		calDate = normalizeDate(calDate)
		pretradeDate, err := time.Parse("2006-01-02", record.PretradeDate)
		if err != nil {
			return nil, fmt.Errorf("parse trade pretrade_date: %w", err)
		}
		pretradeDate = normalizeDate(pretradeDate)
		if exchange != "" && record.Exchange != exchange {
			continue
		}
		if calDate.Before(startDate) || calDate.After(endDate) {
			continue
		}
		items = append(items, datasource.TradeDay{
			Exchange:     record.Exchange,
			CalDate:      calDate,
			IsOpen:       record.IsOpen,
			PretradeDate: pretradeDate,
			Source:       record.Source,
		})
	}

	return items, nil
}

// ListDailyBars 返回确定性的样例日线行情数据。
func (s *Source) ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]datasource.DailyBar, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("list sample daily bars: %w", ctx.Err())
	default:
	}
	startDate, endDate = normalizeDateRange(startDate, endDate)

	var records []dailyBarRecord
	if err := readJSON(filepath.Join(s.baseDir, "stock_daily.json"), &records); err != nil {
		return nil, fmt.Errorf("read sample stock_daily: %w", err)
	}

	items := make([]datasource.DailyBar, 0, len(records))
	for _, record := range records {
		tradeDate, err := time.Parse("2006-01-02", record.TradeDate)
		if err != nil {
			return nil, fmt.Errorf("parse daily trade_date: %w", err)
		}
		tradeDate = normalizeDate(tradeDate)
		if tradeDate.Before(startDate) || tradeDate.After(endDate) {
			continue
		}

		open, err := decimal.NewFromString(record.Open)
		if err != nil {
			return nil, fmt.Errorf("parse daily open: %w", err)
		}
		high, err := decimal.NewFromString(record.High)
		if err != nil {
			return nil, fmt.Errorf("parse daily high: %w", err)
		}
		low, err := decimal.NewFromString(record.Low)
		if err != nil {
			return nil, fmt.Errorf("parse daily low: %w", err)
		}
		closePrice, err := decimal.NewFromString(record.Close)
		if err != nil {
			return nil, fmt.Errorf("parse daily close: %w", err)
		}
		preClose, err := decimal.NewFromString(record.PreClose)
		if err != nil {
			return nil, fmt.Errorf("parse daily pre_close: %w", err)
		}
		change, err := decimal.NewFromString(record.Change)
		if err != nil {
			return nil, fmt.Errorf("parse daily change: %w", err)
		}
		pctChg, err := decimal.NewFromString(record.PctChg)
		if err != nil {
			return nil, fmt.Errorf("parse daily pct_chg: %w", err)
		}
		vol, err := decimal.NewFromString(record.Vol)
		if err != nil {
			return nil, fmt.Errorf("parse daily vol: %w", err)
		}
		amount, err := decimal.NewFromString(record.Amount)
		if err != nil {
			return nil, fmt.Errorf("parse daily amount: %w", err)
		}

		items = append(items, datasource.DailyBar{
			TSCode:    record.TSCode,
			TradeDate: tradeDate,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			PreClose:  preClose,
			Change:    change,
			PctChg:    pctChg,
			Vol:       vol,
			Amount:    amount,
			Source:    record.Source,
		})
	}

	return items, nil
}

// ListKLines 返回样例数据源支持的日线单票查询结果。
func (s *Source) ListKLines(ctx context.Context, query datasource.KLineQuery) ([]datasource.KLine, error) {
	normalizedQuery, err := datasource.NormalizeKLineQuery(query, time.Now)
	if err != nil {
		return nil, err
	}
	if normalizedQuery.Interval != datasource.IntervalDay {
		return nil, apperror.New(
			apperror.CodeDatasourceUnavailable,
			fmt.Errorf("sample datasource does not support interval %s", normalizedQuery.Interval),
		)
	}

	bars, err := s.ListDailyBars(ctx, time.Time{}, queryEndForSampleKLine(normalizedQuery))
	if err != nil {
		return nil, fmt.Errorf("list sample klines: %w", err)
	}

	items := make([]datasource.KLine, 0, len(bars))
	for _, bar := range bars {
		if bar.TSCode != normalizedQuery.TSCode {
			continue
		}
		if !normalizedQuery.StartTime.IsZero() && (bar.TradeDate.Before(normalizedQuery.StartTime) || bar.TradeDate.After(normalizedQuery.EndTime)) {
			continue
		}
		if normalizedQuery.Limit > 0 && bar.TradeDate.After(normalizeDate(normalizedQuery.EndTime)) {
			continue
		}
		items = append(items, mapDailyBarToKLine(bar))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].TradeTime.Before(items[j].TradeTime)
	})

	return datasource.TrimKLinesByLimit(items, normalizedQuery.Limit), nil
}

// StreamKLines 当前样例数据源不支持流式 K 线接口。
func (s *Source) StreamKLines(context.Context, datasource.KLineQuery) (<-chan datasource.KLineStreamItem, error) {
	return nil, datasource.UnsupportedStreamError("sample")
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read json file: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal json file: %w", err)
	}

	return nil
}

func normalizeDateRange(startDate, endDate time.Time) (time.Time, time.Time) {
	return normalizeDate(startDate), normalizeDate(endDate)
}

func normalizeDate(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}

	utcValue := value.UTC()
	return time.Date(utcValue.Year(), utcValue.Month(), utcValue.Day(), 0, 0, 0, 0, time.UTC)
}

func mapDailyBarToKLine(bar datasource.DailyBar) datasource.KLine {
	return datasource.KLine{
		TSCode:    bar.TSCode,
		TradeTime: bar.TradeDate,
		Open:      bar.Open,
		High:      bar.High,
		Low:       bar.Low,
		Close:     bar.Close,
		PreClose:  bar.PreClose,
		Change:    bar.Change,
		PctChg:    bar.PctChg,
		Vol:       bar.Vol,
		Amount:    bar.Amount,
		Source:    bar.Source,
	}
}

func queryEndForSampleKLine(query datasource.KLineQuery) time.Time {
	if query.EndTime.IsZero() {
		return time.Time{}
	}

	return normalizeDate(query.EndTime)
}
