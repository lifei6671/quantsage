package handler

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	userdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/user"
	"github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/dto"
	httpmiddleware "github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/middleware"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

// AuthHandler 提供登录、登出和当前用户接口。
type AuthHandler struct {
	userService userdomain.Service
}

// NewAuthHandler 创建认证接口处理器。
func NewAuthHandler(userService userdomain.Service) *AuthHandler {
	return &AuthHandler{userService: userService}
}

// Login 校验用户名和密码，并写入 Redis Session。
func (h *AuthHandler) Login(c *gin.Context) {
	if err := h.ensureService(); err != nil {
		response.Fail(c, err)
		return
	}

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.New(apperror.CodeBadRequest, fmt.Errorf("bind login request: %w", err)))
		return
	}

	currentUser, err := h.userService.Authenticate(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := httpmiddleware.SetSessionUserID(c, currentUser.ID); err != nil {
		response.Fail(c, apperror.New(apperror.CodeInternal, fmt.Errorf("set session user id: %w", err)))
		return
	}

	response.OK(c, buildCurrentUserDTO(currentUser))
}

// Logout 清理当前登录会话。
func (h *AuthHandler) Logout(c *gin.Context) {
	if err := httpmiddleware.ClearSession(c); err != nil {
		response.Fail(c, apperror.New(apperror.CodeInternal, fmt.Errorf("clear session: %w", err)))
		return
	}

	response.OK(c, gin.H{"status": "ok"})
}

// GetMe 返回当前会话用户信息。
func (h *AuthHandler) GetMe(c *gin.Context) {
	currentUser, ok := httpmiddleware.CurrentUser(c.Request.Context())
	if !ok {
		response.Fail(c, apperror.New(apperror.CodeUnauthorized, errors.New("current user is missing")))
		return
	}

	response.OK(c, buildCurrentUserDTO(currentUser))
}

func (h *AuthHandler) ensureService() error {
	if h.userService != nil {
		return nil
	}

	return apperror.New(apperror.CodeInternal, errors.New("auth handler is not configured"))
}

func buildCurrentUserDTO(currentUser userdomain.User) dto.CurrentUser {
	item := dto.CurrentUser{
		ID:          currentUser.ID,
		Username:    currentUser.Username,
		DisplayName: currentUser.DisplayName,
		Status:      currentUser.Status,
		Role:        currentUser.Role,
	}
	if currentUser.LastLoginAt != nil && !currentUser.LastLoginAt.IsZero() {
		item.LastLoginAt = currentUser.LastLoginAt.UTC().Format(time.RFC3339)
	}

	return item
}
