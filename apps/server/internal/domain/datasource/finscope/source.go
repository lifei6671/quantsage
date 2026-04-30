package finscope

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
	"github.com/lifei6671/quantsage/apps/server/internal/infra/browserfetch"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

// Source 表示 Finscope 浏览器驱动型数据源骨架。
type Source struct {
	browser browserfetch.Runner
	cfg     Config
}

var _ datasource.Source = (*Source)(nil)

// New 创建一个 Finscope 数据源实例。
func New(browser browserfetch.Runner, opts ...Option) *Source {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return &Source{
		browser: browser,
		cfg:     cfg,
	}
}

// ListStocks 串行执行市场股票列表子抓取链路。
func (s *Source) ListStocks(ctx context.Context) ([]datasource.StockBasic, error) {
	loaders := []struct {
		name string
		fn   func(context.Context) ([]datasource.StockBasic, error)
	}{
		{name: "sh-index-constituents", fn: s.listSHIndexConstituentStocks},
	}

	result := make([]datasource.StockBasic, 0, defaultConstituentPageSize)
	seen := make(map[string]struct{})
	for _, loader := range loaders {
		items, err := loader.fn(ctx)
		if err != nil {
			return nil, fmt.Errorf("list finscope stocks via %s: %w", loader.name, err)
		}
		for _, item := range items {
			tsCode := item.TSCode
			if tsCode == "" {
				continue
			}
			if _, ok := seen[tsCode]; ok {
				continue
			}
			seen[tsCode] = struct{}{}
			result = append(result, item)
		}
	}

	return slices.Clone(result), nil
}

// ListTradeCalendar 当前骨架阶段尚未实现交易日历采集。
func (s *Source) ListTradeCalendar(context.Context, string, time.Time, time.Time) ([]datasource.TradeDay, error) {
	return nil, notImplementedError("list trade calendar")
}

// ListDailyBars 当前骨架阶段尚未实现批量日线采集。
func (s *Source) ListDailyBars(context.Context, time.Time, time.Time) ([]datasource.DailyBar, error) {
	return nil, notImplementedError("list daily bars")
}

// ListKLines 当前骨架阶段尚未实现单票 K 线查询。
func (s *Source) ListKLines(context.Context, datasource.KLineQuery) ([]datasource.KLine, error) {
	return nil, notImplementedError("list klines")
}

// StreamKLines 当前骨架阶段尚未实现流式 K 线查询。
func (s *Source) StreamKLines(context.Context, datasource.KLineQuery) (<-chan datasource.KLineStreamItem, error) {
	return nil, notImplementedError("stream klines")
}

func notImplementedError(action string) error {
	return apperror.New(
		apperror.CodeDatasourceUnavailable,
		fmt.Errorf("finscope datasource %s is not implemented yet", action),
	)
}

func (s *Source) listSHIndexConstituentStocks(ctx context.Context) ([]datasource.StockBasic, error) {
	if s == nil || s.browser == nil {
		return nil, browserUnavailableError()
	}

	query := defaultSHIndexConstituentQuery()
	bodies, err := s.watchConstituentPageBodies(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("watch constituents page: %w", err)
	}

	result := make([]datasource.StockBasic, 0, len(bodies)*defaultConstituentPageSize)
	for index, body := range bodies {
		items, err := parseStockBasicsFromConstituentResponse(body)
		if err != nil {
			return nil, fmt.Errorf("parse constituents response %d: %w", index, err)
		}
		result = append(result, items...)
	}

	return result, nil
}
