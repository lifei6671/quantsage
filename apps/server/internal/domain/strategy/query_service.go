package strategy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

const (
	defaultSignalPage     = 1
	defaultSignalPageSize = 20
	maxSignalPageSize     = 100
)

// QueryParams 定义信号查询条件。
type QueryParams struct {
	TradeDate    time.Time
	StrategyCode string
	Page         int
	PageSize     int
}

// QueryResult 定义信号分页结果。
type QueryResult struct {
	Items    []SignalResult
	Page     int
	PageSize int
}

// QueryService 定义信号查询服务契约。
type QueryService interface {
	ListSignals(ctx context.Context, params QueryParams) (QueryResult, error)
}

// SignalReader 定义信号数据读取接口。
type SignalReader interface {
	ListSignals(ctx context.Context, params QueryParams) ([]SignalResult, error)
}

type queryService struct {
	reader SignalReader
}

// NewQueryService 创建信号查询服务。
func NewQueryService(reader SignalReader) QueryService {
	return &queryService{reader: reader}
}

// ListSignals 查询策略信号。
func (s *queryService) ListSignals(ctx context.Context, params QueryParams) (QueryResult, error) {
	if s.reader == nil {
		return QueryResult{}, apperror.New(apperror.CodeInternal, errors.New("signal query service is not configured"))
	}

	page, pageSize := sanitizeSignalPage(params.Page, params.PageSize)
	items, err := s.reader.ListSignals(ctx, QueryParams{
		TradeDate:    params.TradeDate,
		StrategyCode: params.StrategyCode,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		return QueryResult{}, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("list strategy signals: %w", err))
	}

	return QueryResult{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func sanitizeSignalPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultSignalPage
	}
	if pageSize <= 0 {
		pageSize = defaultSignalPageSize
	}
	if pageSize > maxSignalPageSize {
		pageSize = maxSignalPageSize
	}

	return page, pageSize
}
