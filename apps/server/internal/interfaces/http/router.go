package http

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/middleware"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

// NewRouter constructs the base HTTP router for QuantSage APIs.
func NewRouter(logger *slog.Logger) *gin.Engine {
	router := gin.New()
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.RequestLog(logger))

	router.GET("/api/healthz", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok"})
	})

	return router
}
