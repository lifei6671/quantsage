package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestBuildLocalTaskSpecs(t *testing.T) {
	t.Parallel()

	jobNames := []string{
		localDailyMarketPipeline,
		"sync_stock_basic",
		"sync_trade_calendar",
	}

	items, err := buildLocalTaskSpecs(jobNames)
	if err != nil {
		t.Fatalf("buildLocalTaskSpecs() error = %v", err)
	}
	if len(items) != len(jobNames) {
		t.Fatalf("len(items) = %d, want %d", len(items), len(jobNames))
	}
	for index, item := range items {
		if item.JobName != jobNames[index] {
			t.Fatalf("items[%d].JobName = %q, want %q", index, item.JobName, jobNames[index])
		}
		if item.Spec == "" {
			t.Fatalf("items[%d].Spec = empty, want cron spec", index)
		}
	}
}

func TestBuildLocalTaskSpecsUnknownJob(t *testing.T) {
	t.Parallel()

	if _, err := buildLocalTaskSpecs([]string{"unknown_job"}); err == nil {
		t.Fatal("buildLocalTaskSpecs() error = nil, want non-nil")
	}
}

func TestLocalTaskNames(t *testing.T) {
	t.Parallel()

	names := localTaskNames()
	if len(names) != len(localTaskCronSpecs) {
		t.Fatalf("len(names) = %d, want %d", len(names), len(localTaskCronSpecs))
	}
	if names[0] != localDailyMarketPipeline {
		t.Fatalf("names[0] = %q, want %q", names[0], localDailyMarketPipeline)
	}
	if names[len(names)-1] != "sync_trade_calendar" {
		t.Fatalf("last name = %q, want %q", names[len(names)-1], "sync_trade_calendar")
	}
}

func TestLocalPipelineRunnerRunsDailyMarketStepsInOrder(t *testing.T) {
	t.Parallel()

	base := &fakePipelineBaseRunner{}
	runner := newLocalPipelineRunner(base)
	bizDate := time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC)
	if err := runner.Run(context.Background(), localDailyMarketPipeline, bizDate); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !reflect.DeepEqual(base.calls, localDailyMarketPipelineSteps) {
		t.Fatalf("calls = %v, want %v", base.calls, localDailyMarketPipelineSteps)
	}
}

func TestLocalPipelineRunnerStopsAfterStepError(t *testing.T) {
	t.Parallel()

	base := &fakePipelineBaseRunner{
		errs: map[string]error{
			"calc_daily_factor": errors.New("dependency still running"),
		},
	}
	runner := newLocalPipelineRunner(base)
	err := runner.Run(context.Background(), localDailyMarketPipeline, time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}

	wantCalls := []string{"sync_daily_market", "calc_daily_factor"}
	if !reflect.DeepEqual(base.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", base.calls, wantCalls)
	}
}

func TestLocalPipelineRunnerDelegatesIndependentJob(t *testing.T) {
	t.Parallel()

	base := &fakePipelineBaseRunner{}
	runner := newLocalPipelineRunner(base)
	if err := runner.Run(context.Background(), "sync_stock_basic", time.Date(2026, 4, 28, 8, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	wantCalls := []string{"sync_stock_basic"}
	if !reflect.DeepEqual(base.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", base.calls, wantCalls)
	}
}

func TestResolveServerBaseURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		addr      string
		want      string
		expectErr bool
	}{
		{name: "empty host uses loopback", addr: ":8080", want: "http://127.0.0.1:8080"},
		{name: "wildcard host uses loopback", addr: "0.0.0.0:8080", want: "http://127.0.0.1:8080"},
		{name: "explicit host kept", addr: "192.168.10.8:8080", want: "http://192.168.10.8:8080"},
		{name: "full url kept", addr: "http://127.0.0.1:8080/", want: "http://127.0.0.1:8080"},
		{name: "invalid addr", addr: "8080", expectErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveServerBaseURL(tc.addr)
			if tc.expectErr {
				if err == nil {
					t.Fatal("resolveServerBaseURL() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveServerBaseURL() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("resolveServerBaseURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLocalAPIRunnerRun(t *testing.T) {
	t.Parallel()

	requests := make(chan runJobRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			t.Fatalf("request method = %s, want %s", req.Method, http.MethodPost)
		}
		if req.URL.Path != "/internal/jobs/sync_daily_market/run" {
			t.Fatalf("request path = %s, want %s", req.URL.Path, "/internal/jobs/sync_daily_market/run")
		}

		var body runJobRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requests <- body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"errmsg":"","toast":"","data":{"job_name":"sync_daily_market","status":"queued"}}`))
	}))
	defer server.Close()

	runner, err := newLocalAPIRunner(server.URL)
	if err != nil {
		t.Fatalf("newLocalAPIRunner() error = %v", err)
	}

	runDate := time.Date(2026, 4, 28, 18, 5, 0, 0, time.UTC)
	if err := runner.Run(context.Background(), "sync_daily_market", runDate); err != nil {
		t.Fatalf("runner.Run() error = %v", err)
	}

	select {
	case request := <-requests:
		if request.BizDate != "2026-04-28" {
			t.Fatalf("request.BizDate = %q, want %q", request.BizDate, "2026-04-28")
		}
	default:
		t.Fatal("request was not received")
	}
}

func TestNewLocalAPIRunnerUsesJobTimeout(t *testing.T) {
	t.Parallel()

	runner, err := newLocalAPIRunner("127.0.0.1:8080")
	if err != nil {
		t.Fatalf("newLocalAPIRunner() error = %v", err)
	}

	apiRunner, ok := runner.(*localAPIRunner)
	if !ok {
		t.Fatalf("runner type = %T, want *localAPIRunner", runner)
	}
	if apiRunner.client.Timeout != localAPIJobTimeout {
		t.Fatalf("client.Timeout = %s, want %s", apiRunner.client.Timeout, localAPIJobTimeout)
	}
	if apiRunner.client.Timeout <= 10*time.Second {
		t.Fatalf("client.Timeout = %s, want longer than 10s", apiRunner.client.Timeout)
	}
}

func TestLocalAPIRunnerRunBusinessError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":500001,"errmsg":"job failed","toast":"","data":{}}`))
	}))
	defer server.Close()

	runner, err := newLocalAPIRunner(server.URL)
	if err != nil {
		t.Fatalf("newLocalAPIRunner() error = %v", err)
	}

	err = runner.Run(context.Background(), "sync_daily_market", time.Date(2026, 4, 28, 18, 5, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("runner.Run() error = nil, want non-nil")
	}
}

type fakePipelineBaseRunner struct {
	calls []string
	errs  map[string]error
}

func (r *fakePipelineBaseRunner) Run(ctx context.Context, jobName string, bizDate time.Time) error {
	_ = ctx
	_ = bizDate

	r.calls = append(r.calls, jobName)
	if r.errs == nil {
		return nil
	}
	return r.errs[jobName]
}
