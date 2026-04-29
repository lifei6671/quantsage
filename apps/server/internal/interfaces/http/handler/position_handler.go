package handler

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	positiondomain "github.com/lifei6671/quantsage/apps/server/internal/domain/position"
	"github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/dto"
	httpmiddleware "github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/middleware"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

const timeLayoutRFC3339 = "2006-01-02T15:04:05Z07:00"

// PositionHandler 提供用户持仓相关 HTTP 接口。
type PositionHandler struct {
	service positiondomain.Service
}

// NewPositionHandler 创建持仓处理器。
func NewPositionHandler(service positiondomain.Service) *PositionHandler {
	return &PositionHandler{service: service}
}

// List 返回当前用户的持仓列表。
func (h *PositionHandler) List(c *gin.Context) {
	currentUser, err := h.currentUser(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	items, err := h.service.List(c.Request.Context(), currentUser.ID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	result := make([]dto.PositionItem, 0, len(items))
	for _, item := range items {
		result = append(result, dto.PositionItem{
			ID:           item.ID,
			TSCode:       item.TSCode,
			PositionDate: item.PositionDate.Format(dateLayout),
			Quantity:     item.Quantity,
			CostPrice:    item.CostPrice,
			Note:         item.Note,
			CreatedAt:    item.CreatedAt.UTC().Format(timeLayoutRFC3339),
			UpdatedAt:    item.UpdatedAt.UTC().Format(timeLayoutRFC3339),
		})
	}

	response.OK(c, result)
}

// Create 新增一条持仓记录。
func (h *PositionHandler) Create(c *gin.Context) {
	currentUser, err := h.currentUser(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	var req dto.CreatePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.New(apperror.CodeBadRequest, fmt.Errorf("bind create position request: %w", err)))
		return
	}

	item, err := h.service.Create(c.Request.Context(), currentUser.ID, positiondomain.CreateInput{
		TSCode:       req.TSCode,
		PositionDate: req.PositionDate,
		Quantity:     req.Quantity,
		CostPrice:    req.CostPrice,
		Note:         req.Note,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, dto.PositionItem{
		ID:           item.ID,
		TSCode:       item.TSCode,
		PositionDate: item.PositionDate.Format(dateLayout),
		Quantity:     item.Quantity,
		CostPrice:    item.CostPrice,
		Note:         item.Note,
		CreatedAt:    item.CreatedAt.UTC().Format(timeLayoutRFC3339),
		UpdatedAt:    item.UpdatedAt.UTC().Format(timeLayoutRFC3339),
	})
}

// Update 修改指定持仓记录。
func (h *PositionHandler) Update(c *gin.Context) {
	currentUser, err := h.currentUser(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	positionID, err := parseIDParam(c.Param("id"), "position")
	if err != nil {
		response.Fail(c, err)
		return
	}

	var req dto.UpdatePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.New(apperror.CodeBadRequest, fmt.Errorf("bind update position request: %w", err)))
		return
	}

	item, err := h.service.Update(c.Request.Context(), currentUser.ID, positionID, positiondomain.UpdateInput{
		TSCode:       req.TSCode,
		PositionDate: req.PositionDate,
		Quantity:     req.Quantity,
		CostPrice:    req.CostPrice,
		Note:         req.Note,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, dto.PositionItem{
		ID:           item.ID,
		TSCode:       item.TSCode,
		PositionDate: item.PositionDate.Format(dateLayout),
		Quantity:     item.Quantity,
		CostPrice:    item.CostPrice,
		Note:         item.Note,
		CreatedAt:    item.CreatedAt.UTC().Format(timeLayoutRFC3339),
		UpdatedAt:    item.UpdatedAt.UTC().Format(timeLayoutRFC3339),
	})
}

// Delete 删除指定持仓记录。
func (h *PositionHandler) Delete(c *gin.Context) {
	currentUser, err := h.currentUser(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	positionID, err := parseIDParam(c.Param("id"), "position")
	if err != nil {
		response.Fail(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), currentUser.ID, positionID); err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, gin.H{"deleted": true})
}

func (h *PositionHandler) currentUser(c *gin.Context) (dto.CurrentUser, error) {
	if h.service == nil {
		return dto.CurrentUser{}, apperror.New(apperror.CodeInternal, errors.New("position handler is not configured"))
	}

	currentUser, ok := httpmiddleware.CurrentUser(c.Request.Context())
	if !ok {
		return dto.CurrentUser{}, apperror.New(apperror.CodeUnauthorized, errors.New("current user is missing"))
	}

	return dto.CurrentUser{ID: currentUser.ID}, nil
}
