package job

import (
	"context"
	"fmt"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
)

// TradeCalendarWriter persists trade calendar rows.
type TradeCalendarWriter interface {
	UpsertTradeCalendar(ctx context.Context, items []datasource.TradeDay) error
}

// ImportTradeCalendar imports trade calendar rows from a datasource.
func ImportTradeCalendar(ctx context.Context, recorder JobRunRecorder, writer TradeCalendarWriter, source datasource.Source, exchange string, startDate, endDate, bizDate time.Time) error {
	const jobName = "sync_trade_calendar"
	if err := recorder.Start(ctx, jobName, bizDate); err != nil {
		return fmt.Errorf("start trade calendar job: %w", err)
	}

	items, err := source.ListTradeCalendar(ctx, exchange, startDate, endDate)
	if err != nil {
		if failErr := recorder.Fail(ctx, jobName, bizDate, err); failErr != nil {
			return fmt.Errorf("record trade calendar job failure: %w", failErr)
		}
		return fmt.Errorf("list trade calendar: %w", err)
	}

	if err := writer.UpsertTradeCalendar(ctx, items); err != nil {
		if failErr := recorder.Fail(ctx, jobName, bizDate, err); failErr != nil {
			return fmt.Errorf("record trade calendar write failure: %w", failErr)
		}
		return fmt.Errorf("upsert trade calendar: %w", err)
	}

	if err := recorder.Success(ctx, jobName, bizDate); err != nil {
		return fmt.Errorf("mark trade calendar job success: %w", err)
	}

	return nil
}
