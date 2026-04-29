package eastmoney

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
	"github.com/lifei6671/quantsage/apps/server/internal/infra/browserfetch"
)

const (
	defaultBrowserTimeout   = 60 * time.Second
	defaultBrowserCookieTTL = 720 * time.Second
)

// Config 定义东方财富导入源的最小构造参数。
type Config struct {
	Endpoint                  string
	QuoteEndpoint             string
	TimeoutSeconds            int
	MaxRetries                int
	UserAgentMode             string
	FetchMode                 FetchMode
	BrowserPath               string
	BrowserTimeoutSeconds     int
	BrowserCookieTTLSeconds   int
	BrowserHeadless           bool
	BrowserUserAgentMode      string
	BrowserUserAgentPlatform  string
	BrowserCount              int
	BrowserMaxConcurrentTabs  int
	BrowserTabsPerBrowser     int
	BrowserRecycleAfterTabs   int
	BrowserWaitReadySelector  string
	BrowserAcceptLanguage     string
	BrowserDisableImages      bool
	BrowserNoSandbox          bool
	BrowserWindowWidth        int
	BrowserWindowHeight       int
	BrowserBlockedURLPatterns []string
	BrowserExtraFlags         []string
}

// Source 通过东财公开行情接口提供导入层最小契约。
type Source struct {
	config         Config
	clientConfig   ClientConfig
	fallbackConfig FallbackConfig
	client         *Client
	fallbackClient *fallbackClient
	browser        browserRunner
}

var _ datasource.Source = (*Source)(nil)

// NewFromConfig 根据配置创建东财数据源。
func NewFromConfig(cfg Config) *Source {
	normalizedConfig := normalizeSourceConfig(cfg)
	clientConfig := buildClientConfig(normalizedConfig)
	client := NewClient(clientConfig)
	browser := browserfetch.New(buildBrowserFetchConfig(normalizedConfig))

	return newSourceWithDependencies(normalizedConfig, client, browser)
}

func newSourceWithClient(clientConfig ClientConfig, client *Client) *Source {
	return newSourceWithDependencies(sourceConfigFromClientConfig(clientConfig), client, nil)
}

func newSourceWithDependencies(sourceConfig Config, client *Client, browser browserRunner) *Source {
	normalizedConfig := normalizeSourceConfig(sourceConfig)
	clientConfig := buildClientConfig(normalizedConfig)
	if client == nil {
		client = NewClient(clientConfig)
	}
	fallbackConfig := buildFallbackConfig(normalizedConfig)
	fallback := newFallbackClient(client, browser, fallbackConfig)

	return &Source{
		config:         normalizedConfig,
		clientConfig:   clientConfig,
		fallbackConfig: fallbackConfig,
		client:         client,
		fallbackClient: fallback,
		browser:        browser,
	}
}

func sourceConfigFromClientConfig(cfg ClientConfig) Config {
	normalizedClientConfig := normalizeClientConfig(cfg)
	return Config{
		Endpoint:        normalizedClientConfig.Endpoint,
		QuoteEndpoint:   normalizedClientConfig.QuoteEndpoint,
		TimeoutSeconds:  int(normalizedClientConfig.Timeout / time.Second),
		MaxRetries:      normalizedClientConfig.MaxRetries,
		UserAgentMode:   normalizedClientConfig.UserAgentMode,
		FetchMode:       FetchModeHTTP,
		BrowserHeadless: true,
	}
}

func normalizeSourceConfig(cfg Config) Config {
	cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
	if cfg.Endpoint == "" {
		cfg.Endpoint = defaultEndpoint
	}
	cfg.QuoteEndpoint = strings.TrimSpace(cfg.QuoteEndpoint)
	if cfg.QuoteEndpoint == "" {
		cfg.QuoteEndpoint = defaultQuoteEndpoint
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = int(defaultTimeout / time.Second)
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	cfg.UserAgentMode = strings.ToLower(strings.TrimSpace(cfg.UserAgentMode))
	if cfg.UserAgentMode == "" {
		cfg.UserAgentMode = defaultUserAgentMode
	}
	cfg.FetchMode = normalizeFallbackConfig(FallbackConfig{Mode: cfg.FetchMode}).Mode
	cfg.BrowserPath = strings.TrimSpace(cfg.BrowserPath)
	if cfg.BrowserTimeoutSeconds <= 0 {
		cfg.BrowserTimeoutSeconds = int(defaultBrowserTimeout / time.Second)
	}
	if cfg.BrowserCookieTTLSeconds <= 0 {
		cfg.BrowserCookieTTLSeconds = int(defaultBrowserCookieTTL / time.Second)
	}
	cfg.BrowserUserAgentMode = normalizeBrowserUserAgentMode(cfg.BrowserUserAgentMode, cfg.UserAgentMode)
	cfg.BrowserUserAgentPlatform = normalizeBrowserUserAgentPlatform(cfg.BrowserUserAgentPlatform, cfg.BrowserUserAgentMode)
	if cfg.BrowserCount <= 0 {
		cfg.BrowserCount = 1
	}
	if cfg.BrowserTabsPerBrowser <= 0 {
		cfg.BrowserTabsPerBrowser = cfg.BrowserMaxConcurrentTabs
	}
	if cfg.BrowserTabsPerBrowser <= 0 {
		cfg.BrowserTabsPerBrowser = 4
	}
	if cfg.BrowserMaxConcurrentTabs <= 0 {
		cfg.BrowserMaxConcurrentTabs = cfg.BrowserTabsPerBrowser
	}
	if cfg.BrowserRecycleAfterTabs <= 0 {
		cfg.BrowserRecycleAfterTabs = 200
	}
	cfg.BrowserWaitReadySelector = strings.TrimSpace(cfg.BrowserWaitReadySelector)
	if cfg.BrowserWaitReadySelector == "" {
		cfg.BrowserWaitReadySelector = "body"
	}
	cfg.BrowserAcceptLanguage = strings.TrimSpace(cfg.BrowserAcceptLanguage)
	if cfg.BrowserAcceptLanguage == "" {
		cfg.BrowserAcceptLanguage = "zh-CN,zh;q=0.9,en;q=0.8"
	}
	if cfg.BrowserWindowWidth <= 0 {
		cfg.BrowserWindowWidth = 1366
	}
	if cfg.BrowserWindowHeight <= 0 {
		cfg.BrowserWindowHeight = 768
	}
	cfg.BrowserBlockedURLPatterns = compactStringValues(cfg.BrowserBlockedURLPatterns)
	cfg.BrowserExtraFlags = compactStringValues(cfg.BrowserExtraFlags)

	return cfg
}

func normalizeBrowserUserAgentMode(mode string, fallbackMode string) string {
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	switch normalizedMode {
	case "", "stable", "mobile", "default", "custom":
		if normalizedMode == "" {
			return fallbackMode
		}
		return normalizedMode
	default:
		return fallbackMode
	}
}

func normalizeBrowserUserAgentPlatform(platform string, browserMode string) string {
	normalizedPlatform := strings.ToLower(strings.TrimSpace(platform))
	switch normalizedPlatform {
	case browserfetch.UserAgentPlatformMobile:
		return browserfetch.UserAgentPlatformMobile
	case browserfetch.UserAgentPlatformDesktop:
		return browserfetch.UserAgentPlatformDesktop
	}
	if normalizeBrowserUserAgentMode(browserMode, defaultUserAgentMode) == "mobile" {
		return browserfetch.UserAgentPlatformMobile
	}

	return browserfetch.UserAgentPlatformDesktop
}

func buildClientConfig(cfg Config) ClientConfig {
	cfg = normalizeSourceConfig(cfg)
	return normalizeClientConfig(ClientConfig{
		Endpoint:      cfg.Endpoint,
		QuoteEndpoint: cfg.QuoteEndpoint,
		Timeout:       time.Duration(cfg.TimeoutSeconds) * time.Second,
		MaxRetries:    cfg.MaxRetries,
		UserAgentMode: cfg.UserAgentMode,
	})
}

func buildFallbackConfig(cfg Config) FallbackConfig {
	cfg = normalizeSourceConfig(cfg)
	return normalizeFallbackConfig(FallbackConfig{
		Mode:         cfg.FetchMode,
		QuotePageURL: deriveQuotePageURL(cfg.QuoteEndpoint),
	})
}

func buildBrowserFetchConfig(cfg Config) browserfetch.Config {
	cfg = normalizeSourceConfig(cfg)
	mode, userAgent := mapBrowserUserAgentStrategy(cfg.BrowserUserAgentMode, cfg.UserAgentMode)
	headless := cfg.BrowserHeadless

	return browserfetch.Config{
		BrowserPath:        cfg.BrowserPath,
		Headless:           &headless,
		UserAgentMode:      mode,
		UserAgent:          userAgent,
		UserAgentPlatform:  cfg.BrowserUserAgentPlatform,
		AcceptLanguage:     cfg.BrowserAcceptLanguage,
		Timeout:            time.Duration(cfg.BrowserTimeoutSeconds) * time.Second,
		CookieCacheTTL:     time.Duration(cfg.BrowserCookieTTLSeconds) * time.Second,
		BrowserCount:       cfg.BrowserCount,
		TabsPerBrowser:     cfg.BrowserTabsPerBrowser,
		RecycleAfterTabs:   cfg.BrowserRecycleAfterTabs,
		MaxConcurrentTabs:  cfg.BrowserMaxConcurrentTabs,
		WaitReadySelector:  cfg.BrowserWaitReadySelector,
		DisableImages:      cfg.BrowserDisableImages,
		BlockedURLPatterns: append([]string(nil), cfg.BrowserBlockedURLPatterns...),
		NoSandbox:          cfg.BrowserNoSandbox,
		WindowWidth:        cfg.BrowserWindowWidth,
		WindowHeight:       cfg.BrowserWindowHeight,
		ExtraFlags:         append([]string(nil), cfg.BrowserExtraFlags...),
	}
}

func mapBrowserUserAgentStrategy(browserMode string, sourceUserAgentMode string) (string, string) {
	switch normalizeBrowserUserAgentMode(browserMode, sourceUserAgentMode) {
	case "default":
		return browserfetch.UserAgentModeDefault, ""
	case "mobile":
		return browserfetch.UserAgentModeFake, userAgentValue("mobile")
	case "custom":
		return browserfetch.UserAgentModeFake, userAgentValue(sourceUserAgentMode)
	default:
		return browserfetch.UserAgentModeFake, userAgentValue("stable")
	}
}

func compactStringValues(items []string) []string {
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

func (s *Source) ensureReady() error {
	if s == nil || s.client == nil || s.fallbackClient == nil {
		return datasourceUnavailable(errors.New("eastmoney datasource client is nil"))
	}

	return nil
}

// Close 释放东财数据源持有的浏览器抓取资源。
func (s *Source) Close(ctx context.Context) error {
	if s == nil || s.browser == nil {
		return nil
	}
	closeable, ok := s.browser.(interface {
		Close(context.Context) error
	})
	if !ok {
		return nil
	}
	if err := closeable.Close(ctx); err != nil {
		return datasourceUnavailable(fmt.Errorf("close eastmoney browser runner: %w", err))
	}

	return nil
}

func deriveQuotePageURL(rawQuoteEndpoint string) string {
	baseURL, err := url.Parse(strings.TrimSpace(rawQuoteEndpoint))
	if err != nil || baseURL == nil {
		return "https://quote.eastmoney.com/concept/sh000001.html"
	}

	scheme := strings.TrimSpace(baseURL.Scheme)
	if scheme == "" {
		scheme = "https"
	}

	host := strings.TrimSpace(baseURL.Host)
	if host == "" {
		host = "quote.eastmoney.com"
	}
	if strings.Contains(host, "eastmoney.com") {
		host = "quote.eastmoney.com"
	}

	return (&url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   "/concept/sh000001.html",
	}).String()
}
