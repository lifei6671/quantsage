package handler

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	strategydomain "github.com/lifei6671/quantsage/apps/server/internal/domain/strategy"
	infraLog "github.com/lifei6671/quantsage/apps/server/internal/infra/log"
	"github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/dto"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

// SignalHandler 提供信号相关 HTTP 接口。
type SignalHandler struct {
	service strategydomain.QueryService
}

// NewSignalHandler 创建信号接口处理器。
func NewSignalHandler(service strategydomain.QueryService) *SignalHandler {
	return &SignalHandler{service: service}
}

// ListSignals 返回策略信号分页列表。
func (h *SignalHandler) ListSignals(c *gin.Context) {
	if err := h.ensureService(); err != nil {
		response.Fail(c, err)
		return
	}

	var req dto.PageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, apperror.New(apperror.CodeBadRequest, fmt.Errorf("bind signal list query: %w", err)))
		return
	}

	tradeDateText := c.Query("trade_date")
	if tradeDateText == "" {
		response.Fail(c, apperror.New(apperror.CodeBadRequest, errors.New("trade_date is required")))
		return
	}
	tradeDate, err := time.Parse(dateLayout, tradeDateText)
	if err != nil {
		response.Fail(c, apperror.New(apperror.CodeBadRequest, fmt.Errorf("parse trade_date: %w", err)))
		return
	}

	strategyCode := c.Query("strategy_code")
	if strategyCode != "" {
		c.Request = c.Request.WithContext(infraLog.AddInfo(c.Request.Context(), infraLog.String("strategy_code", strategyCode)))
	}

	result, err := h.service.ListSignals(c.Request.Context(), strategydomain.QueryParams{
		TradeDate:    tradeDate,
		StrategyCode: strategyCode,
		Page:         req.Page,
		PageSize:     req.PageSize,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	items := make([]dto.SignalItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, dto.SignalItem{
			StrategyCode:          item.StrategyCode,
			StrategyVersion:       item.StrategyVersion,
			TSCode:                item.TSCode,
			TradeDate:             item.TradeDate.Format(dateLayout),
			SignalType:            item.SignalType,
			SignalStrength:        item.SignalStrength.StringFixedBank(4),
			SignalLevel:           item.SignalLevel,
			BuyPriceRef:           item.BuyPriceRef.StringFixedBank(4),
			StopLossRef:           item.StopLossRef.StringFixedBank(4),
			TakeProfitRef:         item.TakeProfitRef.StringFixedBank(4),
			InvalidationCondition: item.InvalidationCondition,
			Reason:                item.Reason,
		})
	}

	response.OK(c, dto.PageResponse[dto.SignalItem]{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func (h *SignalHandler) ensureService() error {
	if h.service != nil {
		return nil
	}

	return apperror.New(apperror.CodeInternal, errors.New("signal query service is not configured"))
}
