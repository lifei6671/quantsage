package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesAuthBootstrapUsers(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
app:
  name: quantsage
  env: local
  addr: ":8080"
database:
  dsn: postgres://demo
redis:
  addr: 127.0.0.1:6379
  password: ""
  db: 2
auth:
  session_secret: "***"
  session_same_site: strict
  allowed_origins:
    - https://console.example.com
  bootstrap_users:
    - username: admin
      display_name: 管理员
      password_hash: "$2a$10$demo"
      role: admin
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Auth.SessionName != defaultSessionName {
		t.Fatalf("cfg.Auth.SessionName = %q, want %q", cfg.Auth.SessionName, defaultSessionName)
	}
	if cfg.Auth.SessionSameSite != "strict" {
		t.Fatalf("cfg.Auth.SessionSameSite = %q, want %q", cfg.Auth.SessionSameSite, "strict")
	}
	if len(cfg.Auth.AllowedOrigins) != 1 || cfg.Auth.AllowedOrigins[0] != "https://console.example.com" {
		t.Fatalf("cfg.Auth.AllowedOrigins = %v, want [%q]", cfg.Auth.AllowedOrigins, "https://console.example.com")
	}
	if len(cfg.Auth.BootstrapUsers) != 1 {
		t.Fatalf("len(cfg.Auth.BootstrapUsers) = %d, want %d", len(cfg.Auth.BootstrapUsers), 1)
	}
	if cfg.Auth.BootstrapUsers[0].Username != "admin" {
		t.Fatalf("cfg.Auth.BootstrapUsers[0].Username = %q, want %q", cfg.Auth.BootstrapUsers[0].Username, "admin")
	}
	if cfg.Auth.BootstrapUsers[0].Status != "active" {
		t.Fatalf("cfg.Auth.BootstrapUsers[0].Status = %q, want %q", cfg.Auth.BootstrapUsers[0].Status, "active")
	}
}

func TestLoadAppliesEnvOverrides(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
app:
  name: quantsage
database:
  dsn: postgres://from-file
redis:
  addr: 127.0.0.1:6379
  password: ""
  db: 0
auth:
  session_secret: from-file
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("QUANTSAGE_DATABASE_DSN", "postgres://from-env")
	t.Setenv("QUANTSAGE_REDIS_ADDR", "10.0.0.1:6379")
	t.Setenv("QUANTSAGE_REDIS_PASSWORD", "***")
	t.Setenv("QUANTSAGE_REDIS_DB", "4")
	t.Setenv("QUANTSAGE_SESSION_SECRET", "from-env")
	t.Setenv("QUANTSAGE_SESSION_NAME", "custom_session")
	t.Setenv("QUANTSAGE_SESSION_SECURE", "true")
	t.Setenv("QUANTSAGE_SESSION_SAME_SITE", "none")
	t.Setenv("QUANTSAGE_CORS_ALLOWED_ORIGINS", "https://console.example.com, https://ops.example.com")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.DSN != "postgres://from-env" {
		t.Fatalf("cfg.Database.DSN = %q, want %q", cfg.Database.DSN, "postgres://from-env")
	}
	if cfg.Redis.Addr != "10.0.0.1:6379" {
		t.Fatalf("cfg.Redis.Addr = %q, want %q", cfg.Redis.Addr, "10.0.0.1:6379")
	}
	if cfg.Redis.Password != "***" {
		t.Fatalf("cfg.Redis.Password = %q, want %q", cfg.Redis.Password, "***")
	}
	if cfg.Redis.DB != 4 {
		t.Fatalf("cfg.Redis.DB = %d, want %d", cfg.Redis.DB, 4)
	}
	if cfg.Auth.SessionSecret != "from-env" {
		t.Fatalf("cfg.Auth.SessionSecret = %q, want %q", cfg.Auth.SessionSecret, "from-env")
	}
	if !cfg.Auth.SessionSecure {
		t.Fatal("cfg.Auth.SessionSecure = false, want true")
	}
	if cfg.Auth.SessionSameSite != "none" {
		t.Fatalf("cfg.Auth.SessionSameSite = %q, want %q", cfg.Auth.SessionSameSite, "none")
	}
	if len(cfg.Auth.AllowedOrigins) != 2 || cfg.Auth.AllowedOrigins[0] != "https://console.example.com" || cfg.Auth.AllowedOrigins[1] != "https://ops.example.com" {
		t.Fatalf("cfg.Auth.AllowedOrigins = %v, want two configured origins", cfg.Auth.AllowedOrigins)
	}
}
