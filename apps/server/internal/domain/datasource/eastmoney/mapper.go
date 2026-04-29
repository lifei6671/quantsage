package eastmoney

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// ParsedKLine 是东财 K 线文本行解析后的标准结构。
type ParsedKLine struct {
	TradeTime    time.Time
	Open         decimal.Decimal
	Close        decimal.Decimal
	High         decimal.Decimal
	Low          decimal.Decimal
	Vol          decimal.Decimal
	Amount       decimal.Decimal
	PctChg       decimal.Decimal
	Change       decimal.Decimal
	PreClose     decimal.Decimal
	TurnoverRate decimal.Decimal
}

// ParseKLineRows 将东财原始 klines 字符串切片解析成结构化行情。
func ParseKLineRows(entityID string, interval Interval, rows []string) ([]ParsedKLine, error) {
	items := make([]ParsedKLine, 0, len(rows))
	for _, row := range rows {
		parsed, err := parseKLineRow(entityID, interval, row)
		if err != nil {
			return nil, err
		}
		items = append(items, parsed)
	}

	return items, nil
}

func parseKLineRow(entityID string, interval Interval, row string) (ParsedKLine, error) {
	parts := strings.Split(strings.TrimSpace(row), ",")
	if len(parts) < 11 {
		return ParsedKLine{}, fmt.Errorf("eastmoney kline %s has %d fields, want at least 11", entityID, len(parts))
	}

	tradeTime, err := parseKLineTime(interval, parts[0])
	if err != nil {
		return ParsedKLine{}, fmt.Errorf("parse eastmoney kline time for %s: %w", entityID, err)
	}
	open, err := decimal.NewFromString(strings.TrimSpace(parts[1]))
	if err != nil {
		return ParsedKLine{}, fmt.Errorf("parse eastmoney kline open for %s: %w", entityID, err)
	}
	closePrice, err := decimal.NewFromString(strings.TrimSpace(parts[2]))
	if err != nil {
		return ParsedKLine{}, fmt.Errorf("parse eastmoney kline close for %s: %w", entityID, err)
	}
	high, err := decimal.NewFromString(strings.TrimSpace(parts[3]))
	if err != nil {
		return ParsedKLine{}, fmt.Errorf("parse eastmoney kline high for %s: %w", entityID, err)
	}
	low, err := decimal.NewFromString(strings.TrimSpace(parts[4]))
	if err != nil {
		return ParsedKLine{}, fmt.Errorf("parse eastmoney kline low for %s: %w", entityID, err)
	}
	vol, err := decimal.NewFromString(strings.TrimSpace(parts[5]))
	if err != nil {
		return ParsedKLine{}, fmt.Errorf("parse eastmoney kline vol for %s: %w", entityID, err)
	}
	amount, err := decimal.NewFromString(strings.TrimSpace(parts[6]))
	if err != nil {
		return ParsedKLine{}, fmt.Errorf("parse eastmoney kline amount for %s: %w", entityID, err)
	}
	pctChg, err := decimal.NewFromString(strings.TrimSpace(parts[8]))
	if err != nil {
		return ParsedKLine{}, fmt.Errorf("parse eastmoney kline pct_chg for %s: %w", entityID, err)
	}
	change, err := decimal.NewFromString(strings.TrimSpace(parts[9]))
	if err != nil {
		return ParsedKLine{}, fmt.Errorf("parse eastmoney kline change for %s: %w", entityID, err)
	}
	turnoverRate, err := decimal.NewFromString(strings.TrimSpace(parts[10]))
	if err != nil {
		return ParsedKLine{}, fmt.Errorf("parse eastmoney kline turnover_rate for %s: %w", entityID, err)
	}

	return ParsedKLine{
		TradeTime:    tradeTime.UTC(),
		Open:         open,
		Close:        closePrice,
		High:         high,
		Low:          low,
		Vol:          vol,
		Amount:       amount,
		PctChg:       pctChg,
		Change:       change,
		PreClose:     closePrice.Sub(change),
		TurnoverRate: turnoverRate,
	}, nil
}

func parseKLineTime(interval Interval, value string) (time.Time, error) {
	layout := eastMoneyDateLayout
	if isMinuteInterval(interval) {
		layout = eastMoneyMinuteLayout
	}

	return time.ParseInLocation(layout, strings.TrimSpace(value), time.UTC)
}

func isMinuteInterval(interval Interval) bool {
	switch interval {
	case Interval1Min, Interval5Min, Interval15Min, Interval30Min, Interval60Min:
		return true
	default:
		return false
	}
}

func parseOptionalStockListDate(value string) time.Time {
	parsed, err := time.Parse(eastMoneyDateCompactLayout, strings.TrimSpace(value))
	if err == nil {
		return normalizeDate(parsed)
	}

	parsed, err = time.Parse(eastMoneyDateLayout, strings.TrimSpace(value))
	if err == nil {
		return normalizeDate(parsed)
	}

	return time.Time{}
}

func normalizeDateRange(startDate, endDate time.Time) (time.Time, time.Time) {
	startDate = normalizeDate(startDate)
	endDate = normalizeDate(endDate)
	if !startDate.IsZero() && endDate.IsZero() {
		endDate = startDate
	}
	if startDate.IsZero() && !endDate.IsZero() {
		startDate = endDate
	}
	if !startDate.IsZero() && !endDate.IsZero() && startDate.After(endDate) {
		startDate, endDate = endDate, startDate
	}

	return startDate, endDate
}

func normalizeDate(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}

	utcValue := value.UTC()
	return time.Date(utcValue.Year(), utcValue.Month(), utcValue.Day(), 0, 0, 0, 0, time.UTC)
}

func formatEastMoneyDate(value time.Time) string {
	if value.IsZero() {
		return "0"
	}

	return normalizeDate(value).Format(eastMoneyDateCompactLayout)
}
