package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	jobdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/job"
)

const localDailyMarketPipeline = "daily_market_pipeline"

var localDailyMarketPipelineSteps = []string{
	"sync_daily_market",
	"calc_daily_factor",
	"generate_strategy_signals",
}

type localPipelineRunner struct {
	base      jobdomain.Runner
	pipelines map[string][]string
}

func newLocalPipelineRunner(base jobdomain.Runner) jobdomain.Runner {
	return &localPipelineRunner{
		base: base,
		pipelines: map[string][]string{
			localDailyMarketPipeline: append([]string(nil), localDailyMarketPipelineSteps...),
		},
	}
}

func (r *localPipelineRunner) Run(ctx context.Context, jobName string, bizDate time.Time) error {
	if r.base == nil {
		return errors.New("local pipeline runner base runner is required")
	}

	steps, ok := r.pipelines[jobName]
	if !ok {
		return r.base.Run(ctx, jobName, bizDate)
	}

	for _, step := range steps {
		if err := r.base.Run(ctx, step, bizDate); err != nil {
			return fmt.Errorf("run local pipeline %s step %s: %w", jobName, step, err)
		}
	}

	return nil
}
