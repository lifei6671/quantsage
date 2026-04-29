package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSPreflightAllowsCredentialedPrivateMethods(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS([]string{"http://127.0.0.1:4173"}))
	router.OPTIONS("/api/watchlists/1", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/watchlists/1", nil)
	request.Header.Set("Origin", "http://127.0.0.1:4173")
	request.Header.Set("Access-Control-Request-Method", http.MethodDelete)
	request.Header.Set("Access-Control-Request-Headers", "Content-Type, X-Request-Id")

	router.ServeHTTP(recorder, request)

	if got := recorder.Code; got != http.StatusNoContent {
		t.Fatalf("response status = %d, want %d", got, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:4173" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "http://127.0.0.1:4173")
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want %q", got, "true")
	}
	allowMethods := recorder.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(allowMethods, http.MethodPut) || !strings.Contains(allowMethods, http.MethodDelete) {
		t.Fatalf("Access-Control-Allow-Methods = %q, want PUT and DELETE included", allowMethods)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, X-Request-Id" {
		t.Fatalf("Access-Control-Allow-Headers = %q, want %q", got, "Content-Type, X-Request-Id")
	}
}

func TestCORSRejectsUntrustedCredentialedOrigin(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS([]string{"https://console.example.com"}))
	router.GET("/api/auth/me", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	request.Header.Set("Origin", "https://attacker.example.com")

	router.ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty for untrusted origin", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want empty for untrusted origin", got)
	}
}
