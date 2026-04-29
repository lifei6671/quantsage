package middleware

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	userdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/user"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

const sessionUserIDKey = "user_id"

type currentUserContextKey struct{}

// AuthRequired 从 session 恢复当前用户，并将其写入请求上下文。
func AuthRequired(userService userdomain.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if userService == nil {
			response.Fail(c, apperror.New(apperror.CodeInternal, errors.New("auth middleware is not configured")))
			c.Abort()
			return
		}

		userID, err := sessionUserID(c)
		if err != nil {
			response.Fail(c, apperror.New(apperror.CodeUnauthorized, fmt.Errorf("restore session user: %w", err)))
			c.Abort()
			return
		}

		currentUser, err := userService.GetByID(c.Request.Context(), userID)
		if err != nil {
			if apperror.CodeOf(err) == apperror.CodeUnauthorized {
				// 当前会话对应的账号已经失效时，必须把清理结果持久化回 store，避免浏览器反复携带脏 session。
				if clearErr := ClearSession(c); clearErr != nil {
					response.Fail(c, apperror.New(apperror.CodeInternal, fmt.Errorf("clear invalid session: %w", clearErr)))
					c.Abort()
					return
				}
			}
			response.Fail(c, err)
			c.Abort()
			return
		}

		ctx := context.WithValue(c.Request.Context(), currentUserContextKey{}, currentUser)
		c.Request = c.Request.WithContext(ctx)
		c.Set("current_user", currentUser)
		c.Next()
	}
}

// CurrentUser 从请求上下文中读取当前已登录用户。
func CurrentUser(ctx context.Context) (userdomain.User, bool) {
	value := ctx.Value(currentUserContextKey{})
	user, ok := value.(userdomain.User)
	return user, ok
}

// SetSessionUserID 在登录成功后写入用户会话。
func SetSessionUserID(c *gin.Context, userID int64) error {
	session := sessions.Default(c)
	session.Set(sessionUserIDKey, userID)
	if err := session.Save(); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	return nil
}

// ClearSession 清理当前登录会话。
func ClearSession(c *gin.Context) error {
	clearSession(c)
	session := sessions.Default(c)
	if err := session.Save(); err != nil {
		return fmt.Errorf("save cleared session: %w", err)
	}

	return nil
}

func clearSession(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
}

func sessionUserID(c *gin.Context) (int64, error) {
	value := sessions.Default(c).Get(sessionUserIDKey)
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case float64:
		return int64(typed), nil
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse session user id: %w", err)
		}
		return parsed, nil
	case nil:
		return 0, errors.New("session user id is missing")
	default:
		return 0, fmt.Errorf("unsupported session user id type %T", value)
	}
}
