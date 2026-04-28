package scheduler

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

type fakeRunner struct {
	called  bool
	jobName string
	bizDate time.Time
}

func (r *fakeRunner) Run(ctx context.Context, jobName string, bizDate time.Time) error {
	r.called = true
	r.jobName = jobName
	r.bizDate = bizDate
	return nil
}

func TestRegisterTaskNames(t *testing.T) {
	t.Parallel()

	s := New(slog.New(slog.NewTextHandler(io.Discard, nil)), &fakeRunner{})
	if err := s.Register(TaskSpec{JobName: "sync_stock_basic"}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := s.Register(TaskSpec{JobName: "calc_daily_factor"}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	names := s.TaskNames()
	if len(names) != 2 {
		t.Fatalf("len(names) = %d, want %d", len(names), 2)
	}
	if names[0] != "calc_daily_factor" || names[1] != "sync_stock_basic" {
		t.Fatalf("names = %v, want [calc_daily_factor sync_stock_basic]", names)
	}
}

func TestRegisterInvalidSpec(t *testing.T) {
	t.Parallel()

	s := New(slog.New(slog.NewTextHandler(io.Discard, nil)), &fakeRunner{})
	err := s.Register(TaskSpec{JobName: "sync_stock_basic", Spec: "invalid cron"})
	if err == nil {
		t.Fatal("Register() error = nil, want non-nil")
	}
}

func TestSchedulerBizDate(t *testing.T) {
	t.Parallel()

	got := schedulerBizDate(time.Date(2026, 4, 28, 18, 5, 6, 0, time.UTC))
	want := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("schedulerBizDate() = %s, want %s", got, want)
	}
}
