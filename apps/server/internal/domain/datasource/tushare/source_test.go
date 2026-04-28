package tushare

import (
	"context"
	"testing"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

func TestSourceWithoutToken(t *testing.T) {
	t.Setenv("TUSHARE_TOKEN", "")
	source := New()

	_, err := source.ListStocks(context.Background())
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListStocks() code = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}

	_, err = source.ListTradeCalendar(context.Background(), "SSE", time.Now(), time.Now())
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListTradeCalendar() code = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}

	_, err = source.ListDailyBars(context.Background(), time.Now(), time.Now())
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListDailyBars() code = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}
}
