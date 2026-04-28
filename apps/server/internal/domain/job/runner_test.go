package job

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

func TestRegistryRunSuccess(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	var called bool
	if err := registry.Register("sync_stock_basic", func(ctx context.Context, bizDate time.Time) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if err := registry.Run(context.Background(), "sync_stock_basic", time.Now()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !called {
		t.Fatal("job handler was not called")
	}
}

func TestRegistryRunNotFound(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	err := registry.Run(context.Background(), "missing_job", time.Now())
	if code := apperror.CodeOf(err); code != apperror.CodeNotFound {
		t.Fatalf("CodeOf(err) = %d, want %d", code, apperror.CodeNotFound)
	}
}

func TestRegistryRegisterDuplicate(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if err := registry.Register("sync_stock_basic", func(ctx context.Context, bizDate time.Time) error { return nil }); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	err := registry.Register("sync_stock_basic", func(ctx context.Context, bizDate time.Time) error { return nil })
	if code := apperror.CodeOf(err); code != apperror.CodeBadRequest {
		t.Fatalf("CodeOf(err) = %d, want %d", code, apperror.CodeBadRequest)
	}
}

func TestRegistryRunPropagatesError(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if err := registry.Register("sync_stock_basic", func(ctx context.Context, bizDate time.Time) error {
		return errors.New("boom")
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	err := registry.Run(context.Background(), "sync_stock_basic", time.Now())
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
}

func TestRegistryRunSerializesSameJob(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	finished := make(chan struct{}, 2)
	if err := registry.Register("sync_stock_basic", func(ctx context.Context, bizDate time.Time) error {
		started <- struct{}{}
		<-release
		finished <- struct{}{}
		return nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = registry.Run(context.Background(), "sync_stock_basic", time.Now())
	}()
	<-started
	go func() {
		defer wg.Done()
		_ = registry.Run(context.Background(), "sync_stock_basic", time.Now())
	}()

	select {
	case <-started:
		t.Fatal("second run started before first run finished")
	case <-time.After(50 * time.Millisecond):
	}

	release <- struct{}{}
	<-finished

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("second run did not start after first run finished")
	}

	release <- struct{}{}
	<-finished
	wg.Wait()
}

func TestRegistryRunRejectsRunningDependencyForSameBizDate(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	if err := registry.Register("sync_daily_market", func(ctx context.Context, bizDate time.Time) error {
		started <- struct{}{}
		<-release
		return nil
	}); err != nil {
		t.Fatalf("Register(sync_daily_market) error = %v", err)
	}
	if err := registry.Register(
		"calc_daily_factor",
		func(ctx context.Context, bizDate time.Time) error { return nil },
		WithDependencies("sync_daily_market"),
	); err != nil {
		t.Fatalf("Register(calc_daily_factor) error = %v", err)
	}

	bizDate := time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC)
	go func() {
		_ = registry.Run(context.Background(), "sync_daily_market", bizDate)
	}()
	<-started

	err := registry.Run(context.Background(), "calc_daily_factor", bizDate)
	if code := apperror.CodeOf(err); code != apperror.CodeJobRunning {
		t.Fatalf("CodeOf(err) = %d, want %d", code, apperror.CodeJobRunning)
	}

	close(release)
}

func TestRegistryRunRejectsMissingDependencyCompletion(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if err := registry.Register("sync_daily_market", func(ctx context.Context, bizDate time.Time) error { return nil }); err != nil {
		t.Fatalf("Register(sync_daily_market) error = %v", err)
	}
	if err := registry.Register(
		"calc_daily_factor",
		func(ctx context.Context, bizDate time.Time) error { return nil },
		WithDependencies("sync_daily_market"),
	); err != nil {
		t.Fatalf("Register(calc_daily_factor) error = %v", err)
	}

	err := registry.Run(context.Background(), "calc_daily_factor", time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC))
	if code := apperror.CodeOf(err); code != apperror.CodeValidationFailed {
		t.Fatalf("CodeOf(err) = %d, want %d", code, apperror.CodeValidationFailed)
	}
}

func TestRegistryRunAllowsDependencyAfterSuccessfulCompletion(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if err := registry.Register("sync_daily_market", func(ctx context.Context, bizDate time.Time) error { return nil }); err != nil {
		t.Fatalf("Register(sync_daily_market) error = %v", err)
	}

	downstreamCalled := false
	if err := registry.Register(
		"calc_daily_factor",
		func(ctx context.Context, bizDate time.Time) error {
			downstreamCalled = true
			return nil
		},
		WithDependencies("sync_daily_market"),
	); err != nil {
		t.Fatalf("Register(calc_daily_factor) error = %v", err)
	}

	bizDate := time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC)
	if err := registry.Run(context.Background(), "sync_daily_market", bizDate); err != nil {
		t.Fatalf("Run(sync_daily_market) error = %v", err)
	}
	if err := registry.Run(context.Background(), "calc_daily_factor", bizDate); err != nil {
		t.Fatalf("Run(calc_daily_factor) error = %v", err)
	}
	if !downstreamCalled {
		t.Fatal("downstream job was not called")
	}
}

func TestRegistryRunPreservesCompletionAfterFailedRerun(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	runCount := 0
	if err := registry.Register("sync_daily_market", func(ctx context.Context, bizDate time.Time) error {
		runCount++
		if runCount == 2 {
			return errors.New("boom")
		}
		return nil
	}); err != nil {
		t.Fatalf("Register(sync_daily_market) error = %v", err)
	}

	downstreamCalled := false
	if err := registry.Register(
		"calc_daily_factor",
		func(ctx context.Context, bizDate time.Time) error {
			downstreamCalled = true
			return nil
		},
		WithDependencies("sync_daily_market"),
	); err != nil {
		t.Fatalf("Register(calc_daily_factor) error = %v", err)
	}

	bizDate := time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC)
	if err := registry.Run(context.Background(), "sync_daily_market", bizDate); err != nil {
		t.Fatalf("first Run(sync_daily_market) error = %v", err)
	}
	if err := registry.Run(context.Background(), "sync_daily_market", bizDate); err == nil {
		t.Fatal("second Run(sync_daily_market) error = nil, want non-nil")
	}
	if err := registry.Run(context.Background(), "calc_daily_factor", bizDate); err != nil {
		t.Fatalf("Run(calc_daily_factor) error = %v", err)
	}
	if !downstreamCalled {
		t.Fatal("downstream job was not called")
	}
}

func TestRegistryRunInvalidatesDependentCompletionsAfterSuccessfulRerun(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if err := registry.Register("sync_daily_market", func(ctx context.Context, bizDate time.Time) error { return nil }); err != nil {
		t.Fatalf("Register(sync_daily_market) error = %v", err)
	}
	if err := registry.Register(
		"calc_daily_factor",
		func(ctx context.Context, bizDate time.Time) error { return nil },
		WithDependencies("sync_daily_market"),
	); err != nil {
		t.Fatalf("Register(calc_daily_factor) error = %v", err)
	}
	if err := registry.Register(
		"generate_strategy_signals",
		func(ctx context.Context, bizDate time.Time) error { return nil },
		WithDependencies("calc_daily_factor"),
	); err != nil {
		t.Fatalf("Register(generate_strategy_signals) error = %v", err)
	}

	earlierDate := time.Date(2026, 4, 20, 18, 0, 0, 0, time.UTC)
	laterDate := time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC)
	if err := registry.Run(context.Background(), "sync_daily_market", earlierDate); err != nil {
		t.Fatalf("first Run(sync_daily_market earlierDate) error = %v", err)
	}
	if err := registry.Run(context.Background(), "sync_daily_market", laterDate); err != nil {
		t.Fatalf("Run(sync_daily_market laterDate) error = %v", err)
	}
	if err := registry.Run(context.Background(), "calc_daily_factor", laterDate); err != nil {
		t.Fatalf("Run(calc_daily_factor) error = %v", err)
	}
	if err := registry.Run(context.Background(), "generate_strategy_signals", laterDate); err != nil {
		t.Fatalf("first Run(generate_strategy_signals) error = %v", err)
	}
	if err := registry.Run(context.Background(), "sync_daily_market", earlierDate); err != nil {
		t.Fatalf("second Run(sync_daily_market earlierDate) error = %v", err)
	}

	err := registry.Run(context.Background(), "generate_strategy_signals", laterDate)
	if code := apperror.CodeOf(err); code != apperror.CodeValidationFailed {
		t.Fatalf("CodeOf(err) = %d, want %d", code, apperror.CodeValidationFailed)
	}
	if err := registry.Run(context.Background(), "calc_daily_factor", laterDate); err != nil {
		t.Fatalf("Run(calc_daily_factor after invalidation) error = %v", err)
	}
}

func TestRegistryRunRejectsEarlierPrerequisiteRerunWhileLaterDependentRunning(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if err := registry.Register("sync_daily_market", func(ctx context.Context, bizDate time.Time) error { return nil }); err != nil {
		t.Fatalf("Register(sync_daily_market) error = %v", err)
	}
	if err := registry.Register(
		"calc_daily_factor",
		func(ctx context.Context, bizDate time.Time) error { return nil },
		WithDependencies("sync_daily_market"),
	); err != nil {
		t.Fatalf("Register(calc_daily_factor) error = %v", err)
	}

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	if err := registry.Register(
		"generate_strategy_signals",
		func(ctx context.Context, bizDate time.Time) error {
			started <- struct{}{}
			<-release
			return nil
		},
		WithDependencies("calc_daily_factor"),
	); err != nil {
		t.Fatalf("Register(generate_strategy_signals) error = %v", err)
	}

	earlierDate := time.Date(2026, 4, 20, 18, 0, 0, 0, time.UTC)
	laterDate := time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC)
	if err := registry.Run(context.Background(), "sync_daily_market", laterDate); err != nil {
		t.Fatalf("Run(sync_daily_market laterDate) error = %v", err)
	}
	if err := registry.Run(context.Background(), "calc_daily_factor", laterDate); err != nil {
		t.Fatalf("Run(calc_daily_factor) error = %v", err)
	}
	go func() {
		_ = registry.Run(context.Background(), "generate_strategy_signals", laterDate)
	}()
	<-started

	err := registry.Run(context.Background(), "sync_daily_market", earlierDate)
	if code := apperror.CodeOf(err); code != apperror.CodeJobRunning {
		t.Fatalf("CodeOf(err) = %d, want %d", code, apperror.CodeJobRunning)
	}

	close(release)
}

func TestRegistryRunRejectsLaterDependentWhileEarlierPrerequisiteRunning(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	if err := registry.Register("sync_daily_market", func(ctx context.Context, bizDate time.Time) error {
		started <- struct{}{}
		<-release
		return nil
	}); err != nil {
		t.Fatalf("Register(sync_daily_market) error = %v", err)
	}
	if err := registry.Register(
		"calc_daily_factor",
		func(ctx context.Context, bizDate time.Time) error { return nil },
		WithDependencies("sync_daily_market"),
	); err != nil {
		t.Fatalf("Register(calc_daily_factor) error = %v", err)
	}

	earlierDate := time.Date(2026, 4, 20, 18, 0, 0, 0, time.UTC)
	laterDate := time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC)
	if err := registry.MarkCompleted("sync_daily_market", laterDate); err != nil {
		t.Fatalf("MarkCompleted(sync_daily_market) error = %v", err)
	}
	go func() {
		_ = registry.Run(context.Background(), "sync_daily_market", earlierDate)
	}()
	<-started

	err := registry.Run(context.Background(), "calc_daily_factor", laterDate)
	if code := apperror.CodeOf(err); code != apperror.CodeJobRunning {
		t.Fatalf("CodeOf(err) = %d, want %d", code, apperror.CodeJobRunning)
	}

	close(release)
}

func TestRegistryRunRejectsPrerequisiteRerunWhileDependentRunning(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if err := registry.Register("sync_daily_market", func(ctx context.Context, bizDate time.Time) error { return nil }); err != nil {
		t.Fatalf("Register(sync_daily_market) error = %v", err)
	}
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	if err := registry.Register(
		"calc_daily_factor",
		func(ctx context.Context, bizDate time.Time) error {
			started <- struct{}{}
			<-release
			return nil
		},
		WithDependencies("sync_daily_market"),
	); err != nil {
		t.Fatalf("Register(calc_daily_factor) error = %v", err)
	}

	bizDate := time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC)
	if err := registry.Run(context.Background(), "sync_daily_market", bizDate); err != nil {
		t.Fatalf("Run(sync_daily_market) error = %v", err)
	}
	go func() {
		_ = registry.Run(context.Background(), "calc_daily_factor", bizDate)
	}()
	<-started

	err := registry.Run(context.Background(), "sync_daily_market", bizDate)
	if code := apperror.CodeOf(err); code != apperror.CodeJobRunning {
		t.Fatalf("CodeOf(err) = %d, want %d", code, apperror.CodeJobRunning)
	}

	close(release)
}

func TestRegistryRunAllowsLaterDependencyForEarlierBizDate(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	if err := registry.Register("sync_daily_market", func(ctx context.Context, bizDate time.Time) error {
		started <- struct{}{}
		<-release
		return nil
	}); err != nil {
		t.Fatalf("Register(sync_daily_market) error = %v", err)
	}

	downstreamCalled := false
	if err := registry.Register(
		"calc_daily_factor",
		func(ctx context.Context, bizDate time.Time) error {
			downstreamCalled = true
			return nil
		},
		WithDependencies("sync_daily_market"),
	); err != nil {
		t.Fatalf("Register(calc_daily_factor) error = %v", err)
	}

	completedDate := time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC)
	registry.done["sync_daily_market"] = map[string]struct{}{
		runnerDateKey(completedDate): {},
	}

	go func() {
		_ = registry.Run(context.Background(), "sync_daily_market", time.Date(2026, 4, 29, 18, 0, 0, 0, time.UTC))
	}()
	<-started

	err := registry.Run(context.Background(), "calc_daily_factor", completedDate)
	if err != nil {
		t.Fatalf("Run(calc_daily_factor) error = %v", err)
	}
	if !downstreamCalled {
		t.Fatal("downstream job was not called")
	}

	close(release)
}
