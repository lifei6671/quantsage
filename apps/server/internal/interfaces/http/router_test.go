package http

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	infraLog "github.com/lifei6671/quantsage/apps/server/internal/infra/log"
	httpmiddleware "github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/middleware"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

func TestHealthz(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(logger)

	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body response.Body
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	if body.Code != apperror.CodeOK {
		t.Fatalf("code = %d, want %d", body.Code, apperror.CodeOK)
	}

	data, ok := body.Data.(map[string]any)
	if !ok {
		t.Fatalf("data type = %T, want map[string]any", body.Data)
	}
	if data["status"] != "ok" {
		t.Fatalf("status = %v, want %q", data["status"], "ok")
	}
}

func TestNewRouterDoesNotExposeBusinessRoutes(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(logger)

	req := httptest.NewRequest(http.MethodGet, "/api/stocks", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestRequestLogIncludesContextFields(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	router := gin.New()
	router.Use(httpmiddleware.RequestLog(logger))
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(infraLog.AddInfo(c.Request.Context(), infraLog.String("ts_code", "000001.SZ")))
		c.Next()
	})
	router.GET("/probe", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if !bytes.Contains(buf.Bytes(), []byte(`"ts_code":"000001.SZ"`)) {
		t.Fatalf("log output = %q, want ts_code field", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"status":200`)) {
		t.Fatalf("log output = %q, want status field", buf.String())
	}
}

func TestCORSPreflight(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(logger)

	req := httptest.NewRequest(http.MethodOptions, "/api/healthz", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}
