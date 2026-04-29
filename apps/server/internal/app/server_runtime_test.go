package app

import (
	"net/http"
	"testing"

	"github.com/lifei6671/quantsage/apps/server/internal/config"
)

func TestBuildSessionOptionsRejectsInsecureSameSiteNone(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Auth.SessionSameSite = "none"
	cfg.Auth.SessionSecure = false

	_, err := buildSessionOptions(cfg)
	if err == nil {
		t.Fatal("buildSessionOptions() error = nil, want non-nil")
	}
}

func TestBuildSessionOptionsParsesStrictSameSite(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Auth.SessionSameSite = "strict"
	cfg.Auth.SessionSecure = true

	options, err := buildSessionOptions(cfg)
	if err != nil {
		t.Fatalf("buildSessionOptions() error = %v", err)
	}
	if options.SameSite != http.SameSiteStrictMode {
		t.Fatalf("options.SameSite = %v, want %v", options.SameSite, http.SameSiteStrictMode)
	}
}
