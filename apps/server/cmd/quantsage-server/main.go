package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lifei6671/quantsage/apps/server/internal/config"
	infraLog "github.com/lifei6671/quantsage/apps/server/internal/infra/log"
	httpapi "github.com/lifei6671/quantsage/apps/server/internal/interfaces/http"
)

func main() {
	configPath := flag.String("config", "configs/config.example.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	logger := infraLog.New()
	router := httpapi.NewRouter(logger)
	if err := router.Run(cfg.App.Addr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "run server: %v\n", err)
		os.Exit(1)
	}
}
