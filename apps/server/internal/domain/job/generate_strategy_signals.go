package job

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/strategy"
)

// StrategySignalReader 读取待评估的市场上下文。
type StrategySignalReader interface {
	ListStrategyContexts(ctx context.Context, startDate, endDate time.Time) ([]strategy.MarketContext, error)
}

// StrategySignalWriter 持久化策略信号结果。
type StrategySignalWriter interface {
	UpsertStrategySignals(ctx context.Context, items []strategy.SignalResult) error
}

// StrategySignalReplaceWriter 支持按日期区间重算覆盖信号结果。
type StrategySignalReplaceWriter interface {
	ReplaceStrategySignals(ctx context.Context, startDate, endDate time.Time, items []strategy.SignalResult) error
}

// GenerateStrategySignals 生成指定区间内的固定策略信号。
func GenerateStrategySignals(ctx context.Context, recorder JobRunRecorder, reader StrategySignalReader, writer StrategySignalWriter, startDate, endDate, bizDate time.Time) error {
	const jobName = "generate_strategy_signals"
	if err := recorder.Start(ctx, jobName, bizDate); err != nil {
		return fmt.Errorf("start strategy signal job: %w", err)
	}

	contexts, err := reader.ListStrategyContexts(ctx, startDate, endDate)
	if err != nil {
		if failErr := recorder.Fail(ctx, jobName, bizDate, err); failErr != nil {
			return fmt.Errorf("record strategy signal read failure: %w", failErr)
		}
		return fmt.Errorf("list strategy contexts: %w", err)
	}

	signals := make([]strategy.SignalResult, 0, len(contexts))
	sort.Slice(contexts, func(i, j int) bool {
		if contexts[i].CurrentBar.TSCode == contexts[j].CurrentBar.TSCode {
			return contexts[i].CurrentBar.TradeDate.Before(contexts[j].CurrentBar.TradeDate)
		}
		return contexts[i].CurrentBar.TSCode < contexts[j].CurrentBar.TSCode
	})

	for _, item := range contexts {
		result, hit, evalErr := strategy.EvaluateVolumeBreakout(item)
		if evalErr != nil {
			if failErr := recorder.Fail(ctx, jobName, bizDate, evalErr); failErr != nil {
				return fmt.Errorf("record volume breakout failure: %w", failErr)
			}
			return fmt.Errorf("evaluate volume breakout: %w", evalErr)
		}
		if hit && result != nil {
			signals = append(signals, *result)
		}

		result, hit, evalErr = strategy.EvaluateTrendBreak(item)
		if evalErr != nil {
			if failErr := recorder.Fail(ctx, jobName, bizDate, evalErr); failErr != nil {
				return fmt.Errorf("record trend break failure: %w", failErr)
			}
			return fmt.Errorf("evaluate trend break: %w", evalErr)
		}
		if hit && result != nil {
			signals = append(signals, *result)
		}
	}

	var writeErr error
	if replacer, ok := writer.(StrategySignalReplaceWriter); ok {
		writeErr = replacer.ReplaceStrategySignals(ctx, startDate, endDate, signals)
	} else {
		writeErr = writer.UpsertStrategySignals(ctx, signals)
	}
	if writeErr != nil {
		if failErr := recorder.Fail(ctx, jobName, bizDate, writeErr); failErr != nil {
			return fmt.Errorf("record strategy signal write failure: %w", failErr)
		}
		return fmt.Errorf("write strategy signals: %w", writeErr)
	}

	if err := recorder.Success(ctx, jobName, bizDate); err != nil {
		return fmt.Errorf("mark strategy signal job success: %w", err)
	}

	return nil
}
