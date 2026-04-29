package app

import (
	"net/http"
	"testing"

	"github.com/lifei6671/quantsage/apps/server/internal/config"
	eastmoneyds "github.com/lifei6671/quantsage/apps/server/internal/domain/datasource/eastmoney"
	tushareds "github.com/lifei6671/quantsage/apps/server/internal/domain/datasource/tushare"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/consts"
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

func TestBuildImportSourceUsesEastMoneyWhenConfigured(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Datasource.DefaultSource = consts.DatasourceEastMoney
	cfg.Datasource.EastMoney.Endpoint = "https://push2his.eastmoney.com"
	cfg.Datasource.EastMoney.QuoteEndpoint = "https://push2his.eastmoney.com"
	cfg.Datasource.EastMoney.TimeoutSeconds = 30
	cfg.Datasource.EastMoney.MaxRetries = 2
	cfg.Datasource.EastMoney.UserAgentMode = "stable"

	source := buildImportSource(cfg)
	if source == nil {
		t.Fatal("buildImportSource() = nil, want eastmoney source")
	}
	if _, ok := source.(*eastmoneyds.Source); !ok {
		t.Fatalf("buildImportSource() type = %T, want *eastmoney.Source", source)
	}
}

func TestBuildEastMoneyDatasourceConfigMapsBrowserFallbackFields(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Datasource.EastMoney.Endpoint = "https://push2his.eastmoney.com"
	cfg.Datasource.EastMoney.QuoteEndpoint = "https://quote.eastmoney.com"
	cfg.Datasource.EastMoney.TimeoutSeconds = 45
	cfg.Datasource.EastMoney.MaxRetries = 3
	cfg.Datasource.EastMoney.UserAgentMode = "mobile"
	cfg.Datasource.EastMoney.FetchMode = "chromedp"
	cfg.Datasource.EastMoney.BrowserPath = "/usr/bin/chromium"
	cfg.Datasource.EastMoney.BrowserTimeoutSeconds = 70
	cfg.Datasource.EastMoney.BrowserCookieTTLSeconds = 900
	cfg.Datasource.EastMoney.BrowserHeadless = false
	cfg.Datasource.EastMoney.BrowserUserAgentMode = "custom"
	cfg.Datasource.EastMoney.BrowserUserAgentPlatform = "mobile"
	cfg.Datasource.EastMoney.BrowserCount = 4
	cfg.Datasource.EastMoney.BrowserMaxConcurrentTabs = 6
	cfg.Datasource.EastMoney.BrowserTabsPerBrowser = 3
	cfg.Datasource.EastMoney.BrowserRecycleAfterTabs = 30
	cfg.Datasource.EastMoney.BrowserWaitReadySelector = "#app"
	cfg.Datasource.EastMoney.BrowserAcceptLanguage = "zh-CN"
	cfg.Datasource.EastMoney.BrowserDisableImages = true
	cfg.Datasource.EastMoney.BrowserNoSandbox = true
	cfg.Datasource.EastMoney.BrowserWindowWidth = 1440
	cfg.Datasource.EastMoney.BrowserWindowHeight = 900
	cfg.Datasource.EastMoney.BrowserBlockedURLPatterns = []string{"*://*/*.png"}
	cfg.Datasource.EastMoney.BrowserExtraFlags = []string{"disable-dev-shm-usage"}

	got := buildEastMoneyDatasourceConfig(cfg)
	if got.Endpoint != "https://push2his.eastmoney.com" {
		t.Fatalf("Endpoint = %q, want %q", got.Endpoint, "https://push2his.eastmoney.com")
	}
	if got.QuoteEndpoint != "https://quote.eastmoney.com" {
		t.Fatalf("QuoteEndpoint = %q, want %q", got.QuoteEndpoint, "https://quote.eastmoney.com")
	}
	if got.TimeoutSeconds != 45 || got.MaxRetries != 3 {
		t.Fatalf("timeouts/retries = %+v, want timeout=45 maxRetries=3", got)
	}
	if got.UserAgentMode != "mobile" {
		t.Fatalf("UserAgentMode = %q, want %q", got.UserAgentMode, "mobile")
	}
	if got.FetchMode != eastmoneyds.FetchModeChromedp {
		t.Fatalf("FetchMode = %q, want %q", got.FetchMode, eastmoneyds.FetchModeChromedp)
	}
	if got.BrowserPath != "/usr/bin/chromium" {
		t.Fatalf("BrowserPath = %q, want %q", got.BrowserPath, "/usr/bin/chromium")
	}
	if got.BrowserTimeoutSeconds != 70 {
		t.Fatalf("BrowserTimeoutSeconds = %d, want %d", got.BrowserTimeoutSeconds, 70)
	}
	if got.BrowserCookieTTLSeconds != 900 {
		t.Fatalf("BrowserCookieTTLSeconds = %d, want %d", got.BrowserCookieTTLSeconds, 900)
	}
	if got.BrowserHeadless {
		t.Fatal("BrowserHeadless = true, want false")
	}
	if got.BrowserUserAgentMode != "custom" {
		t.Fatalf("BrowserUserAgentMode = %q, want %q", got.BrowserUserAgentMode, "custom")
	}
	if got.BrowserUserAgentPlatform != "mobile" {
		t.Fatalf("BrowserUserAgentPlatform = %q, want %q", got.BrowserUserAgentPlatform, "mobile")
	}
	if got.BrowserCount != 4 {
		t.Fatalf("BrowserCount = %d, want %d", got.BrowserCount, 4)
	}
	if got.BrowserMaxConcurrentTabs != 6 {
		t.Fatalf("BrowserMaxConcurrentTabs = %d, want %d", got.BrowserMaxConcurrentTabs, 6)
	}
	if got.BrowserTabsPerBrowser != 3 {
		t.Fatalf("BrowserTabsPerBrowser = %d, want %d", got.BrowserTabsPerBrowser, 3)
	}
	if got.BrowserRecycleAfterTabs != 30 {
		t.Fatalf("BrowserRecycleAfterTabs = %d, want %d", got.BrowserRecycleAfterTabs, 30)
	}
	if got.BrowserWaitReadySelector != "#app" {
		t.Fatalf("BrowserWaitReadySelector = %q, want %q", got.BrowserWaitReadySelector, "#app")
	}
	if got.BrowserAcceptLanguage != "zh-CN" {
		t.Fatalf("BrowserAcceptLanguage = %q, want %q", got.BrowserAcceptLanguage, "zh-CN")
	}
	if !got.BrowserDisableImages || !got.BrowserNoSandbox {
		t.Fatalf("browser flags = disableImages:%v noSandbox:%v, want true/true", got.BrowserDisableImages, got.BrowserNoSandbox)
	}
	if got.BrowserWindowWidth != 1440 || got.BrowserWindowHeight != 900 {
		t.Fatalf("browser window = %dx%d, want 1440x900", got.BrowserWindowWidth, got.BrowserWindowHeight)
	}
	if len(got.BrowserBlockedURLPatterns) != 1 || got.BrowserBlockedURLPatterns[0] != "*://*/*.png" {
		t.Fatalf("BrowserBlockedURLPatterns = %v, want configured value", got.BrowserBlockedURLPatterns)
	}
	if len(got.BrowserExtraFlags) != 1 || got.BrowserExtraFlags[0] != "disable-dev-shm-usage" {
		t.Fatalf("BrowserExtraFlags = %v, want configured value", got.BrowserExtraFlags)
	}
}

func TestBuildEastMoneyDatasourceConfigPreservesStableBrowserUserAgentMode(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Datasource.EastMoney.FetchMode = "auto"
	cfg.Datasource.EastMoney.UserAgentMode = "mobile"
	cfg.Datasource.EastMoney.BrowserUserAgentMode = "stable"

	got := buildEastMoneyDatasourceConfig(cfg)
	if got.FetchMode != eastmoneyds.FetchModeAuto {
		t.Fatalf("FetchMode = %q, want %q", got.FetchMode, eastmoneyds.FetchModeAuto)
	}
	if got.BrowserUserAgentMode != "stable" {
		t.Fatalf("BrowserUserAgentMode = %q, want %q", got.BrowserUserAgentMode, "stable")
	}
	if got.UserAgentMode != "mobile" {
		t.Fatalf("UserAgentMode = %q, want %q", got.UserAgentMode, "mobile")
	}
}

func TestBuildImportSourceFallsBackToTushareWhenDefaultSourceEmpty(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Datasource.Tushare.Token = "demo-token"

	source := buildImportSource(cfg)
	if source == nil {
		t.Fatal("buildImportSource() = nil, want tushare source")
	}
	if _, ok := source.(*tushareds.Source); !ok {
		t.Fatalf("buildImportSource() type = %T, want *tushare.Source", source)
	}
}

func TestBuildImportSourceReturnsTushareWhenDefaultSourceAndTokenEmpty(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}

	source := buildImportSource(cfg)
	if source == nil {
		t.Fatal("buildImportSource() = nil, want tushare source")
	}
	if _, ok := source.(*tushareds.Source); !ok {
		t.Fatalf("buildImportSource() type = %T, want *tushare.Source", source)
	}
}

func TestBuildImportSourceReturnsNilForUnknownSource(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Datasource.DefaultSource = "unknown"
	cfg.Datasource.Tushare.Token = "demo-token"

	if source := buildImportSource(cfg); source != nil {
		t.Fatalf("buildImportSource() = %T, want nil", source)
	}
}

func TestBuildLocalRuntimeImportSourceReturnsNilForEmptyTushareToken(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}

	source, err := buildLocalRuntimeImportSource(cfg)
	if err != nil {
		t.Fatalf("buildLocalRuntimeImportSource() error = %v", err)
	}
	if source != nil {
		t.Fatalf("buildLocalRuntimeImportSource() = %T, want nil", source)
	}
}

func TestBuildLocalRuntimeImportSourceReturnsErrorForUnknownSource(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Datasource.DefaultSource = "unknown"

	source, err := buildLocalRuntimeImportSource(cfg)
	if err == nil {
		t.Fatal("buildLocalRuntimeImportSource() error = nil, want non-nil")
	}
	if source != nil {
		t.Fatalf("buildLocalRuntimeImportSource() = %T, want nil", source)
	}
}
