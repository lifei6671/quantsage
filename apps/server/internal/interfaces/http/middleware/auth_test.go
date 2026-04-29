package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	userdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/user"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

type fakeUserService struct {
	getByIDFunc func(ctx context.Context, userID int64) (userdomain.User, error)
}

func (f fakeUserService) GetByID(ctx context.Context, userID int64) (userdomain.User, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(ctx, userID)
	}
	return userdomain.User{}, nil
}

func (f fakeUserService) Authenticate(ctx context.Context, username, password string) (userdomain.User, error) {
	return userdomain.User{}, errors.New("not implemented")
}

func (f fakeUserService) SyncBootstrapUsers(ctx context.Context, users []userdomain.BootstrapUser) error {
	return errors.New("not implemented")
}

func TestAuthRequiredClearsPersistedInvalidSession(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(sessions.Sessions("test_session", cookie.NewStore([]byte("test-secret"))))
	router.GET("/seed", func(c *gin.Context) {
		if err := SetSessionUserID(c, 9); err != nil {
			t.Fatalf("SetSessionUserID() error = %v", err)
		}
		c.Status(http.StatusNoContent)
	})
	router.GET("/private",
		AuthRequired(fakeUserService{
			getByIDFunc: func(ctx context.Context, userID int64) (userdomain.User, error) {
				return userdomain.User{}, apperror.New(apperror.CodeUnauthorized, errors.New("user not found"))
			},
		}),
		func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	)
	router.GET("/check", func(c *gin.Context) {
		if _, err := sessionUserID(c); err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}
		c.Status(http.StatusNoContent)
	})

	sessionCookie := performRequest(t, router, http.MethodGet, "/seed", "")
	if sessionCookie == "" {
		t.Fatal("seed response did not set session cookie")
	}

	privateRecorder := performRecorderRequest(router, http.MethodGet, "/private", sessionCookie)
	if updatedCookie := privateRecorder.Header().Get("Set-Cookie"); updatedCookie != "" {
		sessionCookie = updatedCookie
	} else {
		t.Fatal("private response did not persist cleared session cookie")
	}

	checkRecorder := performRecorderRequest(router, http.MethodGet, "/check", sessionCookie)
	if got := checkRecorder.Code; got != http.StatusUnauthorized {
		t.Fatalf("check response status = %d, want %d", got, http.StatusUnauthorized)
	}
}

func TestAuthRequiredKeepsSessionOnDatabaseError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(sessions.Sessions("test_session", cookie.NewStore([]byte("test-secret"))))
	router.GET("/seed", func(c *gin.Context) {
		if err := SetSessionUserID(c, 9); err != nil {
			t.Fatalf("SetSessionUserID() error = %v", err)
		}
		c.Status(http.StatusNoContent)
	})
	router.GET("/private",
		AuthRequired(fakeUserService{
			getByIDFunc: func(ctx context.Context, userID int64) (userdomain.User, error) {
				return userdomain.User{}, apperror.New(apperror.CodeDatabaseError, errors.New("database unavailable"))
			},
		}),
		func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	)
	router.GET("/check", func(c *gin.Context) {
		if _, err := sessionUserID(c); err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}
		c.Status(http.StatusNoContent)
	})

	sessionCookie := performRequest(t, router, http.MethodGet, "/seed", "")
	if sessionCookie == "" {
		t.Fatal("seed response did not set session cookie")
	}

	privateRecorder := performRecorderRequest(router, http.MethodGet, "/private", sessionCookie)
	if updatedCookie := privateRecorder.Header().Get("Set-Cookie"); updatedCookie != "" {
		t.Fatalf("private response unexpectedly rewrote session cookie: %s", updatedCookie)
	}

	checkRecorder := performRecorderRequest(router, http.MethodGet, "/check", sessionCookie)
	if got := checkRecorder.Code; got != http.StatusNoContent {
		t.Fatalf("check response status = %d, want %d", got, http.StatusNoContent)
	}
}

func performRequest(t *testing.T, handler http.Handler, method, path, cookie string) string {
	t.Helper()

	recorder := performRecorderRequest(handler, method, path, cookie)
	return recorder.Header().Get("Set-Cookie")
}

func performRecorderRequest(handler http.Handler, method, path, cookie string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, nil)
	if cookie != "" {
		request.Header.Set("Cookie", cookie)
	}
	handler.ServeHTTP(recorder, request)
	return recorder
}
