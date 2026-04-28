package job

import (
	"context"
	"fmt"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
)

// StockDailyWriter persists stock daily rows.
type StockDailyWriter interface {
	UpsertStockDaily(ctx context.Context, items []datasource.DailyBar) error
}

// ImportStockDaily imports stock daily bars from a datasource.
func ImportStockDaily(ctx context.Context, recorder JobRunRecorder, writer StockDailyWriter, source datasource.Source, startDate, endDate, bizDate time.Time) error {
	const jobName = "sync_daily_market"
	if err := recorder.Start(ctx, jobName, bizDate); err != nil {
		return fmt.Errorf("start stock daily job: %w", err)
	}

	items, err := source.ListDailyBars(ctx, startDate, endDate)
	if err != nil {
		if failErr := recorder.Fail(ctx, jobName, bizDate, err); failErr != nil {
			return fmt.Errorf("record stock daily job failure: %w", failErr)
		}
		return fmt.Errorf("list stock daily bars: %w", err)
	}

	if err := writer.UpsertStockDaily(ctx, items); err != nil {
		if failErr := recorder.Fail(ctx, jobName, bizDate, err); failErr != nil {
			return fmt.Errorf("record stock daily write failure: %w", failErr)
		}
		return fmt.Errorf("upsert stock daily bars: %w", err)
	}

	if err := recorder.Success(ctx, jobName, bizDate); err != nil {
		return fmt.Errorf("mark stock daily job success: %w", err)
	}

	return nil
}
