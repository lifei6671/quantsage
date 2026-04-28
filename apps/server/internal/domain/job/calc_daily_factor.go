package job

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/indicator"
	"github.com/lifei6671/quantsage/apps/server/internal/domain/marketdata"
)

// DailyBarReader 按日期范围读取日线行情。
type DailyBarReader interface {
	ListStockDaily(ctx context.Context, startDate, endDate time.Time) ([]marketdata.DailyBar, error)
}

// DailyFactorWriter 持久化每日因子结果。
type DailyFactorWriter interface {
	UpsertDailyFactors(ctx context.Context, items []indicator.DailyFactor) error
}

// CalcDailyFactor 按股票分组计算指定区间内的日线因子。
func CalcDailyFactor(ctx context.Context, recorder JobRunRecorder, reader DailyBarReader, writer DailyFactorWriter, startDate, endDate, bizDate time.Time) error {
	const jobName = "calc_daily_factor"
	if err := recorder.Start(ctx, jobName, bizDate); err != nil {
		return fmt.Errorf("start daily factor job: %w", err)
	}

	bars, err := reader.ListStockDaily(ctx, startDate, endDate)
	if err != nil {
		if failErr := recorder.Fail(ctx, jobName, bizDate, err); failErr != nil {
			return fmt.Errorf("record daily factor read failure: %w", failErr)
		}
		return fmt.Errorf("list stock daily for factor job: %w", err)
	}

	grouped := groupBarsByTSCode(bars)
	factors := make([]indicator.DailyFactor, 0, len(bars))
	for _, tsCode := range sortedKeys(grouped) {
		items := grouped[tsCode]
		sort.Slice(items, func(i, j int) bool {
			return items[i].TradeDate.Before(items[j].TradeDate)
		})

		dailyFactors, calcErr := indicator.CalculateDailyFactors(items)
		if calcErr != nil {
			if failErr := recorder.Fail(ctx, jobName, bizDate, calcErr); failErr != nil {
				return fmt.Errorf("record daily factor calculation failure: %w", failErr)
			}
			return fmt.Errorf("calculate daily factors for %s: %w", tsCode, calcErr)
		}
		factors = append(factors, dailyFactors...)
	}

	if err := writer.UpsertDailyFactors(ctx, factors); err != nil {
		if failErr := recorder.Fail(ctx, jobName, bizDate, err); failErr != nil {
			return fmt.Errorf("record daily factor write failure: %w", failErr)
		}
		return fmt.Errorf("upsert daily factors: %w", err)
	}

	if err := recorder.Success(ctx, jobName, bizDate); err != nil {
		return fmt.Errorf("mark daily factor job success: %w", err)
	}

	return nil
}

func groupBarsByTSCode(items []marketdata.DailyBar) map[string][]marketdata.DailyBar {
	grouped := make(map[string][]marketdata.DailyBar, len(items))
	for _, item := range items {
		grouped[item.TSCode] = append(grouped[item.TSCode], item)
	}

	return grouped
}

func sortedKeys(grouped map[string][]marketdata.DailyBar) []string {
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}
