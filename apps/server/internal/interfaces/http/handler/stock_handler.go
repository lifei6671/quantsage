package handler

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	stockdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/stock"
	infraLog "github.com/lifei6671/quantsage/apps/server/internal/infra/log"
	"github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/dto"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

const dateLayout = "2006-01-02"

// StockHandler 提供股票相关 HTTP 接口。
type StockHandler struct {
	service stockdomain.Service
}

// NewStockHandler 创建股票接口处理器。
func NewStockHandler(service stockdomain.Service) *StockHandler {
	return &StockHandler{service: service}
}

// ListStocks 返回股票分页列表。
func (h *StockHandler) ListStocks(c *gin.Context) {
	if err := h.ensureService(); err != nil {
		response.Fail(c, err)
		return
	}

	var req dto.PageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, apperror.New(apperror.CodeBadRequest, fmt.Errorf("bind stock list query: %w", err)))
		return
	}

	result, err := h.service.ListStocks(c.Request.Context(), stockdomain.ListParams{
		Keyword:  c.Query("keyword"),
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	items := make([]dto.StockItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, dto.StockItem{
			TSCode:   item.TSCode,
			Symbol:   item.Symbol,
			Name:     item.Name,
			Industry: item.Industry,
			Exchange: item.Exchange,
			IsActive: item.IsActive,
		})
	}

	response.OK(c, dto.PageResponse[dto.StockItem]{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

// GetStock 返回单只股票详情。
func (h *StockHandler) GetStock(c *gin.Context) {
	if err := h.ensureService(); err != nil {
		response.Fail(c, err)
		return
	}

	tsCode := c.Param("ts_code")
	c.Request = c.Request.WithContext(infraLog.AddInfo(c.Request.Context(), infraLog.String("ts_code", tsCode)))

	item, err := h.service.GetStock(c.Request.Context(), tsCode)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, dto.StockItem{
		TSCode:   item.TSCode,
		Symbol:   item.Symbol,
		Name:     item.Name,
		Industry: item.Industry,
		Exchange: item.Exchange,
		IsActive: item.IsActive,
	})
}

// ListDailyBars 返回单只股票日线列表。
func (h *StockHandler) ListDailyBars(c *gin.Context) {
	if err := h.ensureService(); err != nil {
		response.Fail(c, err)
		return
	}

	tsCode := c.Param("ts_code")
	c.Request = c.Request.WithContext(infraLog.AddInfo(c.Request.Context(), infraLog.String("ts_code", tsCode)))

	startDate, endDate, err := parseDateRange(c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		response.Fail(c, err)
		return
	}

	items, err := h.service.ListDailyBars(c.Request.Context(), stockdomain.DailyParams{
		TSCode:    tsCode,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	result := make([]dto.DailyBarItem, 0, len(items))
	for _, item := range items {
		result = append(result, dto.DailyBarItem{
			TSCode:    item.TSCode,
			TradeDate: item.TradeDate.Format(dateLayout),
			Open:      item.Open,
			High:      item.High,
			Low:       item.Low,
			Close:     item.Close,
			PctChg:    item.PctChg,
			Vol:       item.Vol,
			Amount:    item.Amount,
		})
	}

	response.OK(c, result)
}

func parseDateRange(startText, endText string) (time.Time, time.Time, error) {
	if startText == "" || endText == "" {
		return time.Time{}, time.Time{}, apperror.New(apperror.CodeBadRequest, errors.New("start_date and end_date are required"))
	}

	startDate, err := time.Parse(dateLayout, startText)
	if err != nil {
		return time.Time{}, time.Time{}, apperror.New(apperror.CodeBadRequest, fmt.Errorf("parse start_date: %w", err))
	}
	endDate, err := time.Parse(dateLayout, endText)
	if err != nil {
		return time.Time{}, time.Time{}, apperror.New(apperror.CodeBadRequest, fmt.Errorf("parse end_date: %w", err))
	}
	if endDate.Before(startDate) {
		return time.Time{}, time.Time{}, apperror.New(apperror.CodeBadRequest, errors.New("end_date must be greater than or equal to start_date"))
	}

	return startDate, endDate, nil
}

func (h *StockHandler) ensureService() error {
	if h.service != nil {
		return nil
	}

	return apperror.New(apperror.CodeInternal, errors.New("stock service is not configured"))
}
