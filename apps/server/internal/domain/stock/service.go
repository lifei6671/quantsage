package stock

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/infra/db/dbgen"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// Stock 表示股票基础信息。
type Stock struct {
	TSCode   string
	Symbol   string
	Name     string
	Industry string
	Exchange string
	IsActive bool
}

// DailyBar 表示股票日线信息。
type DailyBar struct {
	TSCode    string
	TradeDate time.Time
	Open      string
	High      string
	Low       string
	Close     string
	PctChg    string
	Vol       string
	Amount    string
}

// ListParams 定义股票列表查询条件。
type ListParams struct {
	Keyword  string
	Page     int
	PageSize int
}

// DailyParams 定义股票日线查询条件。
type DailyParams struct {
	TSCode    string
	StartDate time.Time
	EndDate   time.Time
}

// ListResult 表示分页股票列表结果。
type ListResult struct {
	Items    []Stock
	Page     int
	PageSize int
}

// Service 定义股票读服务契约。
type Service interface {
	ListStocks(ctx context.Context, params ListParams) (ListResult, error)
	GetStock(ctx context.Context, tsCode string) (Stock, error)
	ListDailyBars(ctx context.Context, params DailyParams) ([]DailyBar, error)
}

// Querier 定义股票查询依赖。
type Querier interface {
	ListStocks(ctx context.Context, arg dbgen.ListStocksParams) ([]dbgen.ListStocksRow, error)
	GetStock(ctx context.Context, tsCode string) (dbgen.GetStockRow, error)
	ListStockDaily(ctx context.Context, arg dbgen.ListStockDailyParams) ([]dbgen.ListStockDailyRow, error)
}

type service struct {
	querier Querier
}

// NewService 创建股票读服务。
func NewService(querier Querier) Service {
	return &service{querier: querier}
}

// ListStocks 查询股票分页列表。
func (s *service) ListStocks(ctx context.Context, params ListParams) (ListResult, error) {
	if s.querier == nil {
		return ListResult{}, apperror.New(apperror.CodeInternal, errors.New("stock service is not configured"))
	}

	page, pageSize := sanitizePage(params.Page, params.PageSize)
	rows, err := s.querier.ListStocks(ctx, dbgen.ListStocksParams{
		Column1: params.Keyword,
		Limit:   int32(pageSize),
		Offset:  int32((page - 1) * pageSize),
	})
	if err != nil {
		return ListResult{}, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("list stocks from database: %w", err))
	}

	items := make([]Stock, 0, len(rows))
	for _, row := range rows {
		items = append(items, Stock{
			TSCode:   row.TsCode,
			Symbol:   row.Symbol,
			Name:     row.Name,
			Industry: textValue(row.Industry),
			Exchange: row.Exchange,
			IsActive: row.IsActive,
		})
	}

	return ListResult{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetStock 查询单只股票信息。
func (s *service) GetStock(ctx context.Context, tsCode string) (Stock, error) {
	if s.querier == nil {
		return Stock{}, apperror.New(apperror.CodeInternal, errors.New("stock service is not configured"))
	}

	row, err := s.querier.GetStock(ctx, tsCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Stock{}, apperror.New(apperror.CodeNotFound, fmt.Errorf("stock %s not found: %w", tsCode, err))
		}
		return Stock{}, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("get stock from database: %w", err))
	}

	return Stock{
		TSCode:   row.TsCode,
		Symbol:   row.Symbol,
		Name:     row.Name,
		Industry: textValue(row.Industry),
		Exchange: row.Exchange,
		IsActive: row.IsActive,
	}, nil
}

// ListDailyBars 查询单只股票日线数据。
func (s *service) ListDailyBars(ctx context.Context, params DailyParams) ([]DailyBar, error) {
	if s.querier == nil {
		return nil, apperror.New(apperror.CodeInternal, errors.New("stock service is not configured"))
	}

	rows, err := s.querier.ListStockDaily(ctx, dbgen.ListStockDailyParams{
		TsCode: params.TSCode,
		TradeDate: pgtype.Date{
			Time:  params.StartDate,
			Valid: true,
		},
		TradeDate_2: pgtype.Date{
			Time:  params.EndDate,
			Valid: true,
		},
	})
	if err != nil {
		return nil, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("list stock daily from database: %w", err))
	}

	items := make([]DailyBar, 0, len(rows))
	for _, row := range rows {
		items = append(items, DailyBar{
			TSCode:    row.TsCode,
			TradeDate: dateValue(row.TradeDate),
			Open:      numericValue(row.Open),
			High:      numericValue(row.High),
			Low:       numericValue(row.Low),
			Close:     numericValue(row.Close),
			PctChg:    numericValue(row.PctChg),
			Vol:       numericValue(row.Vol),
			Amount:    numericValue(row.Amount),
		})
	}

	return items, nil
}

func sanitizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize
}

func textValue(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}

	return value.String
}

func dateValue(value pgtype.Date) time.Time {
	if !value.Valid {
		return time.Time{}
	}

	return value.Time
}

func numericValue(value pgtype.Numeric) string {
	if !value.Valid || value.Int == nil {
		return ""
	}

	return decimal.NewFromBigInt(new(big.Int).Set(value.Int), value.Exp).String()
}
