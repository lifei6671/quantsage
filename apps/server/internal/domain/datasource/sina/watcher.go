package sina

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
	"github.com/lifei6671/quantsage/apps/server/internal/infra/browserfetch"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

const defaultObserveIdleTimeout = 5 * time.Second

type pageResponseWatcher interface {
	ObserveResponses(ctx context.Context, pageURL string, opts ...browserfetch.ObserveOption) (*browserfetch.ResponseStream, error)
}

func (s *Source) watchKLineResponses(ctx context.Context, query datasource.KLineQuery) (*browserfetch.ResponseStream, error) {
	if s == nil || s.browser == nil {
		return nil, browserUnavailableError()
	}

	requestURL, err := buildKLineRequestURL(query)
	if err != nil {
		return nil, err
	}

	return s.browser.ObserveResponses(
		ctx,
		requestURL,
		browserfetch.WithObserveIdleTimeout(defaultObserveIdleTimeout),
		browserfetch.WithObserveMatch(func(meta browserfetch.ResponseMetadata) bool {
			return matchSinaKLineResponse(meta, query.TSCode, query.Interval)
		}),
	)
}

func buildStockPageURL(tsCode string) (string, error) {
	symbol, err := sinaSymbolFromTSCode(tsCode)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://finance.sina.com.cn/realstock/company/%s/nc.shtml", symbol), nil
}

func buildKLineRequestURL(query datasource.KLineQuery) (string, error) {
	symbol, err := sinaSymbolFromTSCode(query.TSCode)
	if err != nil {
		return "", err
	}

	scale, err := sinaScaleFromInterval(query.Interval)
	if err != nil {
		return "", err
	}

	callback := fmt.Sprintf("callback_%d", time.Now().UnixMilli())
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("scale", scale)
	params.Set("ma", "no")
	params.Set("datalen", fmt.Sprintf("%d", query.Limit))

	return fmt.Sprintf(
		"https://quotes.sina.cn/cn/api/jsonp_v2.php/%s/CN_MarketDataService.getKLineData?%s",
		callback,
		params.Encode(),
	), nil
}

func matchSinaKLineResponse(meta browserfetch.ResponseMetadata, tsCode string, interval datasource.Interval) bool {
	if !strings.Contains(meta.URL, "CN_MarketDataService.getKLineData") {
		return false
	}

	parsedURL, err := url.Parse(meta.URL)
	if err != nil {
		return false
	}
	query := parsedURL.Query()
	if strings.TrimSpace(query.Get("scale")) != mustSinaScale(interval) {
		return false
	}
	expectedSymbol, err := sinaSymbolFromTSCode(tsCode)
	if err != nil {
		return false
	}
	if symbol := strings.TrimSpace(query.Get("symbol")); symbol != "" && !strings.EqualFold(symbol, expectedSymbol) {
		return false
	}
	if query.Get("ma") != "" && !strings.EqualFold(strings.TrimSpace(query.Get("ma")), "no") {
		return false
	}

	return true
}

func sinaSymbolFromTSCode(tsCode string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(tsCode))
	if code == "" {
		return "", fmt.Errorf("convert ts_code to sina symbol: ts_code is required")
	}
	if strings.Contains(code, ".") {
		parts := strings.Split(code, ".")
		if len(parts) == 2 {
			switch parts[1] {
			case "SH", "SS":
				return "sh" + parts[0], nil
			case "SZ":
				return "sz" + parts[0], nil
			case "BJ":
				return "bj" + parts[0], nil
			}
		}
	}
	if strings.HasPrefix(code, "SH") || strings.HasPrefix(code, "SZ") || strings.HasPrefix(code, "BJ") {
		return strings.ToLower(code[:2]) + code[2:], nil
	}
	switch code[0] {
	case '6':
		return "sh" + code, nil
	case '0', '3':
		return "sz" + code, nil
	case '8', '9':
		return "bj" + code, nil
	default:
		return "", fmt.Errorf("convert ts_code %s to sina symbol: unsupported market", tsCode)
	}
}

func sinaScaleFromInterval(interval datasource.Interval) (string, error) {
	switch interval {
	case datasource.Interval1Min:
		return "1", nil
	case datasource.Interval5Min:
		return "5", nil
	case datasource.Interval15Min:
		return "15", nil
	case datasource.Interval30Min:
		return "30", nil
	case datasource.Interval60Min:
		return "60", nil
	case datasource.IntervalDay:
		return "240", nil
	case datasource.IntervalWeek:
		return "1200", nil
	default:
		return "", apperror.New(
			apperror.CodeDatasourceUnavailable,
			fmt.Errorf("sina datasource does not support interval %s", interval),
		)
	}
}

func mustSinaScale(interval datasource.Interval) string {
	scale, _ := sinaScaleFromInterval(interval)
	return scale
}
