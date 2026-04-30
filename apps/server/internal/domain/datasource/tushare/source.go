package tushare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/consts"
)

const (
	defaultEndpoint = "http://api.tushare.pro"
	sourceName      = consts.DatasourceTushare
	tushareDate     = "20060102"
	responseLimit   = 32 << 20
	httpTimeout     = 30 * time.Second
)

var (
	stockBasicFields = []string{"ts_code", "symbol", "name", "area", "industry", "market", "exchange", "list_date"}
	tradeCalFields   = []string{"exchange", "cal_date", "is_open", "pretrade_date"}
	dailyFields      = []string{"ts_code", "trade_date", "open", "high", "low", "close", "pre_close", "change", "pct_chg", "vol", "amount"}
)

// Source 通过 Tushare Pro HTTP API 读取外部行情数据。
type Source struct {
	token      string
	endpoint   string
	httpClient *http.Client
}

// Option 定义 Tushare 数据源的可选配置，主要用于测试和本地诊断。
type Option func(*Source)

// WithEndpoint 覆盖 Tushare API 地址。
func WithEndpoint(endpoint string) Option {
	return func(source *Source) {
		source.endpoint = strings.TrimSpace(endpoint)
	}
}

// WithHTTPClient 覆盖 HTTP Client。
func WithHTTPClient(client *http.Client) Option {
	return func(source *Source) {
		if client != nil {
			source.httpClient = client
		}
	}
}

// New 创建一个 Tushare Pro 数据源。
func New(token string, options ...Option) *Source {
	source := &Source{
		token:      strings.TrimSpace(token),
		endpoint:   defaultEndpoint,
		httpClient: &http.Client{Timeout: httpTimeout},
	}
	for _, option := range options {
		if option != nil {
			option(source)
		}
	}
	if source.endpoint == "" {
		source.endpoint = defaultEndpoint
	}

	return source
}

// ListStocks 读取当前正常上市股票基础信息。
func (s *Source) ListStocks(ctx context.Context) ([]datasource.StockBasic, error) {
	if err := s.requireToken(); err != nil {
		return nil, err
	}

	data, err := s.query(ctx, "stock_basic", map[string]string{
		"exchange":    "",
		"list_status": "L",
	}, stockBasicFields)
	if err != nil {
		return nil, err
	}

	items := make([]datasource.StockBasic, 0, len(data.Items))
	for _, rawItem := range data.Items {
		row, err := buildRow(data.Fields, rawItem)
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("map tushare stock_basic row: %w", err))
		}
		listDate, err := parseDateField(row, "list_date")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare stock list_date: %w", err))
		}
		items = append(items, datasource.StockBasic{
			TSCode:   stringField(row, "ts_code"),
			Symbol:   stringField(row, "symbol"),
			Name:     stringField(row, "name"),
			Area:     stringField(row, "area"),
			Industry: stringField(row, "industry"),
			Market:   stringField(row, "market"),
			Exchange: stringField(row, "exchange"),
			ListDate: listDate,
			Source:   sourceName,
		})
	}

	return items, nil
}

// ListTradeCalendar 读取指定交易所的交易日历。
func (s *Source) ListTradeCalendar(ctx context.Context, exchange string, startDate, endDate time.Time) ([]datasource.TradeDay, error) {
	if err := s.requireToken(); err != nil {
		return nil, err
	}

	data, err := s.query(ctx, "trade_cal", map[string]string{
		"exchange":   strings.TrimSpace(exchange),
		"start_date": formatDate(startDate),
		"end_date":   formatDate(endDate),
	}, tradeCalFields)
	if err != nil {
		return nil, err
	}

	items := make([]datasource.TradeDay, 0, len(data.Items))
	for _, rawItem := range data.Items {
		row, err := buildRow(data.Fields, rawItem)
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("map tushare trade_cal row: %w", err))
		}
		calDate, err := parseDateField(row, "cal_date")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare trade cal_date: %w", err))
		}
		pretradeDate, err := parseOptionalDateField(row, "pretrade_date")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare trade pretrade_date: %w", err))
		}
		items = append(items, datasource.TradeDay{
			Exchange:     stringField(row, "exchange"),
			CalDate:      calDate,
			IsOpen:       stringField(row, "is_open") == "1",
			PretradeDate: pretradeDate,
			Source:       sourceName,
		})
	}

	return items, nil
}

// ListDailyBars 读取 A 股日线行情。
func (s *Source) ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]datasource.DailyBar, error) {
	if err := s.requireToken(); err != nil {
		return nil, err
	}

	params := map[string]string{
		"start_date": formatDate(startDate),
		"end_date":   formatDate(endDate),
	}
	if sameDate(startDate, endDate) {
		params = map[string]string{"trade_date": formatDate(startDate)}
	}

	data, err := s.query(ctx, "daily", params, dailyFields)
	if err != nil {
		return nil, err
	}

	items := make([]datasource.DailyBar, 0, len(data.Items))
	for _, rawItem := range data.Items {
		row, err := buildRow(data.Fields, rawItem)
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("map tushare daily row: %w", err))
		}
		tradeDate, err := parseDateField(row, "trade_date")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily trade_date: %w", err))
		}
		open, err := decimalField(row, "open")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily open: %w", err))
		}
		high, err := decimalField(row, "high")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily high: %w", err))
		}
		low, err := decimalField(row, "low")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily low: %w", err))
		}
		closePrice, err := decimalField(row, "close")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily close: %w", err))
		}
		preClose, err := decimalField(row, "pre_close")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily pre_close: %w", err))
		}
		change, err := decimalField(row, "change")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily change: %w", err))
		}
		pctChg, err := decimalField(row, "pct_chg")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily pct_chg: %w", err))
		}
		vol, err := decimalField(row, "vol")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily vol: %w", err))
		}
		amount, err := decimalField(row, "amount")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily amount: %w", err))
		}

		items = append(items, datasource.DailyBar{
			TSCode:    stringField(row, "ts_code"),
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
			Source:    sourceName,
		})
	}

	return items, nil
}

// ListKLines 返回 Tushare 当前支持的单票日线 K 线。
func (s *Source) ListKLines(ctx context.Context, query datasource.KLineQuery) ([]datasource.KLine, error) {
	if err := s.requireToken(); err != nil {
		return nil, err
	}

	normalizedQuery, err := datasource.NormalizeKLineQuery(query, time.Now)
	if err != nil {
		return nil, err
	}
	if normalizedQuery.Interval != datasource.IntervalDay {
		return nil, apperror.New(
			apperror.CodeDatasourceUnavailable,
			fmt.Errorf("tushare datasource does not support interval %s", normalizedQuery.Interval),
		)
	}

	params := buildDailyKLineParams(normalizedQuery)
	data, err := s.query(ctx, "daily", params, dailyFields)
	if err != nil {
		return nil, err
	}

	items, err := mapTushareDailyDataToKLines(normalizedQuery, data)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].TradeTime.Before(items[j].TradeTime)
	})

	return datasource.TrimKLinesByLimit(items, normalizedQuery.Limit), nil
}

// StreamKLines 当前 Tushare 数据源不支持流式 K 线接口。
func (s *Source) StreamKLines(context.Context, datasource.KLineQuery) (<-chan datasource.KLineStreamItem, error) {
	return nil, datasource.UnsupportedStreamError("tushare")
}

func (s *Source) requireToken() error {
	if s.token != "" {
		return nil
	}

	return apperror.New(apperror.CodeDatasourceUnavailable, errors.New("tushare token is empty"))
}

func (s *Source) query(ctx context.Context, apiName string, params map[string]string, fields []string) (tushareData, error) {
	requestBody := tushareRequest{
		APIName: apiName,
		Token:   s.token,
		Params:  params,
		Fields:  strings.Join(fields, ","),
	}
	encodedBody, err := json.Marshal(requestBody)
	if err != nil {
		return tushareData{}, fmt.Errorf("marshal tushare request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(encodedBody))
	if err != nil {
		return tushareData{}, fmt.Errorf("build tushare request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return tushareData{}, apperror.New(apperror.CodeDatasourceUnavailable, fmt.Errorf("call tushare %s: %w", apiName, err))
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, responseLimit))
		return tushareData{}, apperror.New(apperror.CodeDatasourceUnavailable, fmt.Errorf("call tushare %s: http status %d", apiName, resp.StatusCode))
	}

	var response tushareResponse
	decoder := json.NewDecoder(io.LimitReader(resp.Body, responseLimit))
	decoder.UseNumber()
	if err := decoder.Decode(&response); err != nil {
		return tushareData{}, datasourceUnavailable(fmt.Errorf("decode tushare %s response: %w", apiName, err))
	}
	if response.Code != 0 {
		return tushareData{}, apperror.New(apperror.CodeDatasourceUnavailable, fmt.Errorf("tushare %s error %d: %s", apiName, response.Code, response.Msg))
	}

	return response.Data, nil
}

func datasourceUnavailable(err error) error {
	return apperror.New(apperror.CodeDatasourceUnavailable, err)
}

type tushareRequest struct {
	APIName string            `json:"api_name"`
	Token   string            `json:"token"`
	Params  map[string]string `json:"params"`
	Fields  string            `json:"fields"`
}

type tushareResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data tushareData `json:"data"`
}

type tushareData struct {
	Fields []string `json:"fields"`
	Items  [][]any  `json:"items"`
}

func buildRow(fields []string, item []any) (map[string]any, error) {
	if len(fields) != len(item) {
		return nil, fmt.Errorf("field count %d does not match item count %d", len(fields), len(item))
	}
	row := make(map[string]any, len(fields))
	for index, field := range fields {
		row[field] = item[index]
	}
	return row, nil
}

func stringField(row map[string]any, field string) string {
	value, ok := row[field]
	if !ok || value == nil {
		return ""
	}
	switch typedValue := value.(type) {
	case string:
		return strings.TrimSpace(typedValue)
	case json.Number:
		return typedValue.String()
	case float64:
		return decimal.NewFromFloat(typedValue).String()
	case bool:
		if typedValue {
			return "1"
		}
		return "0"
	default:
		return strings.TrimSpace(fmt.Sprint(typedValue))
	}
}

func decimalField(row map[string]any, field string) (decimal.Decimal, error) {
	text := stringField(row, field)
	if text == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(text)
}

func parseDateField(row map[string]any, field string) (time.Time, error) {
	text := stringField(row, field)
	if text == "" {
		return time.Time{}, nil
	}
	return time.Parse(tushareDate, text)
}

func parseOptionalDateField(row map[string]any, field string) (time.Time, error) {
	return parseDateField(row, field)
}

func formatDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(tushareDate)
}

func sameDate(left, right time.Time) bool {
	if left.IsZero() || right.IsZero() {
		return false
	}
	return formatDate(left) == formatDate(right)
}

func buildDailyKLineParams(query datasource.KLineQuery) map[string]string {
	params := map[string]string{
		"ts_code": strings.TrimSpace(query.TSCode),
	}
	if query.Limit > 0 {
		params["end_date"] = formatDate(query.EndTime)
		return params
	}

	params["start_date"] = formatDate(query.StartTime)
	params["end_date"] = formatDate(query.EndTime)
	return params
}

func mapTushareDailyDataToKLines(query datasource.KLineQuery, data tushareData) ([]datasource.KLine, error) {
	items := make([]datasource.KLine, 0, len(data.Items))
	for _, rawItem := range data.Items {
		row, err := buildRow(data.Fields, rawItem)
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("map tushare daily row: %w", err))
		}
		if tsCode := stringField(row, "ts_code"); tsCode != query.TSCode {
			continue
		}

		tradeDate, err := parseDateField(row, "trade_date")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily trade_date: %w", err))
		}
		if query.Limit <= 0 && (tradeDate.Before(query.StartTime) || tradeDate.After(query.EndTime)) {
			continue
		}
		if query.Limit > 0 && tradeDate.After(query.EndTime) {
			continue
		}

		open, err := decimalField(row, "open")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily open: %w", err))
		}
		high, err := decimalField(row, "high")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily high: %w", err))
		}
		low, err := decimalField(row, "low")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily low: %w", err))
		}
		closePrice, err := decimalField(row, "close")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily close: %w", err))
		}
		preClose, err := decimalField(row, "pre_close")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily pre_close: %w", err))
		}
		change, err := decimalField(row, "change")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily change: %w", err))
		}
		pctChg, err := decimalField(row, "pct_chg")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily pct_chg: %w", err))
		}
		vol, err := decimalField(row, "vol")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily vol: %w", err))
		}
		amount, err := decimalField(row, "amount")
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("parse tushare daily amount: %w", err))
		}

		items = append(items, datasource.KLine{
			TSCode:    query.TSCode,
			TradeTime: tradeDate,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			PreClose:  preClose,
			Change:    change,
			PctChg:    pctChg,
			Vol:       vol,
			Amount:    amount,
			Source:    sourceName,
		})
	}

	return items, nil
}
