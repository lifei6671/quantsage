package eastmoney

import (
	"fmt"
	"strings"
)

// ConvertTSCodeToSecID 将 Tushare 风格 TSCode 转成东财 secid。
func ConvertTSCodeToSecID(tsCode string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(tsCode))
	parts := strings.Split(normalized, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("unsupported eastmoney ts_code %q: want 6-digit code with market suffix", tsCode)
	}

	symbol := strings.TrimSpace(parts[0])
	market := strings.TrimSpace(parts[1])
	if len(symbol) != 6 {
		return "", fmt.Errorf("unsupported eastmoney ts_code %q: symbol must be 6 digits", tsCode)
	}
	for _, ch := range symbol {
		if ch < '0' || ch > '9' {
			return "", fmt.Errorf("unsupported eastmoney ts_code %q: symbol must be numeric", tsCode)
		}
	}

	switch market {
	case "SZ", "BJ":
		return "0." + symbol, nil
	case "SH":
		return "1." + symbol, nil
	default:
		return "", fmt.Errorf("unsupported eastmoney market %q for ts_code %q: only SH, SZ, BJ are supported", market, tsCode)
	}
}

// MapIntervalToEastMoneyKLT 将查询周期映射为东财 KLT 参数。
func MapIntervalToEastMoneyKLT(interval Interval) (string, error) {
	switch Interval(strings.ToLower(strings.TrimSpace(string(interval)))) {
	case Interval1Min:
		return "1", nil
	case Interval5Min:
		return "5", nil
	case Interval15Min:
		return "15", nil
	case Interval30Min:
		return "30", nil
	case Interval60Min:
		return "60", nil
	case IntervalDay:
		return "101", nil
	case IntervalWeek:
		return "102", nil
	case IntervalMonth:
		return "103", nil
	case IntervalQuarter:
		return "104", nil
	case IntervalYear:
		return "105", nil
	default:
		return "", fmt.Errorf("unsupported eastmoney interval %q", interval)
	}
}

// MapAdjustType 将复权类型映射为东财 fqt 参数。
func MapAdjustType(adjust AdjustType) string {
	switch AdjustType(strings.ToLower(strings.TrimSpace(string(adjust)))) {
	case AdjustQFQ:
		return "1"
	case AdjustHFQ:
		return "2"
	default:
		return "0"
	}
}

func mapStockIdentity(marketCode int, symbol string) (tsCode string, exchange string, market string, err error) {
	trimmedSymbol := strings.TrimSpace(symbol)
	if len(trimmedSymbol) != 6 {
		return "", "", "", fmt.Errorf("symbol %q must be 6 digits", symbol)
	}

	suffix := "SZ"
	exchange = "SZSE"
	market = "A"
	switch {
	case marketCode == 1:
		suffix = "SH"
		exchange = "SSE"
		if strings.HasPrefix(trimmedSymbol, "688") {
			market = "STAR"
		} else {
			market = "MAIN"
		}
	case isBeijingSymbol(trimmedSymbol):
		suffix = "BJ"
		exchange = "BSE"
		market = "BSE"
	case strings.HasPrefix(trimmedSymbol, "300"):
		market = "GEM"
	default:
		market = "MAIN"
	}

	return trimmedSymbol + "." + suffix, exchange, market, nil
}

func isBeijingSymbol(symbol string) bool {
	return strings.HasPrefix(symbol, "4") ||
		strings.HasPrefix(symbol, "8") ||
		strings.HasPrefix(symbol, "92")
}
