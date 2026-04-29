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
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

type fakeAuthUserService struct {
	getByIDFunc      func(ctx context.Context, userID int64) (userdomain.User, error)
	authenticateFunc func(ctx context.Context, username, password string) (userdomain.User, error)
}

func (f fakeAuthUserService) GetByID(ctx context.Context, userID int64) (userdomain.User, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(ctx, userID)
	}
	return userdomain.User{ID: userID, Username: "admin", Status: "active"}, nil
}

func (f fakeAuthUserService) Authenticate(ctx context.Context, username, password string) (userdomain.User, error) {
	if f.authenticateFunc != nil {
		return f.authenticateFunc(ctx, username, password)
	}
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

func TestAuthLoginSuccessRestoresCurrentUser(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithRuntime(logger, RouterDependencies{
		UserService: fakeAuthUserService{
			authenticateFunc: func(ctx context.Context, username, password string) (userdomain.User, error) {
				if username != "admin" || password != "admin123" {
					return userdomain.User{}, apperror.New(apperror.CodeUnauthorized, errors.New("invalid credentials"))
				}
				return userdomain.User{ID: 7, Username: "admin", DisplayName: "管理员", Status: "active", Role: "admin"}, nil
			},
			getByIDFunc: func(ctx context.Context, userID int64) (userdomain.User, error) {
				return userdomain.User{ID: userID, Username: "admin", DisplayName: "管理员", Status: "active", Role: "admin"}, nil
			},
		},
		SessionStore: cookie.NewStore([]byte("test-secret")),
		SessionName:  "test_session",
	})

	loginRecorder := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"admin123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(loginRecorder, loginReq)

	assertResponseJSON(t, loginRecorder, `{"code":0,"errmsg":"","toast":"","data":{"id":7,"username":"admin","display_name":"管理员","status":"active","role":"admin"}}`)
	sessionCookie := loginRecorder.Header().Get("Set-Cookie")
	if sessionCookie == "" {
		t.Fatal("login response did not set session cookie")
	}

	meRecorder := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Cookie", sessionCookie)
	router.ServeHTTP(meRecorder, meReq)

	assertResponseJSON(t, meRecorder, `{"code":0,"errmsg":"","toast":"","data":{"id":7,"username":"admin","display_name":"管理员","status":"active","role":"admin"}}`)
}

func TestAuthLoginRejectsWrongPassword(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithRuntime(logger, RouterDependencies{
		UserService: fakeAuthUserService{
			authenticateFunc: func(ctx context.Context, username, password string) (userdomain.User, error) {
				return userdomain.User{}, apperror.New(apperror.CodeUnauthorized, errors.New("invalid credentials"))
			},
		},
		SessionStore: cookie.NewStore([]byte("test-secret")),
		SessionName:  "test_session",
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":401001,"errmsg":"unauthorized","toast":"请先登录","data":{}}`)
	if sessionCookie := recorder.Header().Get("Set-Cookie"); sessionCookie != "" {
		t.Fatalf("login failure unexpectedly set session cookie: %s", sessionCookie)
	}
}

func TestAuthLogoutInvalidatesSession(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithRuntime(logger, RouterDependencies{
		UserService: fakeAuthUserService{
			authenticateFunc: func(ctx context.Context, username, password string) (userdomain.User, error) {
				return userdomain.User{ID: 7, Username: "admin", DisplayName: "管理员", Status: "active", Role: "admin"}, nil
			},
			getByIDFunc: func(ctx context.Context, userID int64) (userdomain.User, error) {
				return userdomain.User{ID: userID, Username: "admin", DisplayName: "管理员", Status: "active", Role: "admin"}, nil
			},
		},
		SessionStore: cookie.NewStore([]byte("test-secret")),
		SessionName:  "test_session",
	})

	loginRecorder := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"admin123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(loginRecorder, loginReq)
	sessionCookie := loginRecorder.Header().Get("Set-Cookie")
	if sessionCookie == "" {
		t.Fatal("login response did not set session cookie")
	}

	logoutRecorder := httptest.NewRecorder()
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.Header.Set("Cookie", sessionCookie)
	router.ServeHTTP(logoutRecorder, logoutReq)
	assertResponseJSON(t, logoutRecorder, `{"code":0,"errmsg":"","toast":"","data":{"status":"ok"}}`)
	clearedCookie := logoutRecorder.Header().Get("Set-Cookie")
	if clearedCookie == "" {
		t.Fatal("logout response did not persist cleared session cookie")
	}

	meRecorder := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Cookie", clearedCookie)
	router.ServeHTTP(meRecorder, meReq)
	assertResponseJSON(t, meRecorder, `{"code":401001,"errmsg":"unauthorized","toast":"请先登录","data":{}}`)
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
