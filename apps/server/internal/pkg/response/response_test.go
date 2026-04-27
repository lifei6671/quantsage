package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

func TestOK(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	OK(ctx, gin.H{"status": "ok"})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body Body
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	if body.Code != apperror.CodeOK {
		t.Fatalf("code = %d, want %d", body.Code, apperror.CodeOK)
	}
	if body.Errmsg != "" || body.Toast != "" {
		t.Fatalf("errmsg/toast = %q/%q, want empty", body.Errmsg, body.Toast)
	}
}

func TestFail(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	Fail(ctx, apperror.New(apperror.CodeBadRequest, errors.New("bad payload")))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body Body
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	if body.Code != apperror.CodeBadRequest {
		t.Fatalf("code = %d, want %d", body.Code, apperror.CodeBadRequest)
	}
	if body.Errmsg != "bad request" {
		t.Fatalf("errmsg = %q, want %q", body.Errmsg, "bad request")
	}
	if body.Toast != "请求参数不正确" {
		t.Fatalf("toast = %q, want %q", body.Toast, "请求参数不正确")
	}
}
