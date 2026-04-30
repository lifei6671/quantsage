package eastmoney

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

// ListKLines 返回东财单票多周期 K 线查询结果。
func (s *Source) ListKLines(ctx context.Context, query datasource.KLineQuery) ([]datasource.KLine, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	normalizedQuery, err := normalizeDatasourceKLineQuery(query)
	if err != nil {
		return nil, err
	}

	return fetchDatasourceKLines(ctx, s.fallbackClient, normalizedQuery)
}

// StreamKLines 当前东财数据源不支持流式 K 线接口。
func (s *Source) StreamKLines(context.Context, datasource.KLineQuery) (<-chan datasource.KLineStreamItem, error) {
	return nil, datasource.UnsupportedStreamError("eastmoney")
}

func normalizeDatasourceKLineQuery(query datasource.KLineQuery) (datasource.KLineQuery, error) {
	normalizedQuery, err := datasource.NormalizeKLineQuery(query, time.Now)
	if err != nil {
		return datasource.KLineQuery{}, err
	}
	return normalizedQuery, nil
}

func fetchDatasourceKLines(ctx context.Context, client HistoryClient, query datasource.KLineQuery) ([]datasource.KLine, error) {
	secID, err := ConvertTSCodeToSecID(query.TSCode)
	if err != nil {
		return nil, apperror.New(apperror.CodeBadRequest, fmt.Errorf("convert ts_code %s to secid: %w", query.TSCode, err))
	}

	requestQuery, err := buildDatasourceKLineQuery(secID, query, AdjustNone)
	if err != nil {
		return nil, apperror.New(apperror.CodeBadRequest, fmt.Errorf("build eastmoney kline query: %w", err))
	}

	body, err := client.GetHistory(ctx, historyKLinePath, requestQuery)
	if err != nil {
		return nil, err
	}

	return decodeDatasourceKLines(query.TSCode, query.Interval, body)
}

func fetchParsedKLines(
	ctx context.Context,
	client HistoryClient,
	tsCode string,
	interval Interval,
	adjust AdjustType,
	startTime time.Time,
	endTime time.Time,
	limit int,
) ([]ParsedKLine, error) {
	secID, err := ConvertTSCodeToSecID(tsCode)
	if err != nil {
		return nil, apperror.New(apperror.CodeBadRequest, fmt.Errorf("convert ts_code %s to secid: %w", tsCode, err))
	}

	requestQuery, err := buildRichKLineQuery(secID, interval, adjust, startTime, endTime, limit)
	if err != nil {
		return nil, apperror.New(apperror.CodeBadRequest, fmt.Errorf("build eastmoney rich kline query: %w", err))
	}

	body, err := client.GetHistory(ctx, historyKLinePath, requestQuery)
	if err != nil {
		return nil, err
	}

	var response KLineAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, datasourceUnavailable(fmt.Errorf("decode eastmoney kline response: %w", err))
	}
	if response.RC != 0 {
		return nil, datasourceUnavailable(
			fmt.Errorf("eastmoney kline rc=%d message=%q", response.RC, response.Message),
		)
	}

	parsed, err := ParseKLineRows(tsCode, interval, response.Data.KLines)
	if err != nil {
		return nil, datasourceUnavailable(fmt.Errorf("parse eastmoney kline rows: %w", err))
	}

	return parsed, nil
}

// FetchParsedKLinesForMarketdata 暴露给 richer marketdata 服务复用的底层查询 helper。
func FetchParsedKLinesForMarketdata(
	ctx context.Context,
	client HistoryClient,
	tsCode string,
	interval Interval,
	adjust AdjustType,
	startTime time.Time,
	endTime time.Time,
	limit int,
) ([]ParsedKLine, error) {
	return fetchParsedKLines(ctx, client, tsCode, interval, adjust, startTime, endTime, limit)
}

func decodeDatasourceKLines(tsCode string, interval datasource.Interval, body []byte) ([]datasource.KLine, error) {
	var response KLineAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, datasourceUnavailable(fmt.Errorf("decode eastmoney kline response: %w", err))
	}
	if response.RC != 0 {
		return nil, datasourceUnavailable(
			fmt.Errorf("eastmoney kline rc=%d message=%q", response.RC, response.Message),
		)
	}

	parsed, err := ParseKLineRows(tsCode, Interval(interval), response.Data.KLines)
	if err != nil {
		return nil, datasourceUnavailable(fmt.Errorf("parse eastmoney kline rows: %w", err))
	}

	items := make([]datasource.KLine, 0, len(parsed))
	for _, item := range parsed {
		items = append(items, datasource.KLine{
			TSCode:    tsCode,
			TradeTime: item.TradeTime,
			Open:      item.Open,
			High:      item.High,
			Low:       item.Low,
			Close:     item.Close,
			PreClose:  item.PreClose,
			Change:    item.Change,
			PctChg:    item.PctChg,
			Vol:       item.Vol,
			Amount:    item.Amount,
			Source:    sourceName,
		})
	}

	return items, nil
}

func buildDatasourceKLineQuery(secID string, query datasource.KLineQuery, adjust AdjustType) (url.Values, error) {
	return buildRichKLineQuery(secID, Interval(query.Interval), adjust, query.StartTime, query.EndTime, query.Limit)
}

func buildRichKLineQuery(secID string, interval Interval, adjust AdjustType, startTime, endTime time.Time, limit int) (url.Values, error) {
	klt, err := MapIntervalToEastMoneyKLT(interval)
	if err != nil {
		return nil, fmt.Errorf("map interval %s to eastmoney klt: %w", interval, err)
	}

	query := url.Values{
		"secid":   []string{secID},
		"klt":     []string{klt},
		"fqt":     []string{MapAdjustType(adjust)},
		"fields1": []string{defaultKLineFields1},
		"fields2": []string{defaultKLineFields2},
	}
	if limit > 0 {
		query.Set("beg", "0")
		query.Set("end", formatKLineBoundary(interval, endTime))
		query.Set("lmt", fmt.Sprintf("%d", limit))
		return query, nil
	}

	query.Set("beg", formatKLineBoundary(interval, startTime))
	query.Set("end", formatKLineBoundary(interval, endTime))
	return query, nil
}

func formatKLineBoundary(interval Interval, value time.Time) string {
	if value.IsZero() {
		return "0"
	}
	if isMinuteInterval(interval) {
		return value.In(eastMoneyMarketLocation).Format("2006-01-02 15:04:05")
	}

	return normalizeDate(value).Format(eastMoneyDateCompactLayout)
}
