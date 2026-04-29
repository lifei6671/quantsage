package eastmoney

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	datasourceeastmoney "github.com/lifei6671/quantsage/apps/server/internal/domain/datasource/eastmoney"
	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

func TestServiceListKLinesMapsRows(t *testing.T) {
	t.Parallel()

	service := newTestService(func(ctx context.Context, path string, query url.Values) ([]byte, error) {
		if got := query.Get("klt"); got != "5" {
			t.Fatalf("klt = %q, want %q", got, "5")
		}
		if got := query.Get("fqt"); got != "1" {
			t.Fatalf("fqt = %q, want %q", got, "1")
		}
		if got := query.Get("lmt"); got != "2" {
			t.Fatalf("lmt = %q, want %q", got, "2")
		}

		return []byte(`{"rc":0,"data":{"klines":["2026-04-29 09:30,10.10,10.20,10.30,10.00,1000,10000,2.0,1.00,0.10,0.50","2026-04-29 09:35,10.20,10.40,10.50,10.10,1200,13000,3.0,1.96,0.20,0.60"]}}`), nil
	})

	items, err := service.ListKLines(context.Background(), Query{
		TSCode:   "000001.SZ",
		Interval: Interval5Min,
		Adjust:   AdjustQFQ,
		Limit:    2,
		EndTime:  time.Date(2026, 4, 29, 15, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ListKLines() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want %d", len(items), 2)
	}
	if !items[1].Close.Equal(decimal.RequireFromString("10.40")) || !items[1].TurnoverRate.Equal(decimal.RequireFromString("0.60")) {
		t.Fatalf("items[1] = %+v, want mapped minute kline", items[1])
	}
}

func TestServiceListKLinesAutoFallsBackAfterHTMLAntiBot(t *testing.T) {
	t.Parallel()

	requester := &stubHistoryRequester{
		historyResponses: []stubHistoryResponse{
			{body: []byte("<html><body>security check</body></html>")},
		},
		historyHeaderResponses: []stubHistoryResponse{
			{body: []byte(`{"rc":0,"data":{"klines":["2026-04-29,10.10,10.20,10.30,10.00,1000,10000,2.0,1.00,0.10,0.50"]}}`)},
		},
	}
	browser := &stubBrowserCookieRunner{
		cookieHeaders: []stubCookieResponse{
			{cookieHeader: "st_si=browser-cookie"},
		},
	}
	service := newServiceWithClient(datasourceeastmoney.NewHistoryClientWithFallback(requester, browser, datasourceeastmoney.FallbackConfig{
		Mode:         datasourceeastmoney.FetchModeAuto,
		QuotePageURL: "https://quote.eastmoney.com/concept/sh000001.html",
	}))

	items, err := service.ListKLines(context.Background(), Query{
		TSCode:  "000001.SZ",
		Limit:   1,
		EndTime: time.Date(2026, 4, 29, 15, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ListKLines() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
	if !items[0].Close.Equal(decimal.RequireFromString("10.20")) {
		t.Fatalf("items[0].Close = %s, want %s", items[0].Close, decimal.RequireFromString("10.20"))
	}
	if len(requester.historyCalls) != 1 {
		t.Fatalf("plain history calls = %d, want %d", len(requester.historyCalls), 1)
	}
	if len(requester.historyHeaderCalls) != 1 {
		t.Fatalf("history header calls = %d, want %d", len(requester.historyHeaderCalls), 1)
	}
	if got := requester.historyHeaderCalls[0].headers.Get("Cookie"); got != "st_si=browser-cookie" {
		t.Fatalf("history header Cookie = %q, want %q", got, "st_si=browser-cookie")
	}
	if len(browser.pageURLs) != 1 {
		t.Fatalf("browser calls = %d, want %d", len(browser.pageURLs), 1)
	}
}

func TestNewFromClientConfigUsesHTTPOnlyHistoryClient(t *testing.T) {
	t.Parallel()

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>security check</body></html>"))
	}))
	t.Cleanup(server.Close)

	service := NewFromClientConfig(datasourceeastmoney.ClientConfig{
		Endpoint:      server.URL,
		QuoteEndpoint: server.URL,
		Timeout:       5 * time.Second,
		MaxRetries:    0,
		UserAgentMode: "stable",
	})

	_, err := service.ListKLines(context.Background(), Query{
		TSCode:  "000001.SZ",
		Limit:   1,
		EndTime: time.Date(2026, 4, 29, 15, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("ListKLines() error = nil, want non-nil because HTTP-only path should not browser-fallback")
	}
	if !strings.Contains(err.Error(), "html or anti-bot response") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "html or anti-bot response")
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want %d because HTTP-only constructor should not trigger retry fallback", requests, 1)
	}
}

func TestGetLatestKLineReturnsNotFoundOnEmptyData(t *testing.T) {
	t.Parallel()

	service := newTestService(func(ctx context.Context, path string, query url.Values) ([]byte, error) {
		return []byte(`{"rc":0,"data":{"klines":[]}}`), nil
	})

	_, err := service.GetLatestKLine(context.Background(), Query{TSCode: "000001.SZ"})
	if err == nil {
		t.Fatal("GetLatestKLine() error = nil, want non-nil")
	}
	if apperror.CodeOf(err) != apperror.CodeNotFound {
		t.Fatalf("CodeOf(error) = %d, want %d", apperror.CodeOf(err), apperror.CodeNotFound)
	}
}

func TestBatchListKLinesReturnsPartialResults(t *testing.T) {
	t.Parallel()

	service := newTestService(func(ctx context.Context, path string, query url.Values) ([]byte, error) {
		switch query.Get("secid") {
		case "0.000001":
			return []byte(`{"rc":0,"data":{"klines":["2026-04-29,10.10,10.20,10.30,10.00,1000,10000,2.0,1.00,0.10,0.50"]}}`), nil
		case "1.600000":
			return []byte(`{"rc":2,"message":"busy"}`), nil
		default:
			t.Fatalf("unexpected secid %q", query.Get("secid"))
			return nil, nil
		}
	})

	result, err := service.BatchListKLines(context.Background(), []Query{
		{TSCode: "000001.SZ"},
		{TSCode: "600000.SH"},
	})
	if err == nil {
		t.Fatal("BatchListKLines() error = nil, want partial failure")
	}
	if len(result["000001.SZ"]) != 1 {
		t.Fatalf("len(result[000001.SZ]) = %d, want %d", len(result["000001.SZ"]), 1)
	}
	if _, ok := result["600000.SH"]; ok {
		t.Fatalf("result[600000.SH] exists, want omitted on failure: %+v", result["600000.SH"])
	}
}

func TestBatchListKLinesRejectsDuplicateTSCode(t *testing.T) {
	t.Parallel()

	service := newTestService(func(ctx context.Context, path string, query url.Values) ([]byte, error) {
		t.Fatal("ListKLines transport should not be called when batch input is invalid")
		return nil, nil
	})

	_, err := service.BatchListKLines(context.Background(), []Query{
		{TSCode: "000001.SZ", Interval: IntervalDay},
		{TSCode: "000001.SZ", Interval: Interval5Min},
	})
	if err == nil {
		t.Fatal("BatchListKLines() error = nil, want duplicate ts_code error")
	}
	if apperror.CodeOf(err) != apperror.CodeBadRequest {
		t.Fatalf("CodeOf(error) = %d, want %d", apperror.CodeOf(err), apperror.CodeBadRequest)
	}
	if !strings.Contains(err.Error(), "duplicate ts_code") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "duplicate ts_code")
	}
}

func TestAttachSimpleMovingAveragesSkipsInsufficientWindow(t *testing.T) {
	t.Parallel()

	items := []KLine{
		{Close: decimal.RequireFromString("10")},
		{Close: decimal.RequireFromString("11")},
		{Close: decimal.RequireFromString("12")},
	}

	enriched := AttachSimpleMovingAverages(items, []int{2, 5})
	if len(enriched[0].MovingAverages) != 0 {
		t.Fatalf("len(enriched[0].MovingAverages) = %d, want %d", len(enriched[0].MovingAverages), 0)
	}
	if len(enriched[2].MovingAverages) != 1 {
		t.Fatalf("len(enriched[2].MovingAverages) = %d, want %d", len(enriched[2].MovingAverages), 1)
	}
	if enriched[2].MovingAverages[0].Period != 2 || !enriched[2].MovingAverages[0].Value.Equal(decimal.RequireFromString("11.5")) {
		t.Fatalf("enriched[2].MovingAverages[0] = %+v, want MA2=11.5", enriched[2].MovingAverages[0])
	}
}

func TestAggregateKLines(t *testing.T) {
	t.Parallel()

	items := []KLine{
		{
			TSCode:       "000001.SZ",
			TradeTime:    time.Date(2026, 4, 29, 9, 30, 0, 0, time.UTC),
			Open:         decimal.RequireFromString("10.00"),
			High:         decimal.RequireFromString("10.20"),
			Low:          decimal.RequireFromString("9.90"),
			Close:        decimal.RequireFromString("10.10"),
			PreClose:     decimal.RequireFromString("9.80"),
			Vol:          decimal.RequireFromString("100"),
			Amount:       decimal.RequireFromString("1000"),
			TurnoverRate: decimal.RequireFromString("0.1"),
		},
		{
			TSCode:       "000001.SZ",
			TradeTime:    time.Date(2026, 4, 29, 9, 35, 0, 0, time.UTC),
			Open:         decimal.RequireFromString("10.10"),
			High:         decimal.RequireFromString("10.30"),
			Low:          decimal.RequireFromString("10.00"),
			Close:        decimal.RequireFromString("10.20"),
			PreClose:     decimal.RequireFromString("10.10"),
			Vol:          decimal.RequireFromString("120"),
			Amount:       decimal.RequireFromString("1200"),
			TurnoverRate: decimal.RequireFromString("0.2"),
		},
		{
			TSCode:       "000001.SZ",
			TradeTime:    time.Date(2026, 4, 29, 9, 40, 0, 0, time.UTC),
			Open:         decimal.RequireFromString("10.20"),
			High:         decimal.RequireFromString("10.40"),
			Low:          decimal.RequireFromString("10.05"),
			Close:        decimal.RequireFromString("10.35"),
			PreClose:     decimal.RequireFromString("10.20"),
			Vol:          decimal.RequireFromString("150"),
			Amount:       decimal.RequireFromString("1600"),
			TurnoverRate: decimal.RequireFromString("0.3"),
		},
	}

	aggregated, err := AggregateKLines(items, 3)
	if err != nil {
		t.Fatalf("AggregateKLines() error = %v", err)
	}
	if len(aggregated) != 1 {
		t.Fatalf("len(aggregated) = %d, want %d", len(aggregated), 1)
	}
	item := aggregated[0]
	if !item.Open.Equal(decimal.RequireFromString("10.00")) ||
		!item.Close.Equal(decimal.RequireFromString("10.35")) ||
		!item.High.Equal(decimal.RequireFromString("10.40")) ||
		!item.Low.Equal(decimal.RequireFromString("9.90")) ||
		!item.Vol.Equal(decimal.RequireFromString("370")) ||
		!item.Amount.Equal(decimal.RequireFromString("3800")) {
		t.Fatalf("aggregated item = %+v, want merged values", item)
	}
}

func TestNormalizeQueryRejectsMissingTSCode(t *testing.T) {
	t.Parallel()

	_, err := normalizeQuery(Query{})
	if err == nil {
		t.Fatal("normalizeQuery() error = nil, want non-nil")
	}
	if apperror.CodeOf(err) != apperror.CodeBadRequest {
		t.Fatalf("CodeOf(error) = %d, want %d", apperror.CodeOf(err), apperror.CodeBadRequest)
	}
}

func TestBatchListKLinesErrorContainsTSCode(t *testing.T) {
	t.Parallel()

	service := newTestService(func(ctx context.Context, path string, query url.Values) ([]byte, error) {
		return []byte(`{"rc":2,"message":"busy"}`), nil
	})

	_, err := service.BatchListKLines(context.Background(), []Query{{TSCode: "000001.SZ"}})
	if err == nil {
		t.Fatal("BatchListKLines() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "000001.SZ") {
		t.Fatalf("error = %q, want contains TSCode", err.Error())
	}
}

type fakeHistoryClient func(ctx context.Context, path string, query url.Values) ([]byte, error)

func (fn fakeHistoryClient) GetHistory(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return fn(ctx, path, query)
}

func newTestService(handle func(ctx context.Context, path string, query url.Values) ([]byte, error)) Service {
	return newServiceWithClient(fakeHistoryClient(handle))
}

func TestServicePropagatesClientError(t *testing.T) {
	t.Parallel()

	service := newTestService(func(ctx context.Context, path string, query url.Values) ([]byte, error) {
		return nil, fmt.Errorf("upstream unavailable")
	})

	_, err := service.ListKLines(context.Background(), Query{TSCode: "000001.SZ"})
	if err == nil {
		t.Fatal("ListKLines() error = nil, want non-nil")
	}
}

type stubHistoryResponse struct {
	body []byte
	err  error
}

type stubHistoryCall struct {
	path    string
	query   url.Values
	headers http.Header
}

type stubHistoryRequester struct {
	historyResponses       []stubHistoryResponse
	historyHeaderResponses []stubHistoryResponse
	historyCalls           []stubHistoryCall
	historyHeaderCalls     []stubHistoryCall
}

func (s *stubHistoryRequester) GetHistory(_ context.Context, path string, query url.Values) ([]byte, error) {
	s.historyCalls = append(s.historyCalls, stubHistoryCall{path: path, query: query})
	if len(s.historyResponses) == 0 {
		return nil, nil
	}

	response := s.historyResponses[0]
	s.historyResponses = s.historyResponses[1:]
	return response.body, response.err
}

func (s *stubHistoryRequester) GetHistoryWithHeaders(_ context.Context, path string, query url.Values, headers http.Header) ([]byte, error) {
	s.historyHeaderCalls = append(s.historyHeaderCalls, stubHistoryCall{
		path:    path,
		query:   query,
		headers: headers.Clone(),
	})
	if len(s.historyHeaderResponses) == 0 {
		return nil, nil
	}

	response := s.historyHeaderResponses[0]
	s.historyHeaderResponses = s.historyHeaderResponses[1:]
	return response.body, response.err
}

type stubCookieResponse struct {
	cookieHeader string
	err          error
}

type stubBrowserCookieRunner struct {
	cookieHeaders []stubCookieResponse
	pageURLs      []string
}

func (s *stubBrowserCookieRunner) FetchCookieHeader(_ context.Context, pageURL string) (string, error) {
	s.pageURLs = append(s.pageURLs, pageURL)
	if len(s.cookieHeaders) == 0 {
		return "", nil
	}

	response := s.cookieHeaders[0]
	s.cookieHeaders = s.cookieHeaders[1:]
	return response.cookieHeader, response.err
}
