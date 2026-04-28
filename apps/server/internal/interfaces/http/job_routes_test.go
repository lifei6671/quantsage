package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jobdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/job"
)

type fakeJobRunner struct {
	jobName string
	bizDate time.Time
	err     error
}

type fakeJobQueryService struct {
	result jobdomain.QueryResult
	err    error
}

func (r *fakeJobRunner) Run(ctx context.Context, jobName string, bizDate time.Time) error {
	r.jobName = jobName
	r.bizDate = bizDate
	return r.err
}

func (s *fakeJobQueryService) ListJobRuns(ctx context.Context, params jobdomain.QueryParams) (jobdomain.QueryResult, error) {
	return s.result, s.err
}

func TestRunJobRoute(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := &fakeJobRunner{}
	router := NewRouterWithServices(logger, nil, runner)

	req := httptest.NewRequest(http.MethodPost, "/api/jobs/sync_daily_market/run", strings.NewReader(`{"biz_date":"2026-04-27"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":0,"errmsg":"","toast":"","data":{"job_name":"sync_daily_market","status":"queued"}}`)
	if runner.jobName != "sync_daily_market" {
		t.Fatalf("runner.jobName = %q, want %q", runner.jobName, "sync_daily_market")
	}
	if runner.bizDate.Format("2006-01-02") != "2026-04-27" {
		t.Fatalf("runner.bizDate = %s, want %s", runner.bizDate.Format("2006-01-02"), "2026-04-27")
	}
}

func TestListJobsRoute(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithDependencies(logger, nil, nil, nil, &fakeJobQueryService{
		result: jobdomain.QueryResult{
			Items: []jobdomain.JobRun{{
				ID:         1,
				JobName:    "sync_daily_market",
				BizDate:    time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
				Status:     "success",
				StartedAt:  time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC),
				FinishedAt: time.Date(2026, 4, 27, 9, 0, 2, 0, time.UTC),
				CreatedAt:  time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC),
			}},
			Page:     1,
			PageSize: 20,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/jobs?job_name=sync_daily_market&biz_date=2026-04-27&page=1&page_size=20", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":0,"errmsg":"","toast":"","data":{"items":[{"id":1,"job_name":"sync_daily_market","biz_date":"2026-04-27","status":"success","started_at":"2026-04-27T09:00:00Z","finished_at":"2026-04-27T09:00:02Z","error_code":0,"error_message":""}],"page":1,"page_size":20}}`)
}

func TestListJobsRouteMidnightTimestamp(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithDependencies(logger, nil, nil, nil, &fakeJobQueryService{
		result: jobdomain.QueryResult{
			Items: []jobdomain.JobRun{{
				ID:         2,
				JobName:    "sync_trade_calendar",
				BizDate:    time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
				Status:     "success",
				StartedAt:  time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC),
				FinishedAt: time.Date(2026, 4, 28, 0, 0, 1, 0, time.UTC),
				CreatedAt:  time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC),
			}},
			Page:     1,
			PageSize: 20,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/jobs?page=1&page_size=20", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":0,"errmsg":"","toast":"","data":{"items":[{"id":2,"job_name":"sync_trade_calendar","biz_date":"2026-04-27","status":"success","started_at":"2026-04-28T00:00:00Z","finished_at":"2026-04-28T00:00:01Z","error_code":0,"error_message":""}],"page":1,"page_size":20}}`)
}

func TestListJobsRouteNotMountedWithoutQueryService(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithServices(logger, nil, &fakeJobRunner{})

	req := httptest.NewRequest(http.MethodGet, "/api/jobs?page=1&page_size=20", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestRunJobRouteBadRequest(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithServices(logger, nil, &fakeJobRunner{})

	req := httptest.NewRequest(http.MethodPost, "/api/jobs/sync_daily_market/run", strings.NewReader(`{"biz_date":"2026/04/27"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":400001,"errmsg":"bad request","toast":"请求参数不正确","data":{}}`)
}

func TestRunJobRouteNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithServices(logger, nil, &fakeJobRunner{
		err: jobdomain.NewRegistry().Run(context.Background(), "missing_job", time.Now()),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/jobs/missing_job/run", strings.NewReader(`{"biz_date":"2026-04-27"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":404001,"errmsg":"not found","toast":"数据不存在","data":{}}`)
}
