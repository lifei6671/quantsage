package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/config"
	infraLog "github.com/lifei6671/quantsage/apps/server/internal/infra/log"
	"github.com/lifei6671/quantsage/apps/server/internal/infra/scheduler"
)

var localTaskCronSpecs = map[string]string{
	"sync_stock_basic":       "0 0 8 * * 1-5",
	"sync_trade_calendar":    "0 5 8 * * 1-5",
	localDailyMarketPipeline: "0 0 18 * * 1-5",
}

func main() {
	configPath := flag.String("config", config.ResolvePath("configs/config.example.yaml", "../../configs/config.example.yaml"), "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	logger := infraLog.New()
	if !strings.EqualFold(cfg.App.Env, "local") {
		logger.Info("quantsage worker bootstrap", "mode", cfg.App.Env, "jobs", []string{})
		return
	}

	// local worker 只负责定时调度；真正的 sample runtime 统一留在 server 进程里，
	// 避免 worker 与 HTTP API 各自维护一份独立内存态，导致 UI 看不到定时任务结果。
	apiRunner, err := newLocalAPIRunner(cfg.App.Addr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "bootstrap local api runner: %v\n", err)
		os.Exit(1)
	}
	runner := newLocalPipelineRunner(apiRunner)

	s := scheduler.New(logger, runner)
	taskSpecs, err := buildLocalTaskSpecs(localTaskNames())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "build scheduler task specs: %v\n", err)
		os.Exit(1)
	}
	for _, task := range taskSpecs {
		if err := s.Register(task); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "register scheduler task %s: %v\n", task.JobName, err)
			os.Exit(1)
		}
	}
	s.Start()

	logger.Info("quantsage worker bootstrap", "mode", "local", "jobs", s.TaskNames())
	waitForShutdown(logger, s)
}

func localTaskNames() []string {
	names := make([]string, 0, len(localTaskCronSpecs))
	for jobName := range localTaskCronSpecs {
		names = append(names, jobName)
	}
	sort.Strings(names)
	return names
}

func buildLocalTaskSpecs(jobNames []string) ([]scheduler.TaskSpec, error) {
	items := make([]scheduler.TaskSpec, 0, len(jobNames))
	for _, jobName := range jobNames {
		spec, ok := localTaskCronSpecs[jobName]
		if !ok {
			return nil, fmt.Errorf("job %s does not have a cron spec", jobName)
		}
		items = append(items, scheduler.TaskSpec{
			JobName: jobName,
			Spec:    spec,
		})
	}

	return items, nil
}

func waitForShutdown(logger *slog.Logger, s *scheduler.Scheduler) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Info("quantsage worker shutting down")

	stopped := s.Stop()
	select {
	case <-stopped.Done():
		logger.Info("quantsage worker stopped")
	case <-time.After(5 * time.Second):
		logger.Warn("quantsage worker stop timeout")
	}
}
