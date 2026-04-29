package tushare

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSourceWithoutToken(t *testing.T) {
	t.Parallel()

	source := New(" ")

	_, err := source.ListStocks(context.Background())
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListStocks() code = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}

	_, err = source.ListTradeCalendar(context.Background(), "SSE", time.Now(), time.Now())
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListTradeCalendar() code = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}

	_, err = source.ListDailyBars(context.Background(), time.Now(), time.Now())
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListDailyBars() code = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}
}

func TestListStocksCallsTushareAndMapsRows(t *testing.T) {
	t.Parallel()

	source := newTestSource(t, func(request tushareRequest) any {
		if request.APIName != "stock_basic" {
			t.Fatalf("request.APIName = %q, want %q", request.APIName, "stock_basic")
		}
		if request.Token != "configured-token" {
			t.Fatalf("request.Token = %q, want configured token", request.Token)
		}
		if request.Params["list_status"] != "L" {
			t.Fatalf("request.Params[list_status] = %q, want L", request.Params["list_status"])
		}
		if request.Fields != strings.Join(stockBasicFields, ",") {
			t.Fatalf("request.Fields = %q, want %q", request.Fields, strings.Join(stockBasicFields, ","))
		}
		return responseBody([]string{"ts_code", "symbol", "name", "area", "industry", "market", "exchange", "list_date"}, [][]any{
			{"000001.SZ", "000001", "平安银行", "深圳", "银行", "主板", "SZSE", "19910403"},
		})
	})

	items, err := source.ListStocks(context.Background())
	if err != nil {
		t.Fatalf("ListStocks() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
	item := items[0]
	if item.TSCode != "000001.SZ" || item.Symbol != "000001" || item.Name != "平安银行" || item.Source != sourceName {
		t.Fatalf("stock item = %+v, want mapped stock basic", item)
	}
	if got := item.ListDate.Format("2006-01-02"); got != "1991-04-03" {
		t.Fatalf("item.ListDate = %q, want %q", got, "1991-04-03")
	}
}

func TestListTradeCalendarCallsTushareAndMapsRows(t *testing.T) {
	t.Parallel()

	source := newTestSource(t, func(request tushareRequest) any {
		if request.APIName != "trade_cal" {
			t.Fatalf("request.APIName = %q, want %q", request.APIName, "trade_cal")
		}
		if request.Params["exchange"] != "SSE" || request.Params["start_date"] != "20260427" || request.Params["end_date"] != "20260428" {
			t.Fatalf("request.Params = %+v, want exchange/date range", request.Params)
		}
		return responseBody([]string{"exchange", "cal_date", "is_open", "pretrade_date"}, [][]any{
			{"SSE", "20260427", json.Number("1"), "20260424"},
			{"SSE", "20260428", json.Number("0"), "20260427"},
		})
	})

	startDate := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)
	items, err := source.ListTradeCalendar(context.Background(), "SSE", startDate, endDate)
	if err != nil {
		t.Fatalf("ListTradeCalendar() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want %d", len(items), 2)
	}
	if !items[0].IsOpen || items[1].IsOpen {
		t.Fatalf("items open flags = %+v, want first open and second closed", items)
	}
	if got := items[0].PretradeDate.Format("2006-01-02"); got != "2026-04-24" {
		t.Fatalf("items[0].PretradeDate = %q, want %q", got, "2026-04-24")
	}
}

func TestListDailyBarsCallsTushareAndMapsRows(t *testing.T) {
	t.Parallel()

	source := newTestSource(t, func(request tushareRequest) any {
		if request.APIName != "daily" {
			t.Fatalf("request.APIName = %q, want %q", request.APIName, "daily")
		}
		if request.Params["trade_date"] != "20260427" {
			t.Fatalf("request.Params = %+v, want trade_date for single-day sync", request.Params)
		}
		return responseBody([]string{"ts_code", "trade_date", "open", "high", "low", "close", "pre_close", "change", "pct_chg", "vol", "amount"}, [][]any{
			{"000001.SZ", "20260427", json.Number("10.10"), json.Number("10.50"), json.Number("10.00"), json.Number("10.40"), json.Number("10.00"), json.Number("0.40"), json.Number("4.00"), json.Number("100000"), json.Number("1000000")},
		})
	})

	tradeDate := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	items, err := source.ListDailyBars(context.Background(), tradeDate, tradeDate)
	if err != nil {
		t.Fatalf("ListDailyBars() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
	item := items[0]
	if item.TSCode != "000001.SZ" ||
		!item.Close.Equal(decimal.RequireFromString("10.40")) ||
		!item.PctChg.Equal(decimal.RequireFromString("4.00")) ||
		item.Source != sourceName {
		t.Fatalf("daily item = %+v, want mapped daily bar", item)
	}
}

func TestListDailyBarsPreservesCallerCalendarDate(t *testing.T) {
	t.Parallel()

	cst := time.FixedZone("CST", 8*60*60)
	source := newTestSource(t, func(request tushareRequest) any {
		if request.APIName != "daily" {
			t.Fatalf("request.APIName = %q, want %q", request.APIName, "daily")
		}
		if request.Params["trade_date"] != "20260427" {
			t.Fatalf("request.Params = %+v, want local calendar date 20260427", request.Params)
		}
		return responseBody([]string{"ts_code", "trade_date", "open", "high", "low", "close", "pre_close", "change", "pct_chg", "vol", "amount"}, [][]any{
			{"000001.SZ", "20260427", json.Number("10.10"), json.Number("10.50"), json.Number("10.00"), json.Number("10.40"), json.Number("10.00"), json.Number("0.40"), json.Number("4.00"), json.Number("100000"), json.Number("1000000")},
		})
	})

	tradeDate := time.Date(2026, 4, 27, 0, 0, 0, 0, cst)
	if _, err := source.ListDailyBars(context.Background(), tradeDate, tradeDate); err != nil {
		t.Fatalf("ListDailyBars() error = %v", err)
	}
}

func TestTushareBusinessError(t *testing.T) {
	t.Parallel()

	source := newTestSource(t, func(request tushareRequest) any {
		return map[string]any{
			"code": 2002,
			"msg":  "没有权限",
			"data": map[string]any{},
		}
	})

	_, err := source.ListStocks(context.Background())
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListStocks() code = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}
	if err == nil || !strings.Contains(err.Error(), "没有权限") {
		t.Fatalf("ListStocks() error = %v, want tushare message", err)
	}
}

func newTestSource(t *testing.T, handle func(request tushareRequest) any) *Source {
	t.Helper()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("request method = %s, want %s", req.Method, http.MethodPost)
			}
			if got := req.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			var request tushareRequest
			if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			response := handle(request)
			body, err := json.Marshal(response)
			if err != nil {
				t.Fatalf("marshal response body: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
		}),
	}

	return New("configured-token", WithEndpoint("http://example.invalid"), WithHTTPClient(client))
}

func responseBody(fields []string, items [][]any) map[string]any {
	return map[string]any{
		"code": 0,
		"msg":  "",
		"data": map[string]any{
			"fields": fields,
			"items":  items,
		},
	}
}
