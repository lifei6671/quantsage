package sina

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
)

type sinaKLineItem struct {
	Day    string `json:"day"`
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Close  string `json:"close"`
	Volume string `json:"volume"`
}

var chinaMarketLocation = time.FixedZone("Asia/Shanghai", 8*3600)

func parseKLinesFromResponse(tsCode string, interval datasource.Interval, body []byte) ([]datasource.KLine, error) {
	items, err := parseJSONPResponse(body)
	if err != nil {
		return nil, err
	}

	result := make([]datasource.KLine, 0, len(items))
	for _, item := range items {
		tradeTime, err := parseSinaTradeTime(interval, item.Day)
		if err != nil {
			return nil, fmt.Errorf("parse trade time %q: %w", item.Day, err)
		}

		openValue, err := parseDecimal(item.Open)
		if err != nil {
			return nil, fmt.Errorf("parse open %q: %w", item.Open, err)
		}
		highValue, err := parseDecimal(item.High)
		if err != nil {
			return nil, fmt.Errorf("parse high %q: %w", item.High, err)
		}
		lowValue, err := parseDecimal(item.Low)
		if err != nil {
			return nil, fmt.Errorf("parse low %q: %w", item.Low, err)
		}
		closeValue, err := parseDecimal(item.Close)
		if err != nil {
			return nil, fmt.Errorf("parse close %q: %w", item.Close, err)
		}
		volValue, err := parseDecimal(item.Volume)
		if err != nil {
			return nil, fmt.Errorf("parse volume %q: %w", item.Volume, err)
		}

		result = append(result, datasource.KLine{
			TSCode:    tsCode,
			TradeTime: tradeTime,
			Open:      openValue,
			High:      highValue,
			Low:       lowValue,
			Close:     closeValue,
			Vol:       volValue,
			Amount:    decimal.Zero,
			Source:    sourceName,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TradeTime.Before(result[j].TradeTime)
	})
	for index := range result {
		if index == 0 {
			continue
		}

		prevClose := result[index-1].Close
		result[index].PreClose = prevClose
		result[index].Change = result[index].Close.Sub(prevClose)
		if !prevClose.IsZero() {
			result[index].PctChg = result[index].Change.Div(prevClose).Mul(decimal.NewFromInt(100))
		}
	}

	return result, nil
}

func parseJSONPResponse(body []byte) ([]sinaKLineItem, error) {
	var items []sinaKLineItem
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, nil
	}

	jsonStart := strings.Index(trimmed, "[")
	if jsonStart >= 0 {
		lastBracket := strings.LastIndex(trimmed, "]")
		if lastBracket < jsonStart {
			return nil, fmt.Errorf("no closing ] found in sina response")
		}
		trimmed = trimmed[jsonStart : lastBracket+1]
	}

	if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
		return nil, fmt.Errorf("decode sina response: %w", err)
	}

	return items, nil
}

func parseSinaTradeTime(interval datasource.Interval, value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	layouts := []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"}
	location := chinaMarketLocation
	if interval == datasource.IntervalDay || interval == datasource.IntervalWeek {
		layouts = []string{"2006-01-02", "2006-01-02 15:04:05", "2006-01-02 15:04"}
		location = time.UTC
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, value, location)
		if err == nil {
			return parsed.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported time format")
}

func parseDecimal(value string) (decimal.Decimal, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "-" || strings.EqualFold(value, "null") {
		return decimal.Zero, nil
	}

	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, err
	}

	return parsed, nil
}
