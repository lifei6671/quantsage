package http

import (
	"log/slog"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	jobdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/job"
	positiondomain "github.com/lifei6671/quantsage/apps/server/internal/domain/position"
	stockdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/stock"
	strategydomain "github.com/lifei6671/quantsage/apps/server/internal/domain/strategy"
	userdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/user"
	watchlistdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/watchlist"
	"github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/handler"
	"github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/middleware"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

// RouterDependencies 定义 QuantSage HTTP Router 需要的可选依赖。
type RouterDependencies struct {
	StockService     stockdomain.Service
	JobRunner        jobdomain.Runner
	SignalService    strategydomain.QueryService
	JobQueryService  jobdomain.QueryService
	UserService      userdomain.Service
	WatchlistService watchlistdomain.Service
	PositionService  positiondomain.Service
	SessionStore     sessions.Store
	SessionName      string
	AllowedOrigins   []string
}

// NewRouter constructs the base HTTP router for QuantSage APIs.
func NewRouter(logger *slog.Logger) *gin.Engine {
	return NewRouterWithRuntime(logger, RouterDependencies{})
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
	return NewRouterWithRuntime(logger, RouterDependencies{
		StockService:    stockService,
		JobRunner:       jobRunner,
		SignalService:   signalService,
		JobQueryService: jobQueryService,
	})
}

// NewRouterWithRuntime 构建带完整运行时依赖的 HTTP 路由。
func NewRouterWithRuntime(logger *slog.Logger, deps RouterDependencies) *gin.Engine {
	router := newBaseRouter(logger, deps.SessionStore, deps.SessionName, deps.AllowedOrigins)
	registerInternalRoutes(router, deps)

	publicGroup := router.Group("/api")
	publicGroup.GET("/healthz", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok"})
	})

	authEnabled := deps.SessionStore != nil && deps.UserService != nil
	if authEnabled {
		authHandler := handler.NewAuthHandler(deps.UserService)
		publicGroup.POST("/auth/login", authHandler.Login)

		privateGroup := publicGroup.Group("")
		privateGroup.Use(middleware.AuthRequired(deps.UserService))
		privateGroup.GET("/auth/me", authHandler.GetMe)
		privateGroup.POST("/auth/logout", authHandler.Logout)
		registerSharedRoutes(privateGroup, deps)
		registerPrivateRoutes(privateGroup, deps)
	} else {
		registerSharedRoutes(publicGroup, deps)
	}

	return router
}

func registerInternalRoutes(router *gin.Engine, deps RouterDependencies) {
	if deps.JobRunner == nil {
		return
	}

	internalGroup := router.Group("/internal")
	internalGroup.Use(middleware.LoopbackOnly())
	jobHandler := handler.NewJobHandler(deps.JobRunner, deps.JobQueryService)
	internalGroup.POST("/jobs/:job_name/run", jobHandler.Run)
}

func registerSharedRoutes(routes gin.IRoutes, deps RouterDependencies) {
	if deps.StockService != nil {
		stockHandler := handler.NewStockHandler(deps.StockService)
		routes.GET("/stocks", stockHandler.ListStocks)
		routes.GET("/stocks/:ts_code", stockHandler.GetStock)
		routes.GET("/stocks/:ts_code/daily", stockHandler.ListDailyBars)
	}

	if deps.JobQueryService != nil {
		jobHandler := handler.NewJobHandler(deps.JobRunner, deps.JobQueryService)
		routes.GET("/jobs", jobHandler.List)
	}
	if deps.JobRunner != nil {
		jobHandler := handler.NewJobHandler(deps.JobRunner, deps.JobQueryService)
		routes.POST("/jobs/:job_name/run", jobHandler.Run)
	}

	if deps.SignalService != nil {
		signalHandler := handler.NewSignalHandler(deps.SignalService)
		routes.GET("/signals", signalHandler.ListSignals)
	}
}

func registerPrivateRoutes(routes gin.IRoutes, deps RouterDependencies) {
	if deps.WatchlistService != nil {
		watchlistHandler := handler.NewWatchlistHandler(deps.WatchlistService)
		routes.GET("/watchlists", watchlistHandler.ListGroups)
		routes.POST("/watchlists", watchlistHandler.CreateGroup)
		routes.PUT("/watchlists/:id", watchlistHandler.UpdateGroup)
		routes.DELETE("/watchlists/:id", watchlistHandler.DeleteGroup)
		routes.GET("/watchlists/:id/items", watchlistHandler.ListItems)
		routes.POST("/watchlists/:id/items", watchlistHandler.CreateItem)
		routes.DELETE("/watchlists/:id/items/:item_id", watchlistHandler.DeleteItem)
	}

	if deps.PositionService != nil {
		positionHandler := handler.NewPositionHandler(deps.PositionService)
		routes.GET("/positions", positionHandler.List)
		routes.POST("/positions", positionHandler.Create)
		routes.PUT("/positions/:id", positionHandler.Update)
		routes.DELETE("/positions/:id", positionHandler.Delete)
	}
}

func newBaseRouter(logger *slog.Logger, sessionStore sessions.Store, sessionName string, allowedOrigins []string) *gin.Engine {
	router := gin.New()
	router.Use(middleware.CORS(allowedOrigins))
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.RequestLog(logger))
	if sessionStore != nil {
		router.Use(sessions.Sessions(sessionName, sessionStore))
	}

	return router
}
