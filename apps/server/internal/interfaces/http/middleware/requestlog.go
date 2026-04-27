package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	infraLog "github.com/lifei6671/quantsage/apps/server/internal/infra/log"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/requestid"
)

// RequestLog 负责初始化请求级日志字段，并在请求完成后统一输出一条访问日志。
func RequestLog(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(requestid.HeaderName)
		if reqID == "" {
			reqID = requestid.New()
		}

		c.Set(requestid.ContextKey, reqID)
		c.Writer.Header().Set(requestid.HeaderName, reqID)
		ctx := infraLog.WithRequestInfo(c.Request.Context())
		ctx = infraLog.AddInfo(ctx,
			infraLog.String("request_id", reqID),
			infraLog.String("method", c.Request.Method),
			infraLog.String("path", c.Request.URL.Path),
		)
		c.Request = c.Request.WithContext(ctx)

		startedAt := time.Now()
		c.Next()

		ctx = infraLog.AddInfo(c.Request.Context(),
			infraLog.Int("status", c.Writer.Status()),
			infraLog.Int64("latency_ms", time.Since(startedAt).Milliseconds()),
		)
		c.Request = c.Request.WithContext(ctx)

		attrs := infraLog.Fields(c.Request.Context())
		logger.InfoContext(c.Request.Context(), "http request completed", attrsToAny(attrs)...)
	}
}

func attrsToAny(attrs []slog.Attr) []any {
	if len(attrs) == 0 {
		return nil
	}

	out := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		out = append(out, attr)
	}

	return out
}
