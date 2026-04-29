package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/consts"
)

const defaultSessionName = "quantsage_session"
const defaultSessionSameSite = "lax"
const defaultDatasourceSource = consts.DatasourceTushare
const defaultEastMoneyEndpoint = "https://push2his.eastmoney.com"
const defaultEastMoneyQuoteEndpoint = "https://push2his.eastmoney.com"
const defaultEastMoneyTimeoutSeconds = 30
const defaultEastMoneyMaxRetries = 2
const defaultEastMoneyUserAgentMode = "stable"
const defaultEastMoneyFetchMode = "auto"
const defaultEastMoneyBrowserTimeoutSeconds = 60
const defaultEastMoneyBrowserCookieTTLSeconds = 720
const defaultEastMoneyBrowserHeadless = true
const defaultEastMoneyBrowserUserAgentMode = defaultEastMoneyUserAgentMode
const defaultEastMoneyBrowserUserAgentPlatform = "desktop"
const defaultEastMoneyBrowserCount = 1
const defaultEastMoneyBrowserMaxConcurrentTabs = 4
const defaultEastMoneyBrowserTabsPerBrowser = defaultEastMoneyBrowserMaxConcurrentTabs
const defaultEastMoneyBrowserRecycleAfterTabs = 200
const defaultEastMoneyBrowserWaitReadySelector = "body"
const defaultEastMoneyBrowserAcceptLanguage = "zh-CN,zh;q=0.9,en;q=0.8"
const defaultEastMoneyBrowserWindowWidth = 1366
const defaultEastMoneyBrowserWindowHeight = 768

// DatasourceConfig 定义导入数据源相关配置。
type DatasourceConfig struct {
	DefaultSource string          `yaml:"default_source"`
	Tushare       TushareConfig   `yaml:"tushare"`
	EastMoney     EastMoneyConfig `yaml:"eastmoney"`
}

// TushareConfig 定义 Tushare 数据源配置。
type TushareConfig struct {
	Token string `yaml:"token"`
}

// EastMoneyConfig 定义东方财富导入源的基础配置。
type EastMoneyConfig struct {
	Endpoint                  string   `yaml:"endpoint"`
	QuoteEndpoint             string   `yaml:"quote_endpoint"`
	TimeoutSeconds            int      `yaml:"timeout_seconds"`
	MaxRetries                int      `yaml:"max_retries"`
	UserAgentMode             string   `yaml:"user_agent_mode"`
	FetchMode                 string   `yaml:"fetch_mode"`
	BrowserPath               string   `yaml:"browser_path"`
	BrowserTimeoutSeconds     int      `yaml:"browser_timeout_seconds"`
	BrowserCookieTTLSeconds   int      `yaml:"browser_cookie_ttl_seconds"`
	BrowserHeadless           bool     `yaml:"browser_headless"`
	BrowserUserAgentMode      string   `yaml:"browser_user_agent_mode"`
	BrowserUserAgentPlatform  string   `yaml:"browser_user_agent_platform"`
	BrowserCount              int      `yaml:"browser_count"`
	BrowserMaxConcurrentTabs  int      `yaml:"browser_max_concurrent_tabs"`
	BrowserTabsPerBrowser     int      `yaml:"browser_tabs_per_browser"`
	BrowserRecycleAfterTabs   int      `yaml:"browser_recycle_after_tabs"`
	BrowserWaitReadySelector  string   `yaml:"browser_wait_ready_selector"`
	BrowserAcceptLanguage     string   `yaml:"browser_accept_language"`
	BrowserDisableImages      bool     `yaml:"browser_disable_images"`
	BrowserNoSandbox          bool     `yaml:"browser_no_sandbox"`
	BrowserWindowWidth        int      `yaml:"browser_window_width"`
	BrowserWindowHeight       int      `yaml:"browser_window_height"`
	BrowserBlockedURLPatterns []string `yaml:"browser_blocked_url_patterns"`
	BrowserExtraFlags         []string `yaml:"browser_extra_flags"`

	// browserHeadlessConfigured 用于区分“未配置”和“显式配置为 false”。
	browserHeadlessConfigured bool `yaml:"-"`
}

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
	Datasource DatasourceConfig `yaml:"datasource"`
	Auth       struct {
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
	var raw struct {
		Datasource struct {
			EastMoney struct {
				BrowserHeadless *bool `yaml:"browser_headless"`
			} `yaml:"eastmoney"`
		} `yaml:"datasource"`
	}
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal config presence flags: %w", err)
	}
	if raw.Datasource.EastMoney.BrowserHeadless != nil {
		cfg.Datasource.EastMoney.BrowserHeadless = *raw.Datasource.EastMoney.BrowserHeadless
		cfg.Datasource.EastMoney.browserHeadlessConfigured = true
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
	if v := os.Getenv("QUANTSAGE_DATASOURCE_DEFAULT_SOURCE"); v != "" {
		cfg.Datasource.DefaultSource = v
	}
	if v := os.Getenv("QUANTSAGE_TUSHARE_TOKEN"); v != "" {
		cfg.Datasource.Tushare.Token = v
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_ENDPOINT"); v != "" {
		cfg.Datasource.EastMoney.Endpoint = v
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_QUOTE_ENDPOINT"); v != "" {
		cfg.Datasource.EastMoney.QuoteEndpoint = v
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Datasource.EastMoney.TimeoutSeconds = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_MAX_RETRIES"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Datasource.EastMoney.MaxRetries = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_USER_AGENT_MODE"); v != "" {
		cfg.Datasource.EastMoney.UserAgentMode = v
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_FETCH_MODE"); v != "" {
		cfg.Datasource.EastMoney.FetchMode = v
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_PATH"); v != "" {
		cfg.Datasource.EastMoney.BrowserPath = v
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Datasource.EastMoney.BrowserTimeoutSeconds = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_COOKIE_TTL_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Datasource.EastMoney.BrowserCookieTTLSeconds = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_HEADLESS"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.Datasource.EastMoney.BrowserHeadless = parsed
			cfg.Datasource.EastMoney.browserHeadlessConfigured = true
		}
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_USER_AGENT_MODE"); v != "" {
		cfg.Datasource.EastMoney.BrowserUserAgentMode = v
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_USER_AGENT_PLATFORM"); v != "" {
		cfg.Datasource.EastMoney.BrowserUserAgentPlatform = v
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_COUNT"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Datasource.EastMoney.BrowserCount = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_MAX_CONCURRENT_TABS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_TABS_PER_BROWSER"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Datasource.EastMoney.BrowserTabsPerBrowser = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_RECYCLE_AFTER_TABS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Datasource.EastMoney.BrowserRecycleAfterTabs = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_WAIT_READY_SELECTOR"); v != "" {
		cfg.Datasource.EastMoney.BrowserWaitReadySelector = v
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_ACCEPT_LANGUAGE"); v != "" {
		cfg.Datasource.EastMoney.BrowserAcceptLanguage = v
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_DISABLE_IMAGES"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.Datasource.EastMoney.BrowserDisableImages = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_NO_SANDBOX"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.Datasource.EastMoney.BrowserNoSandbox = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_WINDOW_WIDTH"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Datasource.EastMoney.BrowserWindowWidth = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_WINDOW_HEIGHT"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Datasource.EastMoney.BrowserWindowHeight = parsed
		}
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_BLOCKED_URL_PATTERNS"); v != "" {
		cfg.Datasource.EastMoney.BrowserBlockedURLPatterns = splitCommaSeparatedValues(v)
	}
	if v := os.Getenv("QUANTSAGE_EASTMONEY_BROWSER_EXTRA_FLAGS"); v != "" {
		cfg.Datasource.EastMoney.BrowserExtraFlags = splitCommaSeparatedValues(v)
	}
}

func applyConfigDefaults(cfg *Config) {
	if cfg.Auth.SessionName == "" {
		cfg.Auth.SessionName = defaultSessionName
	}
	if cfg.Auth.SessionSameSite == "" {
		cfg.Auth.SessionSameSite = defaultSessionSameSite
	}
	// 默认导入源与东财接入参数都在这里统一归一化，避免运行时再散落兜底。
	cfg.Datasource.DefaultSource = strings.ToLower(strings.TrimSpace(cfg.Datasource.DefaultSource))
	if cfg.Datasource.DefaultSource == "" {
		cfg.Datasource.DefaultSource = defaultDatasourceSource
	}
	cfg.Datasource.Tushare.Token = strings.TrimSpace(cfg.Datasource.Tushare.Token)
	cfg.Datasource.EastMoney.Endpoint = strings.TrimSpace(cfg.Datasource.EastMoney.Endpoint)
	if cfg.Datasource.EastMoney.Endpoint == "" {
		cfg.Datasource.EastMoney.Endpoint = defaultEastMoneyEndpoint
	}
	cfg.Datasource.EastMoney.QuoteEndpoint = strings.TrimSpace(cfg.Datasource.EastMoney.QuoteEndpoint)
	if cfg.Datasource.EastMoney.QuoteEndpoint == "" {
		cfg.Datasource.EastMoney.QuoteEndpoint = defaultEastMoneyQuoteEndpoint
	}
	if cfg.Datasource.EastMoney.TimeoutSeconds <= 0 {
		cfg.Datasource.EastMoney.TimeoutSeconds = defaultEastMoneyTimeoutSeconds
	}
	if cfg.Datasource.EastMoney.MaxRetries <= 0 {
		cfg.Datasource.EastMoney.MaxRetries = defaultEastMoneyMaxRetries
	}
	cfg.Datasource.EastMoney.UserAgentMode = strings.ToLower(strings.TrimSpace(cfg.Datasource.EastMoney.UserAgentMode))
	if cfg.Datasource.EastMoney.UserAgentMode == "" {
		cfg.Datasource.EastMoney.UserAgentMode = defaultEastMoneyUserAgentMode
	}
	cfg.Datasource.EastMoney.FetchMode = strings.ToLower(strings.TrimSpace(cfg.Datasource.EastMoney.FetchMode))
	if cfg.Datasource.EastMoney.FetchMode == "browser" {
		cfg.Datasource.EastMoney.FetchMode = "chromedp"
	}
	switch cfg.Datasource.EastMoney.FetchMode {
	case "", "http", "auto", "chromedp":
		if cfg.Datasource.EastMoney.FetchMode == "" {
			cfg.Datasource.EastMoney.FetchMode = defaultEastMoneyFetchMode
		}
	default:
		cfg.Datasource.EastMoney.FetchMode = defaultEastMoneyFetchMode
	}
	cfg.Datasource.EastMoney.BrowserPath = strings.TrimSpace(cfg.Datasource.EastMoney.BrowserPath)
	if cfg.Datasource.EastMoney.BrowserTimeoutSeconds <= 0 {
		cfg.Datasource.EastMoney.BrowserTimeoutSeconds = defaultEastMoneyBrowserTimeoutSeconds
	}
	if cfg.Datasource.EastMoney.BrowserCookieTTLSeconds <= 0 {
		cfg.Datasource.EastMoney.BrowserCookieTTLSeconds = defaultEastMoneyBrowserCookieTTLSeconds
	}
	cfg.Datasource.EastMoney.BrowserUserAgentMode = strings.ToLower(strings.TrimSpace(cfg.Datasource.EastMoney.BrowserUserAgentMode))
	if cfg.Datasource.EastMoney.BrowserUserAgentMode == "" {
		cfg.Datasource.EastMoney.BrowserUserAgentMode = defaultEastMoneyBrowserUserAgentMode
	}
	cfg.Datasource.EastMoney.BrowserUserAgentPlatform = strings.ToLower(strings.TrimSpace(cfg.Datasource.EastMoney.BrowserUserAgentPlatform))
	if cfg.Datasource.EastMoney.BrowserUserAgentPlatform == "" {
		cfg.Datasource.EastMoney.BrowserUserAgentPlatform = defaultEastMoneyBrowserUserAgentPlatform
	}
	if cfg.Datasource.EastMoney.BrowserCount <= 0 {
		cfg.Datasource.EastMoney.BrowserCount = defaultEastMoneyBrowserCount
	}
	if cfg.Datasource.EastMoney.BrowserTabsPerBrowser <= 0 {
		cfg.Datasource.EastMoney.BrowserTabsPerBrowser = cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs
	}
	if cfg.Datasource.EastMoney.BrowserTabsPerBrowser <= 0 {
		cfg.Datasource.EastMoney.BrowserTabsPerBrowser = defaultEastMoneyBrowserTabsPerBrowser
	}
	if cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs <= 0 {
		cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs = cfg.Datasource.EastMoney.BrowserTabsPerBrowser
	}
	if cfg.Datasource.EastMoney.BrowserRecycleAfterTabs <= 0 {
		cfg.Datasource.EastMoney.BrowserRecycleAfterTabs = defaultEastMoneyBrowserRecycleAfterTabs
	}
	cfg.Datasource.EastMoney.BrowserWaitReadySelector = strings.TrimSpace(cfg.Datasource.EastMoney.BrowserWaitReadySelector)
	if cfg.Datasource.EastMoney.BrowserWaitReadySelector == "" {
		cfg.Datasource.EastMoney.BrowserWaitReadySelector = defaultEastMoneyBrowserWaitReadySelector
	}
	cfg.Datasource.EastMoney.BrowserAcceptLanguage = strings.TrimSpace(cfg.Datasource.EastMoney.BrowserAcceptLanguage)
	if cfg.Datasource.EastMoney.BrowserAcceptLanguage == "" {
		cfg.Datasource.EastMoney.BrowserAcceptLanguage = defaultEastMoneyBrowserAcceptLanguage
	}
	if cfg.Datasource.EastMoney.BrowserWindowWidth <= 0 {
		cfg.Datasource.EastMoney.BrowserWindowWidth = defaultEastMoneyBrowserWindowWidth
	}
	if cfg.Datasource.EastMoney.BrowserWindowHeight <= 0 {
		cfg.Datasource.EastMoney.BrowserWindowHeight = defaultEastMoneyBrowserWindowHeight
	}
	cfg.Datasource.EastMoney.BrowserBlockedURLPatterns = compactStrings(cfg.Datasource.EastMoney.BrowserBlockedURLPatterns)
	cfg.Datasource.EastMoney.BrowserExtraFlags = compactStrings(cfg.Datasource.EastMoney.BrowserExtraFlags)
	if !cfg.Datasource.EastMoney.browserHeadlessConfigured {
		cfg.Datasource.EastMoney.BrowserHeadless = defaultEastMoneyBrowserHeadless
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
