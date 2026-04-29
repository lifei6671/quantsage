package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

const defaultSessionName = "quantsage_session"
const defaultSessionSameSite = "lax"

// Config 定义 QuantSage Server 的运行配置。
type Config struct {
	App struct {
		Name string `yaml:"name"`
		Env  string `yaml:"env"`
		Addr string `yaml:"addr"`
	} `yaml:"app"`
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
	Auth struct {
		SessionSecret   string                `yaml:"session_secret"`
		SessionName     string                `yaml:"session_name"`
		SessionSecure   bool                  `yaml:"session_secure"`
		SessionSameSite string                `yaml:"session_same_site"`
		AllowedOrigins  []string              `yaml:"allowed_origins"`
		BootstrapUsers  []BootstrapUserConfig `yaml:"bootstrap_users"`
	} `yaml:"auth"`
}

// BootstrapUserConfig 定义配置文件中的预置账号项。
type BootstrapUserConfig struct {
	Username     string `yaml:"username"`
	DisplayName  string `yaml:"display_name"`
	PasswordHash string `yaml:"password_hash"`
	Status       string `yaml:"status"`
	Role         string `yaml:"role"`
}

// Load 读取 YAML 配置文件，并应用受支持的环境变量覆盖。
func Load(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config file: %w", err)
	}

	applyEnvOverrides(&cfg)
	applyConfigDefaults(&cfg)

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("QUANTSAGE_DATABASE_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("QUANTSAGE_REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("QUANTSAGE_REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("QUANTSAGE_REDIS_DB"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_SESSION_SECRET"); v != "" {
		cfg.Auth.SessionSecret = v
	}
	if v := os.Getenv("QUANTSAGE_SESSION_NAME"); v != "" {
		cfg.Auth.SessionName = v
	}
	if v := os.Getenv("QUANTSAGE_SESSION_SECURE"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.Auth.SessionSecure = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_SESSION_SAME_SITE"); v != "" {
		cfg.Auth.SessionSameSite = v
	}
	if v := os.Getenv("QUANTSAGE_CORS_ALLOWED_ORIGINS"); v != "" {
		cfg.Auth.AllowedOrigins = splitCommaSeparatedValues(v)
	}
}

func applyConfigDefaults(cfg *Config) {
	if cfg.Auth.SessionName == "" {
		cfg.Auth.SessionName = defaultSessionName
	}
	if cfg.Auth.SessionSameSite == "" {
		cfg.Auth.SessionSameSite = defaultSessionSameSite
	}
	cfg.Auth.AllowedOrigins = compactStrings(cfg.Auth.AllowedOrigins)
	for index := range cfg.Auth.BootstrapUsers {
		if cfg.Auth.BootstrapUsers[index].Status == "" {
			cfg.Auth.BootstrapUsers[index].Status = "active"
		}
		if cfg.Auth.BootstrapUsers[index].Role == "" {
			cfg.Auth.BootstrapUsers[index].Role = "user"
		}
	}
}

func splitCommaSeparatedValues(value string) []string {
	return compactStrings(strings.Split(value, ","))
}

func compactStrings(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}

	return result
}
