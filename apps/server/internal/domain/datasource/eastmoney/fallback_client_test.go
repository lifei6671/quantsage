package eastmoney

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

func TestFallbackClientAutoSkipsBrowserWhenHTTPGetHistorySucceeds(t *testing.T) {
	t.Parallel()

	requester := &stubFetchRequester{
		historyResponses: []stubResponse{
			{body: []byte(`{"rc":0}`)},
		},
	}
	browser := &stubBrowserRunner{}
	client := newFallbackClient(requester, browser, FallbackConfig{
		Mode:         FetchModeAuto,
		QuotePageURL: "https://quote.eastmoney.com/concept/sh000001.html",
	})

	body, err := client.GetHistory(context.Background(), historyKLinePath, url.Values{"secid": []string{"1.600000"}})
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if got := string(body); got != `{"rc":0}` {
		t.Fatalf("GetHistory() body = %q, want %q", got, `{"rc":0}`)
	}
	if len(browser.pageURLs) != 0 {
		t.Fatalf("browser calls = %d, want 0", len(browser.pageURLs))
	}
	if len(requester.historyHeaderCalls) != 0 {
		t.Fatalf("history header calls = %d, want 0", len(requester.historyHeaderCalls))
	}
}

func TestFallbackClientAutoFallsBackWhenHTTPBodyLooksLikeHTML(t *testing.T) {
	t.Parallel()

	requester := &stubFetchRequester{
		quoteResponses: []stubResponse{
			{body: []byte("<html><body>captcha verify</body></html>")},
		},
		quoteHeaderResponses: []stubResponse{
			{body: []byte(`{"data":{"diff":[]}}`)},
		},
	}
	browser := &stubBrowserRunner{
		cookieHeaders: []stubCookieResponse{
			{cookieHeader: "st_si=abc123"},
		},
	}
	client := newFallbackClient(requester, browser, FallbackConfig{
		Mode:         FetchModeAuto,
		QuotePageURL: "https://quote.eastmoney.com/concept/sh000001.html",
	})

	body, err := client.GetQuote(context.Background(), stockListPath, url.Values{"pn": []string{"1"}})
	if err != nil {
		t.Fatalf("GetQuote() error = %v", err)
	}
	if got := string(body); got != `{"data":{"diff":[]}}` {
		t.Fatalf("GetQuote() body = %q, want %q", got, `{"data":{"diff":[]}}`)
	}
	if len(browser.pageURLs) != 1 {
		t.Fatalf("browser calls = %d, want 1", len(browser.pageURLs))
	}
	if got := browser.pageURLs[0]; got != "https://quote.eastmoney.com/concept/sh000001.html" {
		t.Fatalf("browser page url = %q, want %q", got, "https://quote.eastmoney.com/concept/sh000001.html")
	}
	if len(requester.quoteHeaderCalls) != 1 {
		t.Fatalf("quote header calls = %d, want 1", len(requester.quoteHeaderCalls))
	}
	if got := requester.quoteHeaderCalls[0].headers.Get("Cookie"); got != "st_si=abc123" {
		t.Fatalf("quote header Cookie = %q, want %q", got, "st_si=abc123")
	}
}

func TestFallbackClientAutoRetriesWithBrowserCookieAfterHTMLAntiBotError(t *testing.T) {
	t.Parallel()

	requester := &stubFetchRequester{
		historyResponses: []stubResponse{
			{err: datasourceUnavailable(&antiBotResponseError{path: "/api", statusCode: http.StatusForbidden})},
		},
		historyHeaderResponses: []stubResponse{
			{body: []byte(`{"data":{"code":"000001"}}`)},
		},
	}
	browser := &stubBrowserRunner{
		cookieHeaders: []stubCookieResponse{
			{cookieHeader: "st_si=xyz789; em_hq_fls=js"},
		},
	}
	client := newFallbackClient(requester, browser, FallbackConfig{
		Mode:         FetchModeAuto,
		QuotePageURL: "https://quote.eastmoney.com/concept/sh000001.html",
	})

	body, err := client.GetHistory(context.Background(), historyKLinePath, url.Values{"secid": []string{"0.000001"}})
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if got := string(body); got != `{"data":{"code":"000001"}}` {
		t.Fatalf("GetHistory() body = %q, want %q", got, `{"data":{"code":"000001"}}`)
	}
	if len(requester.historyCalls) != 1 {
		t.Fatalf("history calls = %d, want 1", len(requester.historyCalls))
	}
	if len(requester.historyHeaderCalls) != 1 {
		t.Fatalf("history header calls = %d, want 1", len(requester.historyHeaderCalls))
	}
	if got := requester.historyHeaderCalls[0].headers.Get("Cookie"); got != "st_si=xyz789; em_hq_fls=js" {
		t.Fatalf("history header Cookie = %q, want %q", got, "st_si=xyz789; em_hq_fls=js")
	}
}

func TestNewHistoryClientWithFallbackUsesSharedHistoryFallback(t *testing.T) {
	t.Parallel()

	requester := &stubFetchRequester{
		historyResponses: []stubResponse{
			{body: []byte("<html><body>security check</body></html>")},
		},
		historyHeaderResponses: []stubResponse{
			{body: []byte(`{"rc":0}`)},
		},
	}
	browser := &stubBrowserRunner{
		cookieHeaders: []stubCookieResponse{
			{cookieHeader: "st_si=shared-history"},
		},
	}

	client := NewHistoryClientWithFallback(requester, browser, FallbackConfig{
		Mode:         FetchModeAuto,
		QuotePageURL: "https://quote.eastmoney.com/concept/sh000001.html",
	})

	body, err := client.GetHistory(context.Background(), historyKLinePath, url.Values{"secid": []string{"0.000001"}})
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if got := string(body); got != `{"rc":0}` {
		t.Fatalf("GetHistory() body = %q, want %q", got, `{"rc":0}`)
	}
	if len(requester.historyCalls) != 1 {
		t.Fatalf("history calls = %d, want 1", len(requester.historyCalls))
	}
	if len(requester.historyHeaderCalls) != 1 {
		t.Fatalf("history header calls = %d, want 1", len(requester.historyHeaderCalls))
	}
	if got := requester.historyHeaderCalls[0].headers.Get("Cookie"); got != "st_si=shared-history" {
		t.Fatalf("history header Cookie = %q, want %q", got, "st_si=shared-history")
	}
}

func TestFallbackClientAutoReturnsFallbackErrorWhenRetryStillFails(t *testing.T) {
	t.Parallel()

	requester := &stubFetchRequester{
		historyResponses: []stubResponse{
			{err: datasourceUnavailable(errors.New("call eastmoney /api: html or anti-bot response"))},
		},
		historyHeaderResponses: []stubResponse{
			{err: datasourceUnavailable(errors.New("call eastmoney /api: http status 503"))},
		},
	}
	browser := &stubBrowserRunner{
		cookieHeaders: []stubCookieResponse{
			{cookieHeader: "st_si=retry"},
		},
	}
	client := newFallbackClient(requester, browser, FallbackConfig{
		Mode:         FetchModeAuto,
		QuotePageURL: "https://quote.eastmoney.com/concept/sh000001.html",
	})

	_, err := client.GetHistory(context.Background(), historyKLinePath, nil)
	if err == nil {
		t.Fatal("GetHistory() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "http status 503") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "http status 503")
	}
	if len(browser.pageURLs) != 1 {
		t.Fatalf("browser calls = %d, want 1", len(browser.pageURLs))
	}
}

func TestFallbackClientHTTPModeDoesNotUseBrowserFallback(t *testing.T) {
	t.Parallel()

	requester := &stubFetchRequester{
		historyResponses: []stubResponse{
			{err: datasourceUnavailable(errors.New("call eastmoney /api: html or anti-bot response"))},
		},
	}
	browser := &stubBrowserRunner{
		cookieHeaders: []stubCookieResponse{
			{cookieHeader: "unused=true"},
		},
	}
	client := newFallbackClient(requester, browser, FallbackConfig{
		Mode:         FetchModeHTTP,
		QuotePageURL: "https://quote.eastmoney.com/concept/sh000001.html",
	})

	_, err := client.GetHistory(context.Background(), historyKLinePath, nil)
	if err == nil {
		t.Fatal("GetHistory() error = nil, want non-nil")
	}
	if len(browser.pageURLs) != 0 {
		t.Fatalf("browser calls = %d, want 0", len(browser.pageURLs))
	}
	if len(requester.historyHeaderCalls) != 0 {
		t.Fatalf("history header calls = %d, want 0", len(requester.historyHeaderCalls))
	}
}

func TestFallbackClientChromedpModeGetsCookieBeforeQuoteRequest(t *testing.T) {
	t.Parallel()

	requester := &stubFetchRequester{
		quoteHeaderResponses: []stubResponse{
			{body: []byte(`{"data":{"total":1}}`)},
		},
	}
	browser := &stubBrowserRunner{
		cookieHeaders: []stubCookieResponse{
			{cookieHeader: "st_si=direct"},
		},
	}
	client := newFallbackClient(requester, browser, FallbackConfig{
		Mode:         FetchModeChromedp,
		QuotePageURL: "https://quote.eastmoney.com/concept/sh000001.html",
	})

	body, err := client.GetQuote(context.Background(), stockListPath, nil)
	if err != nil {
		t.Fatalf("GetQuote() error = %v", err)
	}
	if got := string(body); got != `{"data":{"total":1}}` {
		t.Fatalf("GetQuote() body = %q, want %q", got, `{"data":{"total":1}}`)
	}
	if len(browser.pageURLs) != 1 {
		t.Fatalf("browser calls = %d, want 1", len(browser.pageURLs))
	}
	if len(requester.quoteCalls) != 0 {
		t.Fatalf("plain quote calls = %d, want 0", len(requester.quoteCalls))
	}
	if len(requester.quoteHeaderCalls) != 1 {
		t.Fatalf("quote header calls = %d, want 1", len(requester.quoteHeaderCalls))
	}
	if got := requester.quoteHeaderCalls[0].headers.Get("Cookie"); got != "st_si=direct" {
		t.Fatalf("quote header Cookie = %q, want %q", got, "st_si=direct")
	}
}

func TestFallbackClientAutoFallsBackForHTMLStatusResponse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		statusCode int
	}{
		{name: "403 html", statusCode: http.StatusForbidden},
		{name: "429 html", statusCode: http.StatusTooManyRequests},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			attempt := 0
			client := newClientWithHTTPClient(ClientConfig{
				Endpoint:      "https://hist.example.com",
				QuoteEndpoint: "https://quote.example.com",
				Timeout:       5 * time.Second,
				MaxRetries:    0,
			}, &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					attempt++
					if attempt == 1 {
						return newHTTPResponse(tc.statusCode, map[string]string{
							"Content-Type": "text/html; charset=utf-8",
						}, []byte("<html><body>security check</body></html>")), nil
					}
					if got := r.Header.Get("Cookie"); got != "st_si=browser-cookie" {
						t.Fatalf("Cookie = %q, want %q", got, "st_si=browser-cookie")
					}

					return newHTTPResponse(http.StatusOK, map[string]string{
						"Content-Type": "application/json",
					}, []byte(`{"ok":true}`)), nil
				}),
			})
			browser := &stubBrowserRunner{
				cookieHeaders: []stubCookieResponse{
					{cookieHeader: "st_si=browser-cookie"},
				},
			}
			fallback := newFallbackClient(client, browser, FallbackConfig{
				Mode:         FetchModeAuto,
				QuotePageURL: "https://quote.eastmoney.com/concept/sh000001.html",
			})

			body, err := fallback.GetHistory(context.Background(), historyKLinePath, nil)
			if err != nil {
				t.Fatalf("GetHistory() error = %v", err)
			}
			if got := string(body); got != `{"ok":true}` {
				t.Fatalf("GetHistory() body = %q, want %q", got, `{"ok":true}`)
			}
			if attempt != 2 {
				t.Fatalf("attempts = %d, want 2", attempt)
			}
			if len(browser.pageURLs) != 1 {
				t.Fatalf("browser calls = %d, want 1", len(browser.pageURLs))
			}
		})
	}
}

func TestFallbackClientAutoFailsWhenBrowserRunnerIsNil(t *testing.T) {
	t.Parallel()

	requester := &stubFetchRequester{
		historyResponses: []stubResponse{
			{err: datasourceUnavailable(&antiBotResponseError{path: "/api", statusCode: http.StatusForbidden})},
		},
	}
	client := newFallbackClient(requester, nil, FallbackConfig{
		Mode:         FetchModeAuto,
		QuotePageURL: "https://quote.eastmoney.com/concept/sh000001.html",
	})

	_, err := client.GetHistory(context.Background(), historyKLinePath, nil)
	if err == nil {
		t.Fatal("GetHistory() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "browser fallback runner is nil") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "browser fallback runner is nil")
	}
	if len(requester.historyCalls) != 0 {
		t.Fatalf("history calls = %d, want 0 because invalid config should fail fast", len(requester.historyCalls))
	}
}

func TestFallbackClientAutoFailsWhenQuotePageURLIsEmpty(t *testing.T) {
	t.Parallel()

	requester := &stubFetchRequester{
		historyResponses: []stubResponse{
			{err: datasourceUnavailable(&antiBotResponseError{path: "/api", statusCode: http.StatusForbidden})},
		},
	}
	client := newFallbackClient(requester, &stubBrowserRunner{}, FallbackConfig{
		Mode: FetchModeAuto,
	})

	_, err := client.GetHistory(context.Background(), historyKLinePath, nil)
	if err == nil {
		t.Fatal("GetHistory() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "quote page url is empty") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "quote page url is empty")
	}
	if len(requester.historyCalls) != 0 {
		t.Fatalf("history calls = %d, want 0 because invalid config should fail fast", len(requester.historyCalls))
	}
}

func TestFallbackClientAutoFailsWhenRequesterDoesNotSupportHeaders(t *testing.T) {
	t.Parallel()

	requester := &stubBasicRequester{
		historyResponses: []stubResponse{
			{err: datasourceUnavailable(&antiBotResponseError{path: "/api", statusCode: http.StatusForbidden})},
		},
	}
	client := newFallbackClient(requester, &stubBrowserRunner{
		cookieHeaders: []stubCookieResponse{
			{cookieHeader: "st_si=unused"},
		},
	}, FallbackConfig{
		Mode:         FetchModeAuto,
		QuotePageURL: "https://quote.eastmoney.com/concept/sh000001.html",
	})

	_, err := client.GetHistory(context.Background(), historyKLinePath, nil)
	if err == nil {
		t.Fatal("GetHistory() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "does not support custom headers") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "does not support custom headers")
	}
	if len(requester.historyResponses) != 1 {
		t.Fatalf("remaining history responses = %d, want %d because invalid config should fail before requester call", len(requester.historyResponses), 1)
	}
}

func TestClientGetHistoryWithHeadersMergesCustomHeaders(t *testing.T) {
	t.Parallel()

	client := newClientWithHTTPClient(ClientConfig{
		Endpoint:      "https://hist.example.com",
		QuoteEndpoint: "https://quote.example.com",
		Timeout:       5 * time.Second,
		MaxRetries:    0,
		UserAgentMode: "stable",
	}, &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.Header.Get("Cookie"); got != "st_si=abc123" {
				t.Fatalf("Cookie = %q, want %q", got, "st_si=abc123")
			}
			if got := r.Header.Get("Referer"); got != "https://custom.example.com/" {
				t.Fatalf("Referer = %q, want %q", got, "https://custom.example.com/")
			}
			if got := r.Header.Get("User-Agent"); got != userAgentValue("stable") {
				t.Fatalf("User-Agent = %q, want %q", got, userAgentValue("stable"))
			}

			return newHTTPResponse(http.StatusOK, map[string]string{
				"Content-Type": "application/json",
			}, []byte(`{"ok":true}`)), nil
		}),
	})

	headers := make(http.Header)
	headers.Set("Cookie", "st_si=abc123")
	headers.Set("Referer", "https://custom.example.com/")

	body, err := client.GetHistoryWithHeaders(context.Background(), historyKLinePath, nil, headers)
	if err != nil {
		t.Fatalf("GetHistoryWithHeaders() error = %v", err)
	}
	if got := string(body); got != `{"ok":true}` {
		t.Fatalf("GetHistoryWithHeaders() body = %q, want %q", got, `{"ok":true}`)
	}
}

func TestClientGetQuoteWithHeadersPropagatesDatasourceUnavailableForHTML(t *testing.T) {
	t.Parallel()

	client := newClientWithHTTPClient(ClientConfig{
		Endpoint:      "https://hist.example.com",
		QuoteEndpoint: "https://quote.example.com",
		Timeout:       5 * time.Second,
		MaxRetries:    0,
	}, &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.Header.Get("Cookie"); got != "st_si=quote-cookie" {
				t.Fatalf("Cookie = %q, want %q", got, "st_si=quote-cookie")
			}

			return newHTTPResponse(http.StatusOK, map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			}, []byte("<html><body>security check</body></html>")), nil
		}),
	})

	headers := make(http.Header)
	headers.Set("Cookie", "st_si=quote-cookie")

	_, err := client.GetQuoteWithHeaders(context.Background(), stockListPath, nil, headers)
	if err == nil {
		t.Fatal("GetQuoteWithHeaders() error = nil, want non-nil")
	}
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("CodeOf(error) = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}
	if !strings.Contains(err.Error(), "html or anti-bot response") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "html or anti-bot response")
	}
}

type stubFetchRequester struct {
	historyResponses       []stubResponse
	historyHeaderResponses []stubResponse
	quoteResponses         []stubResponse
	quoteHeaderResponses   []stubResponse
	historyCalls           []stubRequestCall
	historyHeaderCalls     []stubHeaderCall
	quoteCalls             []stubRequestCall
	quoteHeaderCalls       []stubHeaderCall
}

func (s *stubFetchRequester) GetHistory(_ context.Context, path string, query url.Values) ([]byte, error) {
	s.historyCalls = append(s.historyCalls, stubRequestCall{path: path, query: cloneQueryValues(query)})
	return popStubResponse(&s.historyResponses)
}

func (s *stubFetchRequester) GetQuote(_ context.Context, path string, query url.Values) ([]byte, error) {
	s.quoteCalls = append(s.quoteCalls, stubRequestCall{path: path, query: cloneQueryValues(query)})
	return popStubResponse(&s.quoteResponses)
}

func (s *stubFetchRequester) GetHistoryWithHeaders(_ context.Context, path string, query url.Values, headers http.Header) ([]byte, error) {
	s.historyHeaderCalls = append(s.historyHeaderCalls, stubHeaderCall{
		path:    path,
		query:   cloneQueryValues(query),
		headers: headers.Clone(),
	})
	return popStubResponse(&s.historyHeaderResponses)
}

func (s *stubFetchRequester) GetQuoteWithHeaders(_ context.Context, path string, query url.Values, headers http.Header) ([]byte, error) {
	s.quoteHeaderCalls = append(s.quoteHeaderCalls, stubHeaderCall{
		path:    path,
		query:   cloneQueryValues(query),
		headers: headers.Clone(),
	})
	return popStubResponse(&s.quoteHeaderResponses)
}

type stubResponse struct {
	body []byte
	err  error
}

type stubRequestCall struct {
	path  string
	query url.Values
}

type stubHeaderCall struct {
	path    string
	query   url.Values
	headers http.Header
}

func popStubResponse(queue *[]stubResponse) ([]byte, error) {
	if len(*queue) == 0 {
		return nil, errors.New("unexpected requester call")
	}

	response := (*queue)[0]
	*queue = (*queue)[1:]
	return response.body, response.err
}

type stubBasicRequester struct {
	historyResponses []stubResponse
	quoteResponses   []stubResponse
}

func (s *stubBasicRequester) GetHistory(_ context.Context, _ string, _ url.Values) ([]byte, error) {
	return popStubResponse(&s.historyResponses)
}

func (s *stubBasicRequester) GetQuote(_ context.Context, _ string, _ url.Values) ([]byte, error) {
	return popStubResponse(&s.quoteResponses)
}

type stubBrowserRunner struct {
	cookieHeaders []stubCookieResponse
	pageURLs      []string
}

func (s *stubBrowserRunner) FetchCookieHeader(_ context.Context, pageURL string) (string, error) {
	s.pageURLs = append(s.pageURLs, pageURL)
	if len(s.cookieHeaders) == 0 {
		return "", errors.New("unexpected browser call")
	}

	response := s.cookieHeaders[0]
	s.cookieHeaders = s.cookieHeaders[1:]
	return response.cookieHeader, response.err
}

type stubCookieResponse struct {
	cookieHeader string
	err          error
}

func cloneQueryValues(values url.Values) url.Values {
	if values == nil {
		return nil
	}

	cloned := make(url.Values, len(values))
	for key, items := range values {
		copied := make([]string, len(items))
		copy(copied, items)
		cloned[key] = copied
	}

	return cloned
}
