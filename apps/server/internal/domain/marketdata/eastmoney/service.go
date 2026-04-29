package eastmoney

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"

	datasourceeastmoney "github.com/lifei6671/quantsage/apps/server/internal/domain/datasource/eastmoney"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/consts"
)

const (
	defaultQueryLimit       = 120
	defaultBatchConcurrency = 8
)

type Interval = datasourceeastmoney.Interval

const (
	Interval1Min    = datasourceeastmoney.Interval1Min
	Interval5Min    = datasourceeastmoney.Interval5Min
	Interval15Min   = datasourceeastmoney.Interval15Min
	Interval30Min   = datasourceeastmoney.Interval30Min
	Interval60Min   = datasourceeastmoney.Interval60Min
	IntervalDay     = datasourceeastmoney.IntervalDay
	IntervalWeek    = datasourceeastmoney.IntervalWeek
	IntervalMonth   = datasourceeastmoney.IntervalMonth
	IntervalQuarter = datasourceeastmoney.IntervalQuarter
	IntervalYear    = datasourceeastmoney.IntervalYear
)

type AdjustType = datasourceeastmoney.AdjustType

const (
	AdjustNone = datasourceeastmoney.AdjustNone
	AdjustQFQ  = datasourceeastmoney.AdjustQFQ
	AdjustHFQ  = datasourceeastmoney.AdjustHFQ
)

// KLine 表示 richer K 线查询返回的标准结构。
type KLine struct {
	TSCode       string
	TradeTime    time.Time
	Open         decimal.Decimal
	High         decimal.Decimal
	Low          decimal.Decimal
	Close        decimal.Decimal
	PreClose     decimal.Decimal
	Change       decimal.Decimal
	PctChg       decimal.Decimal
	Vol          decimal.Decimal
	Amount       decimal.Decimal
	TurnoverRate decimal.Decimal
	Source       string
}

// MAValue 表示单条均线值。
type MAValue struct {
	Period int
	Value  decimal.Decimal
}

// KLineWithMA 表示附带均线结果的 K 线项。
type KLineWithMA struct {
	KLine
	MovingAverages []MAValue
}

// Query 定义 richer K 线查询条件。
type Query struct {
	TSCode   string
	Interval Interval
	Adjust   AdjustType
	Limit    int
	EndTime  time.Time
}

// Service 定义东财 richer K 线查询契约。
type Service interface {
	ListKLines(ctx context.Context, query Query) ([]KLine, error)
	GetLatestKLine(ctx context.Context, query Query) (KLine, error)
	BatchListKLines(ctx context.Context, queries []Query) (map[string][]KLine, error)
	ListKLinesWithMA(ctx context.Context, query Query, periods []int) ([]KLineWithMA, error)
}

type service struct {
	client datasourceeastmoney.HistoryClient
}

// NewFromClientConfig 根据底层 client 配置创建 richer 行情服务。
func NewFromClientConfig(cfg datasourceeastmoney.ClientConfig) Service {
	return &service{
		client: datasourceeastmoney.NewHistoryClientFromClientConfig(cfg),
	}
}

// NewFromConfig 根据带回退策略的 datasource 配置创建 richer 行情服务。
func NewFromConfig(cfg datasourceeastmoney.Config) Service {
	return &service{
		client: datasourceeastmoney.NewHistoryClientFromConfig(cfg),
	}
}

func newServiceWithClient(client datasourceeastmoney.HistoryClient) Service {
	return &service{client: client}
}

func (s *service) ListKLines(ctx context.Context, query Query) ([]KLine, error) {
	normalizedQuery, err := normalizeQuery(query)
	if err != nil {
		return nil, err
	}

	secID, err := datasourceeastmoney.ConvertTSCodeToSecID(normalizedQuery.TSCode)
	if err != nil {
		return nil, apperror.New(apperror.CodeBadRequest, fmt.Errorf("convert ts_code to secid: %w", err))
	}
	klt, err := datasourceeastmoney.MapIntervalToEastMoneyKLT(normalizedQuery.Interval)
	if err != nil {
		return nil, apperror.New(apperror.CodeBadRequest, fmt.Errorf("map interval: %w", err))
	}

	body, err := s.client.GetHistory(ctx, "/api/qt/stock/kline/get", url.Values{
		"secid":   []string{secID},
		"klt":     []string{klt},
		"fqt":     []string{datasourceeastmoney.MapAdjustType(normalizedQuery.Adjust)},
		"fields1": []string{"f1,f2,f3,f4,f5,f6"},
		"fields2": []string{"f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61"},
		"beg":     []string{"0"},
		"end":     []string{formatQueryEnd(normalizedQuery.Interval, normalizedQuery.EndTime)},
		"lmt":     []string{fmt.Sprintf("%d", normalizedQuery.Limit)},
	})
	if err != nil {
		return nil, err
	}

	var response datasourceeastmoney.KLineAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, apperror.New(apperror.CodeDatasourceUnavailable, fmt.Errorf("decode eastmoney kline response: %w", err))
	}
	if response.RC != 0 {
		return nil, apperror.New(
			apperror.CodeDatasourceUnavailable,
			fmt.Errorf("eastmoney kline rc=%d message=%q", response.RC, strings.TrimSpace(response.Message)),
		)
	}

	parsed, err := datasourceeastmoney.ParseKLineRows(normalizedQuery.TSCode, normalizedQuery.Interval, response.Data.KLines)
	if err != nil {
		return nil, apperror.New(apperror.CodeDatasourceUnavailable, fmt.Errorf("parse eastmoney kline rows: %w", err))
	}

	items := make([]KLine, 0, len(parsed))
	for _, item := range parsed {
		items = append(items, KLine{
			TSCode:       normalizedQuery.TSCode,
			TradeTime:    item.TradeTime,
			Open:         item.Open,
			High:         item.High,
			Low:          item.Low,
			Close:        item.Close,
			PreClose:     item.PreClose,
			Change:       item.Change,
			PctChg:       item.PctChg,
			Vol:          item.Vol,
			Amount:       item.Amount,
			TurnoverRate: item.TurnoverRate,
			Source:       consts.DatasourceEastMoney,
		})
	}

	return items, nil
}

func (s *service) GetLatestKLine(ctx context.Context, query Query) (KLine, error) {
	query.Limit = 1
	items, err := s.ListKLines(ctx, query)
	if err != nil {
		return KLine{}, err
	}
	if len(items) == 0 {
		return KLine{}, apperror.New(apperror.CodeNotFound, errors.New("eastmoney latest kline not found"))
	}

	return items[len(items)-1], nil
}

func (s *service) BatchListKLines(ctx context.Context, queries []Query) (map[string][]KLine, error) {
	result := make(map[string][]KLine, len(queries))
	if len(queries) == 0 {
		return result, nil
	}
	if err := validateBatchQueries(queries); err != nil {
		return nil, err
	}

	var (
		resultMu sync.Mutex
		errMu    sync.Mutex
		errs     []error
	)
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(defaultBatchConcurrency)
	for _, query := range queries {
		query := query
		group.Go(func() error {
			items, err := s.ListKLines(groupCtx, query)
			if err != nil {
				errMu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", strings.TrimSpace(query.TSCode), err))
				errMu.Unlock()
				return nil
			}

			resultMu.Lock()
			result[strings.TrimSpace(query.TSCode)] = items
			resultMu.Unlock()
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	if len(errs) > 0 {
		return result, errors.Join(errs...)
	}

	return result, nil
}

func validateBatchQueries(queries []Query) error {
	seen := make(map[string]struct{}, len(queries))
	for _, query := range queries {
		tsCode := strings.ToUpper(strings.TrimSpace(query.TSCode))
		if tsCode == "" {
			return apperror.New(apperror.CodeBadRequest, errors.New("ts_code is required"))
		}
		if _, exists := seen[tsCode]; exists {
			return apperror.New(
				apperror.CodeBadRequest,
				fmt.Errorf("duplicate ts_code %q in batch query is unsupported", tsCode),
			)
		}
		seen[tsCode] = struct{}{}
	}

	return nil
}

func (s *service) ListKLinesWithMA(ctx context.Context, query Query, periods []int) ([]KLineWithMA, error) {
	items, err := s.ListKLines(ctx, query)
	if err != nil {
		return nil, err
	}

	return AttachSimpleMovingAverages(items, periods), nil
}

func normalizeQuery(query Query) (Query, error) {
	query.TSCode = strings.ToUpper(strings.TrimSpace(query.TSCode))
	if query.TSCode == "" {
		return Query{}, apperror.New(apperror.CodeBadRequest, errors.New("ts_code is required"))
	}
	if query.Interval == "" {
		query.Interval = IntervalDay
	}
	if query.Adjust == "" {
		query.Adjust = AdjustNone
	}
	if query.Limit <= 0 {
		query.Limit = defaultQueryLimit
	}
	if query.EndTime.IsZero() {
		query.EndTime = time.Now().UTC()
	} else {
		query.EndTime = query.EndTime.UTC()
	}

	return query, nil
}

func formatQueryEnd(interval Interval, value time.Time) string {
	if value.IsZero() {
		return "20500101"
	}

	if isMinuteInterval(interval) {
		return value.UTC().Format("2006-01-02 15:04:05")
	}

	return value.UTC().Format("20060102")
}

func isMinuteInterval(interval Interval) bool {
	switch interval {
	case Interval1Min, Interval5Min, Interval15Min, Interval30Min, Interval60Min:
		return true
	default:
		return false
	}
}
