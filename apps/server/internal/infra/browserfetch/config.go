package browserfetch

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// UserAgentModeDefault 表示使用浏览器自身的默认 User-Agent。
	UserAgentModeDefault = "default"
	// UserAgentModeCustom 表示显式使用 Config.UserAgent。
	UserAgentModeCustom = "custom"
	// UserAgentModeFake 表示优先使用 fake-useragent 生成浏览器 User-Agent。
	UserAgentModeFake = "fake"

	UserAgentPlatformDesktop = "desktop"
	UserAgentPlatformMobile  = "mobile"

	defaultBrowserTimeout      = 60 * time.Second
	defaultBrowserCloseTimeout = 10 * time.Second
	defaultCookieCacheTTL      = 720 * time.Second
	defaultHeadless            = true
	defaultBrowserCount        = 1
	defaultMaxConcurrentTabs   = 4
	defaultRecycleAfterTabs    = 200
	defaultWaitReadySelector   = "body"
	defaultAcceptLanguage      = "zh-CN,zh;q=0.9,en;q=0.8"
	defaultUserAgentPlatform   = UserAgentPlatformDesktop
	defaultBrowserWindowWidth  = 1366
	defaultBrowserWindowHigh   = 768
)

// Config 定义 browserfetch 的基础运行参数。
type Config struct {
	BrowserPath        string
	Headless           *bool
	UserAgentMode      string
	UserAgent          string
	UserAgentPlatform  string
	AcceptLanguage     string
	Timeout            time.Duration
	CookieCacheTTL     time.Duration
	BrowserCount       int
	TabsPerBrowser     int
	RecycleAfterTabs   int
	MaxConcurrentTabs  int
	WaitReadySelector  string
	DisableImages      bool
	BlockedURLPatterns []string
	NoSandbox          bool
	WindowWidth        int
	WindowHeight       int
	ExtraFlags         []string
}

type normalizedConfig struct {
	BrowserPath        string
	Headless           bool
	UserAgentMode      string
	UserAgent          string
	UserAgentPlatform  string
	AcceptLanguage     string
	Timeout            time.Duration
	CookieCacheTTL     time.Duration
	BrowserCount       int
	TabsPerBrowser     int
	RecycleAfterTabs   int
	MaxConcurrentTabs  int
	WaitReadySelector  string
	DisableImages      bool
	BlockedURLPatterns []string
	NoSandbox          bool
	WindowWidth        int
	WindowHeight       int
	ExtraFlags         []string
}

// normalizeConfig 应用默认值并裁剪空白字符。
func normalizeConfig(cfg Config) normalizedConfig {
	headless := defaultHeadless
	if cfg.Headless != nil {
		headless = *cfg.Headless
	}

	userAgentMode := normalizeUserAgentMode(cfg.UserAgentMode)
	userAgentPlatform := normalizeUserAgentPlatform(cfg.UserAgentPlatform)

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultBrowserTimeout
	}

	cacheTTL := cfg.CookieCacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultCookieCacheTTL
	}

	browserCount := cfg.BrowserCount
	if browserCount <= 0 {
		browserCount = defaultBrowserCount
	}

	tabsPerBrowser := cfg.TabsPerBrowser
	if tabsPerBrowser <= 0 {
		tabsPerBrowser = cfg.MaxConcurrentTabs
	}
	if tabsPerBrowser <= 0 {
		tabsPerBrowser = defaultMaxConcurrentTabs
	}

	recycleAfterTabs := cfg.RecycleAfterTabs
	if recycleAfterTabs <= 0 {
		recycleAfterTabs = defaultRecycleAfterTabs
	}

	waitReadySelector := strings.TrimSpace(cfg.WaitReadySelector)
	if waitReadySelector == "" {
		waitReadySelector = defaultWaitReadySelector
	}

	acceptLanguage := strings.TrimSpace(cfg.AcceptLanguage)
	if acceptLanguage == "" {
		acceptLanguage = defaultAcceptLanguage
	}

	windowWidth := cfg.WindowWidth
	if windowWidth <= 0 {
		windowWidth = defaultBrowserWindowWidth
	}
	windowHeight := cfg.WindowHeight
	if windowHeight <= 0 {
		windowHeight = defaultBrowserWindowHigh
	}

	return normalizedConfig{
		BrowserPath:        normalizeBrowserPath(cfg.BrowserPath),
		Headless:           headless,
		UserAgentMode:      userAgentMode,
		UserAgent:          strings.TrimSpace(cfg.UserAgent),
		UserAgentPlatform:  userAgentPlatform,
		AcceptLanguage:     acceptLanguage,
		Timeout:            timeout,
		CookieCacheTTL:     cacheTTL,
		BrowserCount:       browserCount,
		TabsPerBrowser:     tabsPerBrowser,
		RecycleAfterTabs:   recycleAfterTabs,
		MaxConcurrentTabs:  tabsPerBrowser,
		WaitReadySelector:  waitReadySelector,
		DisableImages:      cfg.DisableImages,
		BlockedURLPatterns: compactStringSlice(cfg.BlockedURLPatterns),
		NoSandbox:          cfg.NoSandbox,
		WindowWidth:        windowWidth,
		WindowHeight:       windowHeight,
		ExtraFlags:         compactStringSlice(cfg.ExtraFlags),
	}
}

func normalizeBrowserPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	return filepath.Clean(path)
}

func normalizeUserAgentMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case UserAgentModeDefault, UserAgentModeCustom, UserAgentModeFake:
		return mode
	case "":
		return UserAgentModeDefault
	default:
		return UserAgentModeDefault
	}
}

func normalizeUserAgentPlatform(platform string) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	switch platform {
	case UserAgentPlatformMobile:
		return UserAgentPlatformMobile
	default:
		return UserAgentPlatformDesktop
	}
}

func compactStringSlice(items []string) []string {
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

func browserProcessKey(cfg normalizedConfig) string {
	parts := []string{
		cfg.BrowserPath,
		strconv.FormatBool(cfg.Headless),
		strconv.FormatBool(cfg.NoSandbox),
		strconv.Itoa(cfg.WindowWidth),
		strconv.Itoa(cfg.WindowHeight),
	}
	parts = append(parts, cfg.ExtraFlags...)

	return strings.Join(parts, "\x00")
}
