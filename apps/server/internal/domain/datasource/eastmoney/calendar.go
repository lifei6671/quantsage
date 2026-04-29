package eastmoney

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
)

// ListTradeCalendar 基于 A 股统一交易日历基准指数推导交易日。
func (s *Source) ListTradeCalendar(ctx context.Context, exchange string, startDate, endDate time.Time) ([]datasource.TradeDay, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	normalizedExchange, err := normalizeExchange(exchange)
	if err != nil {
		return nil, err
	}
	startDate, endDate = normalizeDateRange(startDate, endDate)

	// 东财未提供稳定且公开的交易日历接口，这里统一使用上证综指日线推导
	// 沪深北三地共享的 A 股交易日，并把假设收口在一个函数里，方便后续替换。
	body, err := s.fallbackClient.GetHistory(ctx, historyKLinePath, buildKLineQuery(calendarBenchmarkSecID(normalizedExchange), IntervalDay, AdjustNone, startDate, endDate, 0))
	if err != nil {
		return nil, err
	}

	var response KLineAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, datasourceUnavailable(fmt.Errorf("decode eastmoney trade calendar response: %w", err))
	}
	if response.RC != 0 {
		return nil, datasourceUnavailable(
			fmt.Errorf("eastmoney trade calendar rc=%d message=%q", response.RC, strings.TrimSpace(response.Message)),
		)
	}

	parsed, err := ParseKLineRows(normalizedExchange, IntervalDay, response.Data.KLines)
	if err != nil {
		return nil, datasourceUnavailable(fmt.Errorf("parse eastmoney trade calendar response: %w", err))
	}

	items := make([]datasource.TradeDay, 0, len(parsed))
	var previousTradeDate time.Time
	for _, item := range parsed {
		tradeDate := normalizeDate(item.TradeTime)
		if tradeDate.Before(startDate) || tradeDate.After(endDate) {
			continue
		}

		items = append(items, datasource.TradeDay{
			Exchange:     normalizedExchange,
			CalDate:      tradeDate,
			IsOpen:       true,
			PretradeDate: previousTradeDate,
			Source:       sourceName,
		})
		previousTradeDate = tradeDate
	}

	return items, nil
}

func calendarBenchmarkSecID(exchange string) string {
	// 当前三地交易日历保持一致，统一选择上证综指作为可验证基准。
	switch exchange {
	case "SSE", "SZSE", "BSE":
		return "1.000001"
	default:
		return "1.000001"
	}
}

func normalizeExchange(exchange string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(exchange)) {
	case "", "SSE":
		return "SSE", nil
	case "SZSE":
		return "SZSE", nil
	case "BSE":
		return "BSE", nil
	default:
		return "", datasourceUnavailable(fmt.Errorf("unsupported eastmoney exchange %q", exchange))
	}
}

func buildKLineQuery(secID string, interval Interval, adjust AdjustType, startDate, endDate time.Time, limit int) url.Values {
	query := url.Values{
		"secid":   []string{secID},
		"klt":     []string{mustMapInterval(interval)},
		"fqt":     []string{MapAdjustType(adjust)},
		"fields1": []string{defaultKLineFields1},
		"fields2": []string{defaultKLineFields2},
		"beg":     []string{formatEastMoneyDate(startDate)},
		"end":     []string{formatEastMoneyDate(endDate)},
	}
	if limit > 0 {
		query.Set("lmt", fmt.Sprintf("%d", limit))
	}

	return query
}

func mustMapInterval(interval Interval) string {
	mapped, err := MapIntervalToEastMoneyKLT(interval)
	if err != nil {
		return ""
	}

	return mapped
}
