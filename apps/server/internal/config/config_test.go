package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/consts"
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
datasource:
  default_source: eastmoney
  tushare:
    token: " file-token "
  eastmoney:
    endpoint: " https://hist.example.com "
    quote_endpoint: " https://quote.example.com "
    timeout_seconds: 45
    max_retries: 4
    user_agent_mode: " MOBILE "
    fetch_mode: " Chromedp "
    browser_path: " /opt/google/chrome "
    browser_timeout_seconds: 88
    browser_cookie_ttl_seconds: 1440
    browser_headless: false
    browser_user_agent_mode: " Desktop "
    browser_user_agent_platform: " Mobile "
    browser_count: 2
    browser_max_concurrent_tabs: 3
    browser_tabs_per_browser: 5
    browser_recycle_after_tabs: 20
    browser_wait_ready_selector: "#app"
    browser_accept_language: "zh-CN,zh;q=0.8"
    browser_disable_images: true
    browser_no_sandbox: true
    browser_window_width: 1280
    browser_window_height: 720
    browser_blocked_url_patterns:
      - " *://*/*.png "
    browser_extra_flags:
      - " disable-blink-features=AutomationControlled "
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
	if cfg.Datasource.Tushare.Token != "file-token" {
		t.Fatalf("cfg.Datasource.Tushare.Token = %q, want %q", cfg.Datasource.Tushare.Token, "file-token")
	}
	if cfg.Datasource.DefaultSource != consts.DatasourceEastMoney {
		t.Fatalf("cfg.Datasource.DefaultSource = %q, want %q", cfg.Datasource.DefaultSource, consts.DatasourceEastMoney)
	}
	if cfg.Datasource.EastMoney.Endpoint != "https://hist.example.com" {
		t.Fatalf("cfg.Datasource.EastMoney.Endpoint = %q, want %q", cfg.Datasource.EastMoney.Endpoint, "https://hist.example.com")
	}
	if cfg.Datasource.EastMoney.QuoteEndpoint != "https://quote.example.com" {
		t.Fatalf("cfg.Datasource.EastMoney.QuoteEndpoint = %q, want %q", cfg.Datasource.EastMoney.QuoteEndpoint, "https://quote.example.com")
	}
	if cfg.Datasource.EastMoney.TimeoutSeconds != 45 {
		t.Fatalf("cfg.Datasource.EastMoney.TimeoutSeconds = %d, want %d", cfg.Datasource.EastMoney.TimeoutSeconds, 45)
	}
	if cfg.Datasource.EastMoney.MaxRetries != 4 {
		t.Fatalf("cfg.Datasource.EastMoney.MaxRetries = %d, want %d", cfg.Datasource.EastMoney.MaxRetries, 4)
	}
	if cfg.Datasource.EastMoney.UserAgentMode != "mobile" {
		t.Fatalf("cfg.Datasource.EastMoney.UserAgentMode = %q, want %q", cfg.Datasource.EastMoney.UserAgentMode, "mobile")
	}
	if cfg.Datasource.EastMoney.FetchMode != "chromedp" {
		t.Fatalf("cfg.Datasource.EastMoney.FetchMode = %q, want %q", cfg.Datasource.EastMoney.FetchMode, "chromedp")
	}
	if cfg.Datasource.EastMoney.BrowserPath != "/opt/google/chrome" {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserPath = %q, want %q", cfg.Datasource.EastMoney.BrowserPath, "/opt/google/chrome")
	}
	if cfg.Datasource.EastMoney.BrowserTimeoutSeconds != 88 {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserTimeoutSeconds = %d, want %d", cfg.Datasource.EastMoney.BrowserTimeoutSeconds, 88)
	}
	if cfg.Datasource.EastMoney.BrowserCookieTTLSeconds != 1440 {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserCookieTTLSeconds = %d, want %d", cfg.Datasource.EastMoney.BrowserCookieTTLSeconds, 1440)
	}
	if cfg.Datasource.EastMoney.BrowserHeadless {
		t.Fatal("cfg.Datasource.EastMoney.BrowserHeadless = true, want false")
	}
	if cfg.Datasource.EastMoney.BrowserUserAgentMode != "desktop" {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserUserAgentMode = %q, want %q", cfg.Datasource.EastMoney.BrowserUserAgentMode, "desktop")
	}
	if cfg.Datasource.EastMoney.BrowserUserAgentPlatform != "mobile" {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserUserAgentPlatform = %q, want %q", cfg.Datasource.EastMoney.BrowserUserAgentPlatform, "mobile")
	}
	if cfg.Datasource.EastMoney.BrowserCount != 2 {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserCount = %d, want %d", cfg.Datasource.EastMoney.BrowserCount, 2)
	}
	if cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs != 3 {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs = %d, want %d", cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs, 3)
	}
	if cfg.Datasource.EastMoney.BrowserTabsPerBrowser != 5 {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserTabsPerBrowser = %d, want %d", cfg.Datasource.EastMoney.BrowserTabsPerBrowser, 5)
	}
	if cfg.Datasource.EastMoney.BrowserRecycleAfterTabs != 20 {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserRecycleAfterTabs = %d, want %d", cfg.Datasource.EastMoney.BrowserRecycleAfterTabs, 20)
	}
	if cfg.Datasource.EastMoney.BrowserWaitReadySelector != "#app" {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserWaitReadySelector = %q, want %q", cfg.Datasource.EastMoney.BrowserWaitReadySelector, "#app")
	}
	if cfg.Datasource.EastMoney.BrowserAcceptLanguage != "zh-CN,zh;q=0.8" {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserAcceptLanguage = %q, want configured value", cfg.Datasource.EastMoney.BrowserAcceptLanguage)
	}
	if !cfg.Datasource.EastMoney.BrowserDisableImages {
		t.Fatal("cfg.Datasource.EastMoney.BrowserDisableImages = false, want true")
	}
	if !cfg.Datasource.EastMoney.BrowserNoSandbox {
		t.Fatal("cfg.Datasource.EastMoney.BrowserNoSandbox = false, want true")
	}
	if cfg.Datasource.EastMoney.BrowserWindowWidth != 1280 || cfg.Datasource.EastMoney.BrowserWindowHeight != 720 {
		t.Fatalf("browser window = %dx%d, want 1280x720", cfg.Datasource.EastMoney.BrowserWindowWidth, cfg.Datasource.EastMoney.BrowserWindowHeight)
	}
	if len(cfg.Datasource.EastMoney.BrowserBlockedURLPatterns) != 1 || cfg.Datasource.EastMoney.BrowserBlockedURLPatterns[0] != "*://*/*.png" {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserBlockedURLPatterns = %v, want trimmed pattern", cfg.Datasource.EastMoney.BrowserBlockedURLPatterns)
	}
	if len(cfg.Datasource.EastMoney.BrowserExtraFlags) != 1 || cfg.Datasource.EastMoney.BrowserExtraFlags[0] != "disable-blink-features=AutomationControlled" {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserExtraFlags = %v, want trimmed flag", cfg.Datasource.EastMoney.BrowserExtraFlags)
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
	t.Setenv("QUANTSAGE_DATASOURCE_DEFAULT_SOURCE", consts.DatasourceEastMoney)
	t.Setenv("QUANTSAGE_TUSHARE_TOKEN", "from-env-token")
	t.Setenv("QUANTSAGE_EASTMONEY_ENDPOINT", "https://env-hist.example.com")
	t.Setenv("QUANTSAGE_EASTMONEY_QUOTE_ENDPOINT", "https://env-quote.example.com")
	t.Setenv("QUANTSAGE_EASTMONEY_TIMEOUT_SECONDS", "61")
	t.Setenv("QUANTSAGE_EASTMONEY_MAX_RETRIES", "5")
	t.Setenv("QUANTSAGE_EASTMONEY_USER_AGENT_MODE", "desktop")
	t.Setenv("QUANTSAGE_EASTMONEY_FETCH_MODE", " chromedp ")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_PATH", " /Applications/Google Chrome.app/Contents/MacOS/Google Chrome ")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_TIMEOUT_SECONDS", "91")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_COOKIE_TTL_SECONDS", "1500")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_HEADLESS", "false")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_USER_AGENT_MODE", " mobile ")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_USER_AGENT_PLATFORM", " mobile ")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_COUNT", "4")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_MAX_CONCURRENT_TABS", "6")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_TABS_PER_BROWSER", "3")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_RECYCLE_AFTER_TABS", "30")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_WAIT_READY_SELECTOR", "#root")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_ACCEPT_LANGUAGE", "zh-CN")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_DISABLE_IMAGES", "true")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_NO_SANDBOX", "true")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_WINDOW_WIDTH", "1440")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_WINDOW_HEIGHT", "900")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_BLOCKED_URL_PATTERNS", "*://*/*.png, *://*/*.jpg ")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_EXTRA_FLAGS", "disable-dev-shm-usage, disable-blink-features=AutomationControlled")

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
	if cfg.Datasource.Tushare.Token != "from-env-token" {
		t.Fatalf("cfg.Datasource.Tushare.Token = %q, want %q", cfg.Datasource.Tushare.Token, "from-env-token")
	}
	if cfg.Datasource.DefaultSource != consts.DatasourceEastMoney {
		t.Fatalf("cfg.Datasource.DefaultSource = %q, want %q", cfg.Datasource.DefaultSource, consts.DatasourceEastMoney)
	}
	if cfg.Datasource.EastMoney.Endpoint != "https://env-hist.example.com" {
		t.Fatalf("cfg.Datasource.EastMoney.Endpoint = %q, want %q", cfg.Datasource.EastMoney.Endpoint, "https://env-hist.example.com")
	}
	if cfg.Datasource.EastMoney.QuoteEndpoint != "https://env-quote.example.com" {
		t.Fatalf("cfg.Datasource.EastMoney.QuoteEndpoint = %q, want %q", cfg.Datasource.EastMoney.QuoteEndpoint, "https://env-quote.example.com")
	}
	if cfg.Datasource.EastMoney.TimeoutSeconds != 61 {
		t.Fatalf("cfg.Datasource.EastMoney.TimeoutSeconds = %d, want %d", cfg.Datasource.EastMoney.TimeoutSeconds, 61)
	}
	if cfg.Datasource.EastMoney.MaxRetries != 5 {
		t.Fatalf("cfg.Datasource.EastMoney.MaxRetries = %d, want %d", cfg.Datasource.EastMoney.MaxRetries, 5)
	}
	if cfg.Datasource.EastMoney.UserAgentMode != "desktop" {
		t.Fatalf("cfg.Datasource.EastMoney.UserAgentMode = %q, want %q", cfg.Datasource.EastMoney.UserAgentMode, "desktop")
	}
	if cfg.Datasource.EastMoney.FetchMode != "chromedp" {
		t.Fatalf("cfg.Datasource.EastMoney.FetchMode = %q, want %q", cfg.Datasource.EastMoney.FetchMode, "chromedp")
	}
	if cfg.Datasource.EastMoney.BrowserPath != "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserPath = %q, want %q", cfg.Datasource.EastMoney.BrowserPath, "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome")
	}
	if cfg.Datasource.EastMoney.BrowserTimeoutSeconds != 91 {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserTimeoutSeconds = %d, want %d", cfg.Datasource.EastMoney.BrowserTimeoutSeconds, 91)
	}
	if cfg.Datasource.EastMoney.BrowserCookieTTLSeconds != 1500 {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserCookieTTLSeconds = %d, want %d", cfg.Datasource.EastMoney.BrowserCookieTTLSeconds, 1500)
	}
	if cfg.Datasource.EastMoney.BrowserHeadless {
		t.Fatal("cfg.Datasource.EastMoney.BrowserHeadless = true, want false")
	}
	if cfg.Datasource.EastMoney.BrowserUserAgentMode != "mobile" {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserUserAgentMode = %q, want %q", cfg.Datasource.EastMoney.BrowserUserAgentMode, "mobile")
	}
	if cfg.Datasource.EastMoney.BrowserUserAgentPlatform != "mobile" {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserUserAgentPlatform = %q, want %q", cfg.Datasource.EastMoney.BrowserUserAgentPlatform, "mobile")
	}
	if cfg.Datasource.EastMoney.BrowserCount != 4 {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserCount = %d, want %d", cfg.Datasource.EastMoney.BrowserCount, 4)
	}
	if cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs != 6 {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs = %d, want %d", cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs, 6)
	}
	if cfg.Datasource.EastMoney.BrowserTabsPerBrowser != 3 {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserTabsPerBrowser = %d, want %d", cfg.Datasource.EastMoney.BrowserTabsPerBrowser, 3)
	}
	if cfg.Datasource.EastMoney.BrowserRecycleAfterTabs != 30 {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserRecycleAfterTabs = %d, want %d", cfg.Datasource.EastMoney.BrowserRecycleAfterTabs, 30)
	}
	if cfg.Datasource.EastMoney.BrowserWaitReadySelector != "#root" {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserWaitReadySelector = %q, want %q", cfg.Datasource.EastMoney.BrowserWaitReadySelector, "#root")
	}
	if cfg.Datasource.EastMoney.BrowserAcceptLanguage != "zh-CN" {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserAcceptLanguage = %q, want %q", cfg.Datasource.EastMoney.BrowserAcceptLanguage, "zh-CN")
	}
	if !cfg.Datasource.EastMoney.BrowserDisableImages {
		t.Fatal("cfg.Datasource.EastMoney.BrowserDisableImages = false, want true")
	}
	if !cfg.Datasource.EastMoney.BrowserNoSandbox {
		t.Fatal("cfg.Datasource.EastMoney.BrowserNoSandbox = false, want true")
	}
	if cfg.Datasource.EastMoney.BrowserWindowWidth != 1440 || cfg.Datasource.EastMoney.BrowserWindowHeight != 900 {
		t.Fatalf("browser window = %dx%d, want 1440x900", cfg.Datasource.EastMoney.BrowserWindowWidth, cfg.Datasource.EastMoney.BrowserWindowHeight)
	}
	if len(cfg.Datasource.EastMoney.BrowserBlockedURLPatterns) != 2 {
		t.Fatalf("BrowserBlockedURLPatterns = %v, want 2 patterns", cfg.Datasource.EastMoney.BrowserBlockedURLPatterns)
	}
	if len(cfg.Datasource.EastMoney.BrowserExtraFlags) != 2 {
		t.Fatalf("BrowserExtraFlags = %v, want 2 flags", cfg.Datasource.EastMoney.BrowserExtraFlags)
	}
}

func TestLoadAppliesDatasourceDefaults(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
app:
  name: quantsage
database:
  dsn: postgres://demo
redis:
  addr: 127.0.0.1:6379
auth:
  session_secret: "***"
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Datasource.DefaultSource != defaultDatasourceSource {
		t.Fatalf("cfg.Datasource.DefaultSource = %q, want %q", cfg.Datasource.DefaultSource, defaultDatasourceSource)
	}
	if cfg.Datasource.EastMoney.Endpoint != defaultEastMoneyEndpoint {
		t.Fatalf("cfg.Datasource.EastMoney.Endpoint = %q, want %q", cfg.Datasource.EastMoney.Endpoint, defaultEastMoneyEndpoint)
	}
	if cfg.Datasource.EastMoney.QuoteEndpoint != defaultEastMoneyQuoteEndpoint {
		t.Fatalf("cfg.Datasource.EastMoney.QuoteEndpoint = %q, want %q", cfg.Datasource.EastMoney.QuoteEndpoint, defaultEastMoneyQuoteEndpoint)
	}
	if cfg.Datasource.EastMoney.TimeoutSeconds != defaultEastMoneyTimeoutSeconds {
		t.Fatalf("cfg.Datasource.EastMoney.TimeoutSeconds = %d, want %d", cfg.Datasource.EastMoney.TimeoutSeconds, defaultEastMoneyTimeoutSeconds)
	}
	if cfg.Datasource.EastMoney.MaxRetries != defaultEastMoneyMaxRetries {
		t.Fatalf("cfg.Datasource.EastMoney.MaxRetries = %d, want %d", cfg.Datasource.EastMoney.MaxRetries, defaultEastMoneyMaxRetries)
	}
	if cfg.Datasource.EastMoney.UserAgentMode != defaultEastMoneyUserAgentMode {
		t.Fatalf("cfg.Datasource.EastMoney.UserAgentMode = %q, want %q", cfg.Datasource.EastMoney.UserAgentMode, defaultEastMoneyUserAgentMode)
	}
	if cfg.Datasource.EastMoney.FetchMode != defaultEastMoneyFetchMode {
		t.Fatalf("cfg.Datasource.EastMoney.FetchMode = %q, want %q", cfg.Datasource.EastMoney.FetchMode, defaultEastMoneyFetchMode)
	}
	if cfg.Datasource.EastMoney.BrowserTimeoutSeconds != defaultEastMoneyBrowserTimeoutSeconds {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserTimeoutSeconds = %d, want %d", cfg.Datasource.EastMoney.BrowserTimeoutSeconds, defaultEastMoneyBrowserTimeoutSeconds)
	}
	if cfg.Datasource.EastMoney.BrowserCookieTTLSeconds != defaultEastMoneyBrowserCookieTTLSeconds {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserCookieTTLSeconds = %d, want %d", cfg.Datasource.EastMoney.BrowserCookieTTLSeconds, defaultEastMoneyBrowserCookieTTLSeconds)
	}
	if !cfg.Datasource.EastMoney.BrowserHeadless {
		t.Fatal("cfg.Datasource.EastMoney.BrowserHeadless = false, want true")
	}
	if cfg.Datasource.EastMoney.BrowserUserAgentMode != defaultEastMoneyBrowserUserAgentMode {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserUserAgentMode = %q, want %q", cfg.Datasource.EastMoney.BrowserUserAgentMode, defaultEastMoneyBrowserUserAgentMode)
	}
	if cfg.Datasource.EastMoney.BrowserUserAgentPlatform != defaultEastMoneyBrowserUserAgentPlatform {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserUserAgentPlatform = %q, want %q", cfg.Datasource.EastMoney.BrowserUserAgentPlatform, defaultEastMoneyBrowserUserAgentPlatform)
	}
	if cfg.Datasource.EastMoney.BrowserCount != defaultEastMoneyBrowserCount {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserCount = %d, want %d", cfg.Datasource.EastMoney.BrowserCount, defaultEastMoneyBrowserCount)
	}
	if cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs != defaultEastMoneyBrowserMaxConcurrentTabs {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs = %d, want %d", cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs, defaultEastMoneyBrowserMaxConcurrentTabs)
	}
	if cfg.Datasource.EastMoney.BrowserTabsPerBrowser != defaultEastMoneyBrowserTabsPerBrowser {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserTabsPerBrowser = %d, want %d", cfg.Datasource.EastMoney.BrowserTabsPerBrowser, defaultEastMoneyBrowserTabsPerBrowser)
	}
	if cfg.Datasource.EastMoney.BrowserRecycleAfterTabs != defaultEastMoneyBrowserRecycleAfterTabs {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserRecycleAfterTabs = %d, want %d", cfg.Datasource.EastMoney.BrowserRecycleAfterTabs, defaultEastMoneyBrowserRecycleAfterTabs)
	}
	if cfg.Datasource.EastMoney.BrowserWaitReadySelector != defaultEastMoneyBrowserWaitReadySelector {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserWaitReadySelector = %q, want %q", cfg.Datasource.EastMoney.BrowserWaitReadySelector, defaultEastMoneyBrowserWaitReadySelector)
	}
	if cfg.Datasource.EastMoney.BrowserAcceptLanguage != defaultEastMoneyBrowserAcceptLanguage {
		t.Fatalf("cfg.Datasource.EastMoney.BrowserAcceptLanguage = %q, want %q", cfg.Datasource.EastMoney.BrowserAcceptLanguage, defaultEastMoneyBrowserAcceptLanguage)
	}
	if cfg.Datasource.EastMoney.BrowserWindowWidth != defaultEastMoneyBrowserWindowWidth || cfg.Datasource.EastMoney.BrowserWindowHeight != defaultEastMoneyBrowserWindowHeight {
		t.Fatalf("browser window = %dx%d, want default", cfg.Datasource.EastMoney.BrowserWindowWidth, cfg.Datasource.EastMoney.BrowserWindowHeight)
	}
}

func TestLoadFallsBackInvalidFetchModeToAuto(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
app:
  name: quantsage
database:
  dsn: postgres://demo
redis:
  addr: 127.0.0.1:6379
datasource:
  eastmoney:
    fetch_mode: invalid-value
auth:
  session_secret: "***"
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Datasource.EastMoney.FetchMode != defaultEastMoneyFetchMode {
		t.Fatalf("cfg.Datasource.EastMoney.FetchMode = %q, want %q", cfg.Datasource.EastMoney.FetchMode, defaultEastMoneyFetchMode)
	}
}
