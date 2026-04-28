package http

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	jobdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/job"
	stockdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/stock"
	strategydomain "github.com/lifei6671/quantsage/apps/server/internal/domain/strategy"
	"github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/handler"
	"github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/middleware"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

// NewRouter constructs the base HTTP router for QuantSage APIs.
func NewRouter(logger *slog.Logger) *gin.Engine {
	return newBaseRouter(logger)
}

// NewRouterWithStockService 构建带股票服务的 HTTP 路由。
func NewRouterWithStockService(logger *slog.Logger, stockService stockdomain.Service) *gin.Engine {
	return NewRouterWithDependencies(logger, stockService, nil, nil, nil)
}

// NewRouterWithServices 构建带股票服务和任务执行器的 HTTP 路由。
func NewRouterWithServices(logger *slog.Logger, stockService stockdomain.Service, jobRunner jobdomain.Runner) *gin.Engine {
	return NewRouterWithDependencies(logger, stockService, jobRunner, nil, nil)
}

// NewRouterWithDependencies 构建带股票服务、任务执行器、信号服务和任务记录服务的 HTTP 路由。
func NewRouterWithDependencies(logger *slog.Logger, stockService stockdomain.Service, jobRunner jobdomain.Runner, signalService strategydomain.QueryService, jobQueryService jobdomain.QueryService) *gin.Engine {
	router := newBaseRouter(logger)

	if stockService != nil {
		stockHandler := handler.NewStockHandler(stockService)
		router.GET("/api/stocks", stockHandler.ListStocks)
		router.GET("/api/stocks/:ts_code", stockHandler.GetStock)
		router.GET("/api/stocks/:ts_code/daily", stockHandler.ListDailyBars)
	}

	if jobQueryService != nil {
		jobHandler := handler.NewJobHandler(jobRunner, jobQueryService)
		router.GET("/api/jobs", jobHandler.List)
	}
	if jobRunner != nil {
		jobHandler := handler.NewJobHandler(jobRunner, jobQueryService)
		router.POST("/api/jobs/:job_name/run", jobHandler.Run)
	}

	if signalService != nil {
		signalHandler := handler.NewSignalHandler(signalService)
		router.GET("/api/signals", signalHandler.ListSignals)
	}

	return router
}

func newBaseRouter(logger *slog.Logger) *gin.Engine {
	router := gin.New()
	router.Use(middleware.CORS())
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.RequestLog(logger))

	router.GET("/api/healthz", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok"})
	})

	return router
}
