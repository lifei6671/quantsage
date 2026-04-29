package eastmoney

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

// ClientConfig 定义东财 HTTP client 的基础参数。
type ClientConfig struct {
	Endpoint      string
	QuoteEndpoint string
	Timeout       time.Duration
	MaxRetries    int
	UserAgentMode string
}

// Client 封装东财底层 HTTP 调用。
type Client struct {
	httpClient *http.Client
	config     ClientConfig
}

// NewClient 使用标准库 HTTP client 创建东财客户端。
func NewClient(cfg ClientConfig) *Client {
	return newClientWithHTTPClient(cfg, nil)
}

func newClientWithHTTPClient(cfg ClientConfig, httpClient *http.Client) *Client {
	normalized := normalizeClientConfig(cfg)
	if httpClient == nil {
		httpClient = &http.Client{Timeout: normalized.Timeout}
	} else if normalized.Timeout > 0 {
		httpClient.Timeout = normalized.Timeout
	}

	return &Client{
		httpClient: httpClient,
		config:     normalized,
	}
}

// GetHistory 读取东财历史行情接口原始响应体。
func (c *Client) GetHistory(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.get(ctx, c.config.Endpoint, path, query)
}

// GetHistoryWithHeaders 读取东财历史行情接口原始响应体，并叠加自定义请求头。
func (c *Client) GetHistoryWithHeaders(ctx context.Context, path string, query url.Values, headers http.Header) ([]byte, error) {
	return c.getWithHeaders(ctx, c.config.Endpoint, path, query, headers)
}

// GetQuote 读取东财行情接口原始响应体。
func (c *Client) GetQuote(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.get(ctx, c.config.QuoteEndpoint, path, query)
}

// GetQuoteWithHeaders 读取东财行情接口原始响应体，并叠加自定义请求头。
func (c *Client) GetQuoteWithHeaders(ctx context.Context, path string, query url.Values, headers http.Header) ([]byte, error) {
	return c.getWithHeaders(ctx, c.config.QuoteEndpoint, path, query, headers)
}

func normalizeClientConfig(cfg ClientConfig) ClientConfig {
	cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
	if cfg.Endpoint == "" {
		cfg.Endpoint = defaultEndpoint
	}
	cfg.QuoteEndpoint = strings.TrimSpace(cfg.QuoteEndpoint)
	if cfg.QuoteEndpoint == "" {
		cfg.QuoteEndpoint = defaultQuoteEndpoint
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	cfg.UserAgentMode = strings.ToLower(strings.TrimSpace(cfg.UserAgentMode))
	if cfg.UserAgentMode == "" {
		cfg.UserAgentMode = defaultUserAgentMode
	}

	return cfg
}

func (c *Client) get(ctx context.Context, baseURL, path string, query url.Values) ([]byte, error) {
	return c.getWithHeaders(ctx, baseURL, path, query, nil)
}

func (c *Client) getWithHeaders(ctx context.Context, baseURL, path string, query url.Values, headers http.Header) ([]byte, error) {
	requestURL, err := buildRequestURL(baseURL, path, query)
	if err != nil {
		return nil, datasourceUnavailable(fmt.Errorf("build eastmoney request url: %w", err))
	}

	attempts := c.config.MaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		body, retryable, err := c.doGetOnce(ctx, requestURL, headers)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retryable || attempt == attempts {
			break
		}
	}

	return nil, lastErr
}

func buildRequestURL(baseURL, path string, query url.Values) (*url.URL, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, err
	}
	if base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("invalid base url %q", baseURL)
	}

	resolved := base.ResolveReference(&url.URL{Path: path})
	if query != nil {
		resolved.RawQuery = query.Encode()
	}

	return resolved, nil
}

func (c *Client) doGetOnce(ctx context.Context, requestURL *url.URL, headers http.Header) ([]byte, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, false, datasourceUnavailable(fmt.Errorf("build eastmoney request: %w", err))
	}
	setDefaultHeaders(req, c.config.UserAgentMode)
	mergeHeaders(req.Header, headers)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, false, datasourceUnavailable(fmt.Errorf("call eastmoney %s: %w", requestURL.Path, err))
		}
		return nil, true, datasourceUnavailable(fmt.Errorf("call eastmoney %s: %w", requestURL.Path, err))
	}
	defer resp.Body.Close()

	body, err := readResponseBody(resp)
	if err != nil {
		return nil, shouldRetryStatus(resp.StatusCode), datasourceUnavailable(fmt.Errorf("read eastmoney %s response: %w", requestURL.Path, err))
	}

	if isHTMLOrBotPage(resp.Header.Get("Content-Type"), body) {
		return nil, false, datasourceUnavailable(&antiBotResponseError{
			path:       requestURL.Path,
			statusCode: resp.StatusCode,
		})
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, shouldRetryStatus(resp.StatusCode), datasourceUnavailable(
			fmt.Errorf("call eastmoney %s: http status %d", requestURL.Path, resp.StatusCode),
		)
	}

	return body, false, nil
}

type antiBotResponseError struct {
	path       string
	statusCode int
}

func (e *antiBotResponseError) Error() string {
	if e == nil {
		return "eastmoney anti-bot response"
	}
	if e.statusCode > 0 {
		return fmt.Sprintf("call eastmoney %s: html or anti-bot response (status %d)", e.path, e.statusCode)
	}

	return fmt.Sprintf("call eastmoney %s: html or anti-bot response", e.path)
}

func readResponseBody(resp *http.Response) ([]byte, error) {
	reader := io.Reader(resp.Body)
	if strings.EqualFold(strings.TrimSpace(resp.Header.Get("Content-Encoding")), "gzip") {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("decode gzip body: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	return io.ReadAll(io.LimitReader(reader, responseBodyLimit))
}

func setDefaultHeaders(req *http.Request, userAgentMode string) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Referer", req.URL.Scheme+"://"+req.URL.Host+"/")
	req.Header.Set("User-Agent", userAgentValue(userAgentMode))
}

func mergeHeaders(dst, src http.Header) {
	for key, values := range src {
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func userAgentValue(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "mobile":
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1"
	default:
		return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"
	}
}

func isHTMLOrBotPage(contentType string, body []byte) bool {
	return looksLikeHTML(contentType, body) || looksLikeBotPage(body)
}

func looksLikeHTML(contentType string, body []byte) bool {
	if strings.Contains(strings.ToLower(strings.TrimSpace(contentType)), "text/html") {
		return true
	}

	snippet := strings.ToLower(strings.TrimSpace(string(limitBodyPrefix(body, 256))))
	return strings.HasPrefix(snippet, "<!doctype html") ||
		strings.HasPrefix(snippet, "<html") ||
		strings.HasPrefix(snippet, "<body")
}

func looksLikeBotPage(body []byte) bool {
	snippet := strings.ToLower(string(limitBodyPrefix(body, 512)))
	for _, indicator := range []string{
		"captcha",
		"verify",
		"robot",
		"access denied",
		"security check",
		"验证码",
		"人机验证",
		"访问受限",
	} {
		if strings.Contains(snippet, indicator) {
			return true
		}
	}

	return false
}

func limitBodyPrefix(body []byte, limit int) []byte {
	if len(body) <= limit {
		return body
	}

	return body[:limit]
}

func shouldRetryStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

func datasourceUnavailable(err error) error {
	if err == nil {
		err = errors.New("eastmoney datasource unavailable")
	}

	return apperror.New(apperror.CodeDatasourceUnavailable, err)
}
