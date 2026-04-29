package eastmoney

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	secID, err := ConvertTSCodeToSecID(tsCode)
	if err != nil {
		return nil, datasourceUnavailable(fmt.Errorf("convert ts_code %s to secid: %w", tsCode, err))
	}

	body, err := s.fallbackClient.GetHistory(ctx, historyKLinePath, buildKLineQuery(secID, IntervalDay, AdjustNone, startDate, endDate, 0))
	if err != nil {
		return nil, err
	}

	var response KLineAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, datasourceUnavailable(fmt.Errorf("decode eastmoney daily response: %w", err))
	}
	if response.RC != 0 {
		return nil, datasourceUnavailable(
			fmt.Errorf("eastmoney daily rc=%d message=%q", response.RC, strings.TrimSpace(response.Message)),
		)
	}

	parsed, err := ParseKLineRows(tsCode, IntervalDay, response.Data.KLines)
	if err != nil {
		return nil, datasourceUnavailable(fmt.Errorf("parse eastmoney daily rows: %w", err))
	}

	items := make([]datasource.DailyBar, 0, len(parsed))
	for _, item := range parsed {
		tradeDate := normalizeDate(item.TradeTime)
		if tradeDate.Before(startDate) || tradeDate.After(endDate) {
			continue
		}
		items = append(items, datasource.DailyBar{
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

	return items, nil
}
