package job

import (
	"context"
	"fmt"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
)

// JobRunRecorder records job lifecycle state transitions.
type JobRunRecorder interface {
	Start(ctx context.Context, jobName string, bizDate time.Time) error
	Success(ctx context.Context, jobName string, bizDate time.Time) error
	Fail(ctx context.Context, jobName string, bizDate time.Time, err error) error
}

// StockBasicWriter persists stock basic rows.
type StockBasicWriter interface {
	UpsertStockBasics(ctx context.Context, items []datasource.StockBasic) error
}

// ImportStockBasic imports stock basics from a datasource into storage.
func ImportStockBasic(ctx context.Context, recorder JobRunRecorder, writer StockBasicWriter, source datasource.Source, bizDate time.Time) error {
	const jobName = "sync_stock_basic"
	if err := recorder.Start(ctx, jobName, bizDate); err != nil {
		return fmt.Errorf("start stock basic job: %w", err)
	}

	items, err := source.ListStocks(ctx)
	if err != nil {
		if failErr := recorder.Fail(ctx, jobName, bizDate, err); failErr != nil {
			return fmt.Errorf("record stock basic job failure: %w", failErr)
		}
		return fmt.Errorf("list stock basics: %w", err)
	}

	if err := writer.UpsertStockBasics(ctx, items); err != nil {
		if failErr := recorder.Fail(ctx, jobName, bizDate, err); failErr != nil {
			return fmt.Errorf("record stock basic write failure: %w", failErr)
		}
		return fmt.Errorf("upsert stock basics: %w", err)
	}

	if err := recorder.Success(ctx, jobName, bizDate); err != nil {
		return fmt.Errorf("mark stock basic job success: %w", err)
	}

	return nil
}
