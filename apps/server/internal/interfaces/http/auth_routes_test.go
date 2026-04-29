package http

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/sessions/cookie"

	jobdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/job"
	userdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/user"
)

type fakeAuthUserService struct{}

func (fakeAuthUserService) GetByID(ctx context.Context, userID int64) (userdomain.User, error) {
	return userdomain.User{ID: userID, Username: "admin", Status: "active"}, nil
}

func (fakeAuthUserService) Authenticate(ctx context.Context, username, password string) (userdomain.User, error) {
	return userdomain.User{}, errors.New("not implemented")
}

func (fakeAuthUserService) SyncBootstrapUsers(ctx context.Context, users []userdomain.BootstrapUser) error {
	return nil
}

func TestAuthEnabledProtectsSharedRoutes(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithRuntime(logger, RouterDependencies{
		StockService: &fakeStockService{},
		UserService:  fakeAuthUserService{},
		SessionStore: cookie.NewStore([]byte("test-secret")),
		SessionName:  "test_session",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/stocks", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":401001,"errmsg":"unauthorized","toast":"请先登录","data":{}}`)
}

func TestInternalJobRunAllowsLoopbackWithoutSession(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := &fakeInternalJobRunner{}
	router := NewRouterWithRuntime(logger, RouterDependencies{
		JobRunner: runner,
	})

	req := httptest.NewRequest(http.MethodPost, "/internal/jobs/sync_daily_market/run", strings.NewReader(`{"biz_date":"2026-04-27"}`))
	req.RemoteAddr = "127.0.0.1:34567"
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":0,"errmsg":"","toast":"","data":{"job_name":"sync_daily_market","status":"queued"}}`)
	if runner.jobName != "sync_daily_market" {
		t.Fatalf("runner.jobName = %q, want %q", runner.jobName, "sync_daily_market")
	}
	if got := runner.bizDate.Format("2006-01-02"); got != "2026-04-27" {
		t.Fatalf("runner.bizDate = %q, want %q", got, "2026-04-27")
	}
}

func TestInternalJobRunRejectsNonLoopback(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithRuntime(logger, RouterDependencies{
		JobRunner: &fakeInternalJobRunner{},
	})

	req := httptest.NewRequest(http.MethodPost, "/internal/jobs/sync_daily_market/run", strings.NewReader(`{"biz_date":"2026-04-27"}`))
	req.RemoteAddr = "203.0.113.10:34567"
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":403001,"errmsg":"forbidden","toast":"没有操作权限","data":{}}`)
}

type fakeInternalJobRunner struct {
	jobName string
	bizDate time.Time
}

func (f *fakeInternalJobRunner) Run(ctx context.Context, jobName string, bizDate time.Time) error {
	_ = ctx
	f.jobName = jobName
	f.bizDate = bizDate
	return nil
}

var _ jobdomain.Runner = (*fakeInternalJobRunner)(nil)
