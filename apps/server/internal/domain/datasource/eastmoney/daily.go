package eastmoney

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
)

// ListDailyBars 读取 A 股全市场日线行情。
func (s *Source) ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]datasource.DailyBar, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	startDate, endDate = normalizeDateRange(startDate, endDate)

	stocks, err := s.ListStocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("list eastmoney stocks before daily sync: %w", err)
	}

	result := make([]datasource.DailyBar, 0, len(stocks))
	var resultMu sync.Mutex

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(defaultDailyFetchConcurrency)
	for _, stock := range stocks {
		stock := stock
		group.Go(func() error {
			items, itemErr := s.listDailyBarsByTSCode(groupCtx, stock.TSCode, startDate, endDate)
			if itemErr != nil {
				return fmt.Errorf("list eastmoney daily bars for %s: %w", stock.TSCode, itemErr)
			}

			resultMu.Lock()
			result = append(result, items...)
			resultMu.Unlock()
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Source) listDailyBarsByTSCode(ctx context.Context, tsCode string, startDate, endDate time.Time) ([]datasource.DailyBar, error) {
	items, err := s.ListKLines(ctx, datasource.KLineQuery{
		TSCode:    tsCode,
		Interval:  datasource.IntervalDay,
		StartTime: startDate,
		EndTime:   endDate,
	})
	if err != nil {
		return nil, err
	}

	result := make([]datasource.DailyBar, 0, len(items))
	for _, item := range items {
		tradeDate := normalizeDate(item.TradeTime)
		if tradeDate.Before(startDate) || tradeDate.After(endDate) {
			continue
		}
		result = append(result, datasource.DailyBar{
			TSCode:    tsCode,
			TradeDate: tradeDate,
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

	return result, nil
}
