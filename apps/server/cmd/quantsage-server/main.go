package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/lifei6671/quantsage/apps/server/internal/app"
	"github.com/lifei6671/quantsage/apps/server/internal/config"
	infraLog "github.com/lifei6671/quantsage/apps/server/internal/infra/log"
	httpapi "github.com/lifei6671/quantsage/apps/server/internal/interfaces/http"
)

func main() {
	configPath := flag.String("config", config.ResolvePath("configs/config.example.yaml", "../../configs/config.example.yaml"), "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	logger := infraLog.New()
	runtime, err := app.NewServerRuntime(ctx, cfg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "bootstrap server runtime: %v\n", err)
		os.Exit(1)
	}
	defer runtime.Close()

	sessionStore, err := app.NewSessionStore(cfg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "bootstrap session store: %v\n", err)
		os.Exit(1)
	}

	router := httpapi.NewRouterWithRuntime(logger, httpapi.RouterDependencies{
		StockService:     runtime.StockService(),
		JobRunner:        runtime.Runner(),
		SignalService:    runtime.SignalQueryService(),
		JobQueryService:  runtime.JobQueryService(),
		UserService:      runtime.UserService(),
		WatchlistService: runtime.WatchlistService(),
		PositionService:  runtime.PositionService(),
		SessionStore:     sessionStore,
		SessionName:      cfg.Auth.SessionName,
		AllowedOrigins:   cfg.Auth.AllowedOrigins,
	})

	if err := router.Run(cfg.App.Addr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "run server: %v\n", err)
		os.Exit(1)
	}
}
