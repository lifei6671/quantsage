package middleware

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/requestid"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

// Recovery converts panics into structured API failures.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.ErrorContext(c.Request.Context(), "http panic recovered",
			"request_id", requestid.FromContext(c),
			"panic", fmt.Sprint(recovered),
		)
		response.Fail(c, apperror.New(apperror.CodeInternal, fmt.Errorf("recover panic: %v", recovered)))
	})
}
