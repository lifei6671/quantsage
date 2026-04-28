package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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

	logger := infraLog.New()
	router := httpapi.NewRouter(logger)
	if strings.EqualFold(cfg.App.Env, "local") {
		runtime, err := app.NewSampleRuntime("apps/server/testdata/sample")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "bootstrap sample runtime: %v\n", err)
			os.Exit(1)
		}

		router = httpapi.NewRouterWithDependencies(
			logger,
			runtime.StockService(),
			runtime.Runner(),
			runtime.SignalQueryService(),
			runtime.JobQueryService(),
		)
	}

	if err := router.Run(cfg.App.Addr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "run server: %v\n", err)
		os.Exit(1)
	}
}
