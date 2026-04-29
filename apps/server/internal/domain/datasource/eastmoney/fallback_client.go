package eastmoney

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/lifei6671/quantsage/apps/server/internal/infra/browserfetch"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

// HistoryClient 定义仅历史行情抓取所需的最小契约。
type HistoryClient interface {
	GetHistory(ctx context.Context, path string, query url.Values) ([]byte, error)
}

// HistoryHeaderClient 为历史行情抓取增加自定义请求头能力。
type HistoryHeaderClient interface {
	HistoryClient
	GetHistoryWithHeaders(ctx context.Context, path string, query url.Values, headers http.Header) ([]byte, error)
}

// CookieHeaderFetcher 定义浏览器 Cookie Header 获取能力。
type CookieHeaderFetcher interface {
	FetchCookieHeader(ctx context.Context, pageURL string) (string, error)
}

type fetchRequester interface {
	GetHistory(ctx context.Context, path string, query url.Values) ([]byte, error)
	GetQuote(ctx context.Context, path string, query url.Values) ([]byte, error)
}

type headerFetchRequester interface {
	fetchRequester
	GetHistoryWithHeaders(ctx context.Context, path string, query url.Values, headers http.Header) ([]byte, error)
	GetQuoteWithHeaders(ctx context.Context, path string, query url.Values, headers http.Header) ([]byte, error)
}

type browserRunner interface {
	FetchCookieHeader(ctx context.Context, pageURL string) (string, error)
}

type historyFallbackRequester struct {
	requester HistoryHeaderClient
}

// NewHistoryClientFromClientConfig 使用基础 HTTP client 配置创建 history client。
// 该入口固定为 HTTP-only，不启用浏览器回退。
func NewHistoryClientFromClientConfig(cfg ClientConfig) HistoryClient {
	return NewHistoryClientWithFallback(NewClient(cfg), nil, FallbackConfig{Mode: FetchModeHTTP})
}

// NewHistoryClientFromConfig 使用 datasource 配置创建带统一回退策略的 history client。
func NewHistoryClientFromConfig(cfg Config) HistoryClient {
	normalizedConfig := normalizeSourceConfig(cfg)
	client := NewClient(buildClientConfig(normalizedConfig))

	var browser CookieHeaderFetcher
	if buildFallbackConfig(normalizedConfig).Mode != FetchModeHTTP {
		browser = browserfetch.New(buildBrowserFetchConfig(normalizedConfig))
	}

	return NewHistoryClientWithFallback(client, browser, buildFallbackConfig(normalizedConfig))
}

// NewHistoryClientWithFallback 使用共享 history fallback 策略组装一个可注入依赖的 history client。
func NewHistoryClientWithFallback(requester HistoryHeaderClient, cookieFetcher CookieHeaderFetcher, cfg FallbackConfig) HistoryClient {
	var wrapped fetchRequester
	if requester != nil {
		wrapped = historyFallbackRequester{requester: requester}
	}

	return newFallbackClient(wrapped, cookieFetcher, cfg)
}

// fallbackClient 在现有 HTTP client 外包一层最小浏览器回退能力。
type fallbackClient struct {
	requester fetchRequester
	browser   browserRunner
	config    FallbackConfig
	initErr   error
}

func newFallbackClient(requester fetchRequester, browser browserRunner, cfg FallbackConfig) *fallbackClient {
	normalizedConfig := normalizeFallbackConfig(cfg)
	return &fallbackClient{
		requester: requester,
		browser:   browser,
		config:    normalizedConfig,
		initErr:   buildFallbackClientInitError(requester, browser, normalizedConfig),
	}
}

func normalizeFallbackConfig(cfg FallbackConfig) FallbackConfig {
	switch cfg.Mode {
	case FetchModeHTTP, FetchModeAuto, FetchModeChromedp:
	default:
		cfg.Mode = FetchModeHTTP
	}

	cfg.QuotePageURL = strings.TrimSpace(cfg.QuotePageURL)
	return cfg
}

func buildFallbackClientInitError(requester fetchRequester, browser browserRunner, cfg FallbackConfig) error {
	switch cfg.Mode {
	case FetchModeAuto, FetchModeChromedp:
		if requester == nil {
			return datasourceUnavailable(errors.New("eastmoney fallback client requester is nil"))
		}
		if _, ok := requester.(headerFetchRequester); !ok {
			return datasourceUnavailable(errors.New("eastmoney fallback requester does not support custom headers"))
		}
		if browser == nil {
			return datasourceUnavailable(errors.New("eastmoney browser fallback runner is nil"))
		}
		if cfg.QuotePageURL == "" {
			return datasourceUnavailable(errors.New("eastmoney browser fallback quote page url is empty"))
		}
	}

	return nil
}

func (c *fallbackClient) GetHistory(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.fetch(ctx, requestKindHistory, path, query)
}

func (c *fallbackClient) GetQuote(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.fetch(ctx, requestKindQuote, path, query)
}

func (c *fallbackClient) fetch(ctx context.Context, kind requestKind, path string, query url.Values) ([]byte, error) {
	if c == nil || c.requester == nil {
		return nil, datasourceUnavailable(errors.New("eastmoney fallback client requester is nil"))
	}
	if c.initErr != nil {
		return nil, c.initErr
	}

	switch c.config.Mode {
	case FetchModeChromedp:
		return c.fetchWithBrowserCookie(ctx, kind, path, query)
	case FetchModeAuto:
		body, err := c.fetchViaRequester(ctx, kind, path, query)
		if err == nil && !shouldFallback(body, nil) {
			return body, nil
		}
		if err != nil && !shouldFallback(nil, err) {
			return nil, err
		}

		return c.fetchWithBrowserCookie(ctx, kind, path, query)
	default:
		return c.fetchViaRequester(ctx, kind, path, query)
	}
}

func (c *fallbackClient) fetchViaRequester(ctx context.Context, kind requestKind, path string, query url.Values) ([]byte, error) {
	switch kind {
	case requestKindQuote:
		return c.requester.GetQuote(ctx, path, query)
	default:
		return c.requester.GetHistory(ctx, path, query)
	}
}

func (c *fallbackClient) fetchWithBrowserCookie(ctx context.Context, kind requestKind, path string, query url.Values) ([]byte, error) {
	headerRequester, ok := c.requester.(headerFetchRequester)
	if !ok {
		return nil, datasourceUnavailable(errors.New("eastmoney fallback requester does not support custom headers"))
	}
	if c.browser == nil {
		return nil, datasourceUnavailable(errors.New("eastmoney browser fallback runner is nil"))
	}
	if c.config.QuotePageURL == "" {
		return nil, datasourceUnavailable(errors.New("eastmoney browser fallback quote page url is empty"))
	}

	cookieHeader, err := c.browser.FetchCookieHeader(ctx, c.config.QuotePageURL)
	if err != nil {
		return nil, datasourceUnavailable(fmt.Errorf("fetch eastmoney browser cookie header: %w", err))
	}

	headers := make(http.Header)
	if strings.TrimSpace(cookieHeader) != "" {
		headers.Set("Cookie", cookieHeader)
	}

	switch kind {
	case requestKindQuote:
		return headerRequester.GetQuoteWithHeaders(ctx, path, query, headers)
	default:
		return headerRequester.GetHistoryWithHeaders(ctx, path, query, headers)
	}
}

func shouldFallback(body []byte, err error) bool {
	if len(body) > 0 && (looksLikeHTML("", body) || looksLikeBotPage(body)) {
		return true
	}

	return isAntiBotFallbackError(err)
}

func (r historyFallbackRequester) GetHistory(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return r.requester.GetHistory(ctx, path, query)
}

func (r historyFallbackRequester) GetQuote(_ context.Context, _ string, _ url.Values) ([]byte, error) {
	return nil, datasourceUnavailable(errors.New("eastmoney history fallback requester does not support quote requests"))
}

func (r historyFallbackRequester) GetHistoryWithHeaders(ctx context.Context, path string, query url.Values, headers http.Header) ([]byte, error) {
	return r.requester.GetHistoryWithHeaders(ctx, path, query, headers)
}

func (r historyFallbackRequester) GetQuoteWithHeaders(_ context.Context, _ string, _ url.Values, _ http.Header) ([]byte, error) {
	return nil, datasourceUnavailable(errors.New("eastmoney history fallback requester does not support quote requests"))
}

func isAntiBotFallbackError(err error) bool {
	if err == nil || apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		return false
	}

	if _, ok := errors.AsType[*antiBotResponseError](err); ok {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "html or anti-bot response") ||
		strings.Contains(message, "anti-bot response")
}

type requestKind string

const (
	requestKindHistory requestKind = "history"
	requestKindQuote   requestKind = "quote"
)
