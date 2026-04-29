package eastmoney

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/infra/browserfetch"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

func TestSourceListStocksMapsEastMoneyRows(t *testing.T) {
	t.Parallel()

	source := newTestSource(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != stockListPath {
			t.Fatalf("request path = %q, want %q", req.URL.Path, stockListPath)
		}

		return newHTTPResponse(http.StatusOK, map[string]string{
			"Content-Type": "application/json",
		}, []byte(`{"rc":0,"data":{"diff":[{"f12":"000001","f13":0,"f14":"平安银行","f100":"银行","f26":"19910403"},{"f12":"600000","f13":1,"f14":"浦发银行","f100":"银行","f26":"19991110"},{"f12":"430001","f13":0,"f14":"北交样例","f100":"北交所","f26":"20211115"}]}}`)), nil
	})

	items, err := source.ListStocks(context.Background())
	if err != nil {
		t.Fatalf("ListStocks() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want %d", len(items), 3)
	}
	if items[0].TSCode != "000001.SZ" || items[0].Exchange != "SZSE" || items[0].Source != sourceName {
		t.Fatalf("items[0] = %+v, want mapped SZ stock", items[0])
	}
	if items[1].TSCode != "600000.SH" || items[1].Exchange != "SSE" {
		t.Fatalf("items[1] = %+v, want mapped SH stock", items[1])
	}
	if items[2].TSCode != "430001.BJ" || items[2].Exchange != "BSE" {
		t.Fatalf("items[2] = %+v, want mapped BJ stock", items[2])
	}
	if got := items[0].ListDate.Format("2006-01-02"); got != "1991-04-03" {
		t.Fatalf("items[0].ListDate = %q, want %q", got, "1991-04-03")
	}
}

func TestSourceListTradeCalendarDerivesTradeDays(t *testing.T) {
	t.Parallel()

	source := newTestSource(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != historyKLinePath {
			t.Fatalf("request path = %q, want %q", req.URL.Path, historyKLinePath)
		}
		if got := req.URL.Query().Get("secid"); got != "1.000001" {
			t.Fatalf("secid = %q, want %q", got, "1.000001")
		}
		if got := req.URL.Query().Get("beg"); got != "20260427" {
			t.Fatalf("beg = %q, want %q", got, "20260427")
		}
		if got := req.URL.Query().Get("end"); got != "20260429" {
			t.Fatalf("end = %q, want %q", got, "20260429")
		}

		return newHTTPResponse(http.StatusOK, map[string]string{
			"Content-Type": "application/json",
		}, []byte(`{"rc":0,"data":{"klines":["2026-04-27,10,10.4,10.5,9.9,100000,1000000,2.0,4.0,0.4,1.0","2026-04-28,10.4,10.2,10.6,10.1,90000,920000,2.0,-1.92,-0.2,0.9","2026-04-29,10.2,10.3,10.4,10.0,88000,900000,1.5,0.98,0.1,0.8"]}}`)), nil
	})

	startDate := time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 29, 7, 0, 0, 0, time.UTC)
	items, err := source.ListTradeCalendar(context.Background(), "SSE", startDate, endDate)
	if err != nil {
		t.Fatalf("ListTradeCalendar() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want %d", len(items), 3)
	}
	if !items[0].IsOpen || !items[1].IsOpen || !items[2].IsOpen {
		t.Fatalf("items = %+v, want all open", items)
	}
	if !items[1].PretradeDate.Equal(time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("items[1].PretradeDate = %s, want 2026-04-27", items[1].PretradeDate)
	}
}

func TestSourceListTradeCalendarRejectsUnsupportedExchange(t *testing.T) {
	t.Parallel()

	source := newTestSource(func(req *http.Request) (*http.Response, error) {
		return newHTTPResponse(http.StatusOK, map[string]string{"Content-Type": "application/json"}, []byte(`{"rc":0,"data":{"klines":[]}}`)), nil
	})

	_, err := source.ListTradeCalendar(context.Background(), "HKEX", time.Now(), time.Now())
	if err == nil {
		t.Fatal("ListTradeCalendar() error = nil, want non-nil")
	}
	if apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("CodeOf(error) = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
}

func TestSourceListDailyBarsMapsRows(t *testing.T) {
	t.Parallel()

	source := newTestSource(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case stockListPath:
			return newHTTPResponse(http.StatusOK, map[string]string{
				"Content-Type": "application/json",
			}, []byte(`{"rc":0,"data":{"diff":[{"f12":"000001","f13":0,"f14":"平安银行"},{"f12":"600000","f13":1,"f14":"浦发银行"}]}}`)), nil
		case historyKLinePath:
			switch req.URL.Query().Get("secid") {
			case "0.000001":
				return newHTTPResponse(http.StatusOK, map[string]string{
					"Content-Type": "application/json",
				}, []byte(`{"rc":0,"data":{"klines":["2026-04-27,10.10,10.40,10.50,10.00,100000,1000000,5.0,4.00,0.40,1.20"]}}`)), nil
			case "1.600000":
				return newHTTPResponse(http.StatusOK, map[string]string{
					"Content-Type": "application/json",
				}, []byte(`{"rc":0,"data":{"klines":["2026-04-27,8.10,8.05,8.20,8.00,80000,640000,2.5,-0.62,-0.05,0.70"]}}`)), nil
			default:
				t.Fatalf("unexpected secid %q", req.URL.Query().Get("secid"))
			}
		default:
			t.Fatalf("unexpected request path %q", req.URL.Path)
		}

		return nil, nil
	})

	startDate := time.Date(2026, 4, 27, 9, 30, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 27, 15, 0, 0, 0, time.UTC)
	items, err := source.ListDailyBars(context.Background(), startDate, endDate)
	if err != nil {
		t.Fatalf("ListDailyBars() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want %d", len(items), 2)
	}

	gotByCode := make(map[string]struct {
		closePrice decimal.Decimal
		preClose   decimal.Decimal
		pctChg     decimal.Decimal
	})
	for _, item := range items {
		gotByCode[item.TSCode] = struct {
			closePrice decimal.Decimal
			preClose   decimal.Decimal
			pctChg     decimal.Decimal
		}{
			closePrice: item.Close,
			preClose:   item.PreClose,
			pctChg:     item.PctChg,
		}
	}

	szItem, ok := gotByCode["000001.SZ"]
	if !ok {
		t.Fatal("missing 000001.SZ daily bar")
	}
	if !szItem.closePrice.Equal(decimal.RequireFromString("10.40")) || !szItem.preClose.Equal(decimal.RequireFromString("10.00")) {
		t.Fatalf("000001.SZ = %+v, want close=10.40 pre_close=10.00", szItem)
	}

	shItem, ok := gotByCode["600000.SH"]
	if !ok {
		t.Fatal("missing 600000.SH daily bar")
	}
	if !shItem.pctChg.Equal(decimal.RequireFromString("-0.62")) {
		t.Fatalf("600000.SH pct_chg = %s, want -0.62", shItem.pctChg)
	}
}

func TestSourceBusinessErrorsWrapDatasourceUnavailable(t *testing.T) {
	t.Parallel()

	source := newTestSource(func(req *http.Request) (*http.Response, error) {
		return newHTTPResponse(http.StatusOK, map[string]string{
			"Content-Type": "application/json",
		}, []byte(`{"rc":2,"message":"system busy"}`)), nil
	})

	_, err := source.ListStocks(context.Background())
	if err == nil {
		t.Fatal("ListStocks() error = nil, want non-nil")
	}
	if apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("CodeOf(error) = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
	if !strings.Contains(err.Error(), "system busy") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "system busy")
	}
}

func TestSourceListStocksFallsBackToBrowserCookieAfterHTMLAntiBot(t *testing.T) {
	t.Parallel()

	requests := 0
	source := newTestSourceWithFallback(FetchModeAuto, &stubBrowserRunner{
		cookieHeaders: []stubCookieResponse{
			{cookieHeader: "st_si=browser-cookie"},
		},
	}, func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path != stockListPath {
			t.Fatalf("request path = %q, want %q", req.URL.Path, stockListPath)
		}
		if requests == 1 {
			return newHTTPResponse(http.StatusForbidden, map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			}, []byte("<html><body>security check</body></html>")), nil
		}
		if got := req.Header.Get("Cookie"); got != "st_si=browser-cookie" {
			t.Fatalf("Cookie = %q, want %q", got, "st_si=browser-cookie")
		}

		return newHTTPResponse(http.StatusOK, map[string]string{
			"Content-Type": "application/json",
		}, []byte(`{"rc":0,"data":{"diff":[{"f12":"000001","f13":0,"f14":"平安银行","f100":"银行","f26":"19910403"}]}}`)), nil
	})

	items, err := source.ListStocks(context.Background())
	if err != nil {
		t.Fatalf("ListStocks() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
	if items[0].TSCode != "000001.SZ" {
		t.Fatalf("items[0].TSCode = %q, want %q", items[0].TSCode, "000001.SZ")
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want %d", requests, 2)
	}
}

func TestSourceListTradeCalendarFallsBackToBrowserCookieAfterHTMLAntiBot(t *testing.T) {
	t.Parallel()

	requests := 0
	source := newTestSourceWithFallback(FetchModeAuto, &stubBrowserRunner{
		cookieHeaders: []stubCookieResponse{
			{cookieHeader: "st_si=history-cookie"},
		},
	}, func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path != historyKLinePath {
			t.Fatalf("request path = %q, want %q", req.URL.Path, historyKLinePath)
		}
		if requests == 1 {
			return newHTTPResponse(http.StatusForbidden, map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			}, []byte("<html><body>security check</body></html>")), nil
		}
		if got := req.Header.Get("Cookie"); got != "st_si=history-cookie" {
			t.Fatalf("Cookie = %q, want %q", got, "st_si=history-cookie")
		}

		return newHTTPResponse(http.StatusOK, map[string]string{
			"Content-Type": "application/json",
		}, []byte(`{"rc":0,"data":{"klines":["2026-04-28,10.4,10.2,10.6,10.1,90000,920000,2.0,-1.92,-0.2,0.9"]}}`)), nil
	})

	items, err := source.ListTradeCalendar(
		context.Background(),
		"SSE",
		time.Date(2026, 4, 28, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 28, 15, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("ListTradeCalendar() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
	if !items[0].IsOpen {
		t.Fatalf("items[0].IsOpen = %v, want true", items[0].IsOpen)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want %d", requests, 2)
	}
}

func TestBuildBrowserFetchConfigMapsUserAgentStrategyAndKeepsBrowserTimeoutDefaultSeparateFromHTTP(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		sourceConfig  Config
		wantMode      string
		wantUserAgent string
		wantPlatform  string
		wantTimeout   time.Duration
		wantCookieTTL time.Duration
		wantHeadless  bool
	}{
		{
			name: "stable browser ua reuses stable custom ua",
			sourceConfig: Config{
				UserAgentMode:           "stable",
				BrowserUserAgentMode:    "stable",
				BrowserTimeoutSeconds:   70,
				BrowserCookieTTLSeconds: 900,
				BrowserHeadless:         true,
				BrowserCount:            4,
				BrowserTabsPerBrowser:   3,
				BrowserRecycleAfterTabs: 30,
			},
			wantMode:      browserfetch.UserAgentModeFake,
			wantUserAgent: userAgentValue("stable"),
			wantPlatform:  browserfetch.UserAgentPlatformDesktop,
			wantTimeout:   70 * time.Second,
			wantCookieTTL: 900 * time.Second,
			wantHeadless:  true,
		},
		{
			name: "mobile browser ua reuses mobile custom ua",
			sourceConfig: Config{
				UserAgentMode:        "stable",
				BrowserUserAgentMode: "mobile",
			},
			wantMode:      browserfetch.UserAgentModeFake,
			wantUserAgent: userAgentValue("mobile"),
			wantPlatform:  browserfetch.UserAgentPlatformMobile,
			wantTimeout:   defaultBrowserTimeout,
			wantCookieTTL: defaultBrowserCookieTTL,
			wantHeadless:  false,
		},
		{
			name: "default browser ua keeps browser native ua",
			sourceConfig: Config{
				UserAgentMode:        "mobile",
				BrowserUserAgentMode: "default",
				BrowserHeadless:      true,
			},
			wantMode:      browserfetch.UserAgentModeDefault,
			wantUserAgent: "",
			wantPlatform:  browserfetch.UserAgentPlatformDesktop,
			wantTimeout:   defaultBrowserTimeout,
			wantCookieTTL: defaultBrowserCookieTTL,
			wantHeadless:  true,
		},
		{
			name: "custom browser ua inherits normalized source ua",
			sourceConfig: Config{
				UserAgentMode:        "mobile",
				BrowserUserAgentMode: "custom",
				BrowserHeadless:      true,
			},
			wantMode:      browserfetch.UserAgentModeFake,
			wantUserAgent: userAgentValue("mobile"),
			wantPlatform:  browserfetch.UserAgentPlatformDesktop,
			wantTimeout:   defaultBrowserTimeout,
			wantCookieTTL: defaultBrowserCookieTTL,
			wantHeadless:  true,
		},
		{
			name: "invalid browser ua mode falls back predictably to source ua mode",
			sourceConfig: Config{
				UserAgentMode:        "mobile",
				BrowserUserAgentMode: "unexpected",
				BrowserHeadless:      true,
			},
			wantMode:      browserfetch.UserAgentModeFake,
			wantUserAgent: userAgentValue("mobile"),
			wantPlatform:  browserfetch.UserAgentPlatformMobile,
			wantTimeout:   defaultBrowserTimeout,
			wantCookieTTL: defaultBrowserCookieTTL,
			wantHeadless:  true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildBrowserFetchConfig(tc.sourceConfig)
			if got.UserAgentMode != tc.wantMode {
				t.Fatalf("UserAgentMode = %q, want %q", got.UserAgentMode, tc.wantMode)
			}
			if got.UserAgent != tc.wantUserAgent {
				t.Fatalf("UserAgent = %q, want %q", got.UserAgent, tc.wantUserAgent)
			}
			if got.UserAgentPlatform != tc.wantPlatform {
				t.Fatalf("UserAgentPlatform = %q, want %q", got.UserAgentPlatform, tc.wantPlatform)
			}
			if got.Timeout != tc.wantTimeout {
				t.Fatalf("Timeout = %s, want %s", got.Timeout, tc.wantTimeout)
			}
			if got.CookieCacheTTL != tc.wantCookieTTL {
				t.Fatalf("CookieCacheTTL = %s, want %s", got.CookieCacheTTL, tc.wantCookieTTL)
			}
			if tc.sourceConfig.BrowserCount > 0 && got.BrowserCount != tc.sourceConfig.BrowserCount {
				t.Fatalf("BrowserCount = %d, want %d", got.BrowserCount, tc.sourceConfig.BrowserCount)
			}
			if tc.sourceConfig.BrowserTabsPerBrowser > 0 && got.TabsPerBrowser != tc.sourceConfig.BrowserTabsPerBrowser {
				t.Fatalf("TabsPerBrowser = %d, want %d", got.TabsPerBrowser, tc.sourceConfig.BrowserTabsPerBrowser)
			}
			if tc.sourceConfig.BrowserRecycleAfterTabs > 0 && got.RecycleAfterTabs != tc.sourceConfig.BrowserRecycleAfterTabs {
				t.Fatalf("RecycleAfterTabs = %d, want %d", got.RecycleAfterTabs, tc.sourceConfig.BrowserRecycleAfterTabs)
			}
			if got.Headless == nil {
				t.Fatal("Headless = nil, want non-nil")
			}
			if *got.Headless != tc.wantHeadless {
				t.Fatalf("Headless = %v, want %v", *got.Headless, tc.wantHeadless)
			}
		})
	}
}

func TestSourceCloseReleasesBrowserRunner(t *testing.T) {
	t.Parallel()

	browser := &closableBrowserRunner{}
	source := newTestSourceWithFallback(FetchModeAuto, browser, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"data":{"klines":[]}}`)),
		}, nil
	})

	if err := source.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if browser.closeCalls != 1 {
		t.Fatalf("closeCalls = %d, want 1", browser.closeCalls)
	}
}

func newTestSource(handle func(req *http.Request) (*http.Response, error)) *Source {
	return newTestSourceWithFallback(FetchModeHTTP, nil, handle)
}

func newTestSourceWithFallback(mode FetchMode, browser browserRunner, handle func(req *http.Request) (*http.Response, error)) *Source {
	sourceConfig := Config{
		Endpoint:        "https://hist.example.com",
		QuoteEndpoint:   "https://quote.example.com",
		TimeoutSeconds:  5,
		MaxRetries:      0,
		UserAgentMode:   "stable",
		FetchMode:       mode,
		BrowserHeadless: true,
	}
	client := newClientWithHTTPClient(buildClientConfig(sourceConfig), &http.Client{
		Transport: roundTripFunc(handle),
	})
	return newSourceWithDependencies(sourceConfig, client, browser)
}

func TestRoundTripFuncImplementsTransport(t *testing.T) {
	t.Parallel()

	var called bool
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("ok")),
		}, nil
	})

	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	resp.Body.Close()
	if !called {
		t.Fatal("transport callback not called")
	}
}

type closableBrowserRunner struct {
	closeCalls int
}

func (r *closableBrowserRunner) FetchCookieHeader(context.Context, string) (string, error) {
	return "sid=1", nil
}

func (r *closableBrowserRunner) Close(context.Context) error {
	r.closeCalls++
	return nil
}
