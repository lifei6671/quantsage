package tushare

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

// Source 是 V1 阶段用于接线的 Tushare 占位数据源。
type Source struct{}

// New 创建一个 Tushare 占位数据源。
func New() *Source {
	return &Source{}
}

// ListStocks 在缺少 Token 时返回数据源不可用错误。
func (s *Source) ListStocks(ctx context.Context) ([]datasource.StockBasic, error) {
	if token := os.Getenv("TUSHARE_TOKEN"); token == "" {
		return nil, apperror.New(apperror.CodeDatasourceUnavailable, errors.New("tushare token is empty"))
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return nil, apperror.New(apperror.CodeDatasourceUnavailable, errors.New("tushare integration is not implemented"))
}

// ListTradeCalendar 在缺少 Token 时返回数据源不可用错误。
func (s *Source) ListTradeCalendar(ctx context.Context, exchange string, startDate, endDate time.Time) ([]datasource.TradeDay, error) {
	if token := os.Getenv("TUSHARE_TOKEN"); token == "" {
		return nil, apperror.New(apperror.CodeDatasourceUnavailable, errors.New("tushare token is empty"))
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return nil, apperror.New(apperror.CodeDatasourceUnavailable, errors.New("tushare integration is not implemented"))
}

// ListDailyBars 在缺少 Token 时返回数据源不可用错误。
func (s *Source) ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]datasource.DailyBar, error) {
	if token := os.Getenv("TUSHARE_TOKEN"); token == "" {
		return nil, apperror.New(apperror.CodeDatasourceUnavailable, errors.New("tushare token is empty"))
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return nil, apperror.New(apperror.CodeDatasourceUnavailable, errors.New("tushare integration is not implemented"))
}
