package handler

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	watchlistdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/watchlist"
	"github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/dto"
	httpmiddleware "github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/middleware"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

// WatchlistHandler 提供用户自选股相关 HTTP 接口。
type WatchlistHandler struct {
	service watchlistdomain.Service
}

// NewWatchlistHandler 创建自选股处理器。
func NewWatchlistHandler(service watchlistdomain.Service) *WatchlistHandler {
	return &WatchlistHandler{service: service}
}

// ListGroups 返回当前用户的自选分组列表。
func (h *WatchlistHandler) ListGroups(c *gin.Context) {
	currentUser, err := h.currentUser(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	items, err := h.service.ListGroups(c.Request.Context(), currentUser.ID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	result := make([]dto.WatchlistGroupItem, 0, len(items))
	for _, item := range items {
		result = append(result, dto.WatchlistGroupItem{
			ID:        item.ID,
			Name:      item.Name,
			SortOrder: item.SortOrder,
			CreatedAt: item.CreatedAt.UTC().Format(timeLayoutRFC3339),
			UpdatedAt: item.UpdatedAt.UTC().Format(timeLayoutRFC3339),
		})
	}
	response.OK(c, result)
}

// CreateGroup 新增自选分组。
func (h *WatchlistHandler) CreateGroup(c *gin.Context) {
	currentUser, err := h.currentUser(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	var req dto.CreateWatchlistGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.New(apperror.CodeBadRequest, fmt.Errorf("bind create watchlist group request: %w", err)))
		return
	}

	item, err := h.service.CreateGroup(c.Request.Context(), currentUser.ID, watchlistdomain.CreateGroupInput{
		Name:      req.Name,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, dto.WatchlistGroupItem{
		ID:        item.ID,
		Name:      item.Name,
		SortOrder: item.SortOrder,
		CreatedAt: item.CreatedAt.UTC().Format(timeLayoutRFC3339),
		UpdatedAt: item.UpdatedAt.UTC().Format(timeLayoutRFC3339),
	})
}

// UpdateGroup 更新指定分组。
func (h *WatchlistHandler) UpdateGroup(c *gin.Context) {
	currentUser, err := h.currentUser(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	groupID, err := parseIDParam(c.Param("id"), "watchlist group")
	if err != nil {
		response.Fail(c, err)
		return
	}

	var req dto.UpdateWatchlistGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.New(apperror.CodeBadRequest, fmt.Errorf("bind update watchlist group request: %w", err)))
		return
	}

	item, err := h.service.UpdateGroup(c.Request.Context(), currentUser.ID, groupID, watchlistdomain.UpdateGroupInput{
		Name:      req.Name,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, dto.WatchlistGroupItem{
		ID:        item.ID,
		Name:      item.Name,
		SortOrder: item.SortOrder,
		CreatedAt: item.CreatedAt.UTC().Format(timeLayoutRFC3339),
		UpdatedAt: item.UpdatedAt.UTC().Format(timeLayoutRFC3339),
	})
}

// DeleteGroup 删除指定分组。
func (h *WatchlistHandler) DeleteGroup(c *gin.Context) {
	currentUser, err := h.currentUser(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	groupID, err := parseIDParam(c.Param("id"), "watchlist group")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.DeleteGroup(c.Request.Context(), currentUser.ID, groupID); err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, gin.H{"deleted": true})
}

// ListItems 返回指定分组下的自选股列表。
func (h *WatchlistHandler) ListItems(c *gin.Context) {
	currentUser, err := h.currentUser(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	groupID, err := parseIDParam(c.Param("id"), "watchlist group")
	if err != nil {
		response.Fail(c, err)
		return
	}

	items, err := h.service.ListItems(c.Request.Context(), currentUser.ID, groupID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	result := make([]dto.WatchlistItem, 0, len(items))
	for _, item := range items {
		result = append(result, dto.WatchlistItem{
			ID:        item.ID,
			GroupID:   item.GroupID,
			TSCode:    item.TSCode,
			Note:      item.Note,
			CreatedAt: item.CreatedAt.UTC().Format(timeLayoutRFC3339),
		})
	}

	response.OK(c, result)
}

// CreateItem 在分组内新增一只股票。
func (h *WatchlistHandler) CreateItem(c *gin.Context) {
	currentUser, err := h.currentUser(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	groupID, err := parseIDParam(c.Param("id"), "watchlist group")
	if err != nil {
		response.Fail(c, err)
		return
	}

	var req dto.CreateWatchlistItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.New(apperror.CodeBadRequest, fmt.Errorf("bind create watchlist item request: %w", err)))
		return
	}

	item, err := h.service.CreateItem(c.Request.Context(), currentUser.ID, groupID, watchlistdomain.CreateItemInput{
		TSCode: req.TSCode,
		Note:   req.Note,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, dto.WatchlistItem{
		ID:        item.ID,
		GroupID:   item.GroupID,
		TSCode:    item.TSCode,
		Note:      item.Note,
		CreatedAt: item.CreatedAt.UTC().Format(timeLayoutRFC3339),
	})
}

// DeleteItem 删除分组内一条股票记录。
func (h *WatchlistHandler) DeleteItem(c *gin.Context) {
	currentUser, err := h.currentUser(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	groupID, err := parseIDParam(c.Param("id"), "watchlist group")
	if err != nil {
		response.Fail(c, err)
		return
	}
	itemID, err := parseIDParam(c.Param("item_id"), "watchlist item")
	if err != nil {
		response.Fail(c, err)
		return
	}

	if err := h.service.DeleteItem(c.Request.Context(), currentUser.ID, groupID, itemID); err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, gin.H{"deleted": true})
}

func (h *WatchlistHandler) currentUser(c *gin.Context) (dto.CurrentUser, error) {
	if h.service == nil {
		return dto.CurrentUser{}, apperror.New(apperror.CodeInternal, errors.New("watchlist handler is not configured"))
	}

	currentUser, ok := httpmiddleware.CurrentUser(c.Request.Context())
	if !ok {
		return dto.CurrentUser{}, apperror.New(apperror.CodeUnauthorized, errors.New("current user is missing"))
	}

	return dto.CurrentUser{ID: currentUser.ID}, nil
}

func parseIDParam(value string, target string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, apperror.New(apperror.CodeBadRequest, fmt.Errorf("invalid %s id", target))
	}

	return parsed, nil
}
