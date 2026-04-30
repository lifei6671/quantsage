package sina

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

const sourceName = "sina"

const maxKLineLimit = 1023

// Source 通过新浪页面驱动响应提供单票 K 线能力。
type Source struct {
	browser pageResponseWatcher
}

var _ datasource.Source = (*Source)(nil)

// New 创建新浪 K 线数据源。
func New(browser pageResponseWatcher) *Source {
	return &Source{browser: browser}
}

func newSourceWithWatcher(browser pageResponseWatcher) *Source {
	return &Source{browser: browser}
}

// ListStocks 当前新浪页面驱动源不支持批量股票基础信息导入。
func (s *Source) ListStocks(context.Context) ([]datasource.StockBasic, error) {
	return nil, unsupportedCapabilityError("list stocks")
}

// ListTradeCalendar 当前新浪页面驱动源不支持批量交易日历导入。
func (s *Source) ListTradeCalendar(context.Context, string, time.Time, time.Time) ([]datasource.TradeDay, error) {
	return nil, unsupportedCapabilityError("list trade calendar")
}

// ListDailyBars 当前新浪页面驱动源不支持批量全市场日线导入。
func (s *Source) ListDailyBars(context.Context, time.Time, time.Time) ([]datasource.DailyBar, error) {
	return nil, unsupportedCapabilityError("list daily bars")
}

// ListKLines 消费内部流式响应并收口成标准 K 线列表。
func (s *Source) ListKLines(ctx context.Context, query datasource.KLineQuery) ([]datasource.KLine, error) {
	stream, err := s.StreamKLines(ctx, query)
	if err != nil {
		return nil, err
	}

	normalizedQuery, err := normalizeQuery(query)
	if err != nil {
		return nil, err
	}

	collector := newCollector(normalizedQuery)
	for item := range stream {
		if item.Err != nil {
			return nil, fmt.Errorf("stream sina klines: %w", item.Err)
		}
		collector.Add(item.Items)
	}

	return collector.Finalize(), nil
}

// StreamKLines 持续监听页面响应并把解析后的 K 线批次回推给调用方。
func (s *Source) StreamKLines(ctx context.Context, query datasource.KLineQuery) (<-chan datasource.KLineStreamItem, error) {
	normalizedQuery, err := normalizeQuery(query)
	if err != nil {
		return nil, err
	}

	stream, err := s.watchKLineResponses(ctx, normalizedQuery)
	if err != nil {
		return nil, err
	}

	out := make(chan datasource.KLineStreamItem, 8)
	go func() {
		defer close(out)
		defer stream.Close()

		for item := range stream.Responses {
			if item.Err != nil {
				emitStreamItem(ctx, out, datasource.KLineStreamItem{
					Err: fmt.Errorf("watch sina kline response: %w", item.Err),
				})
				return
			}

			parsed, err := parseKLinesFromResponse(normalizedQuery.TSCode, normalizedQuery.Interval, item.Body)
			if err != nil {
				emitStreamItem(ctx, out, datasource.KLineStreamItem{
					Err: fmt.Errorf("parse sina kline response: %w", err),
				})
				return
			}
			if len(parsed) == 0 {
				continue
			}
			if !emitStreamItem(ctx, out, datasource.KLineStreamItem{Items: parsed}) {
				return
			}
		}

		if err := <-stream.Done; err != nil {
			emitStreamItem(ctx, out, datasource.KLineStreamItem{
				Err: fmt.Errorf("watch sina kline response stream: %w", err),
			})
		}
	}()

	return out, nil
}

func normalizeQuery(query datasource.KLineQuery) (datasource.KLineQuery, error) {
	explicitEndTime := !query.EndTime.IsZero()
	normalizedQuery, err := datasource.NormalizeKLineQuery(query, nil)
	if err != nil {
		return datasource.KLineQuery{}, err
	}
	if _, err := sinaScaleFromInterval(normalizedQuery.Interval); err != nil {
		return datasource.KLineQuery{}, err
	}
	if query.Limit <= 0 {
		return datasource.KLineQuery{}, apperror.New(
			apperror.CodeDatasourceUnavailable,
			errors.New("sina datasource only supports latest-N kline queries"),
		)
	}
	if explicitEndTime {
		return datasource.KLineQuery{}, apperror.New(
			apperror.CodeDatasourceUnavailable,
			errors.New("sina datasource does not support explicit end_time; only latest-N queries ending now are supported"),
		)
	}
	if normalizedQuery.Limit > maxKLineLimit {
		return datasource.KLineQuery{}, apperror.New(
			apperror.CodeDatasourceUnavailable,
			fmt.Errorf("sina datasource limit %d exceeds max supported %d", normalizedQuery.Limit, maxKLineLimit),
		)
	}

	return normalizedQuery, nil
}

func unsupportedCapabilityError(action string) error {
	return apperror.New(
		apperror.CodeDatasourceUnavailable,
		fmt.Errorf("sina datasource does not support %s", action),
	)
}

func emitStreamItem(ctx context.Context, out chan<- datasource.KLineStreamItem, item datasource.KLineStreamItem) bool {
	select {
	case <-ctx.Done():
		return false
	case out <- item:
		return true
	}
}

func browserUnavailableError() error {
	return apperror.New(
		apperror.CodeDatasourceUnavailable,
		errors.New("sina datasource browser watcher is not configured"),
	)
}
