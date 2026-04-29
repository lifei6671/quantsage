package middleware

import (
	"errors"
	"fmt"
	"net"

	"github.com/gin-gonic/gin"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

// LoopbackOnly 仅允许本机来源访问内部接口，给 local worker 调用 server 内存态任务使用。
func LoopbackOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			response.Fail(c, apperror.New(apperror.CodeForbidden, fmt.Errorf("parse remote addr: %w", err)))
			c.Abort()
			return
		}
		remoteIP := net.ParseIP(host)
		if remoteIP == nil || !remoteIP.IsLoopback() {
			response.Fail(c, apperror.New(apperror.CodeForbidden, errors.New("internal endpoint requires loopback client")))
			c.Abort()
			return
		}

		c.Next()
	}
}
