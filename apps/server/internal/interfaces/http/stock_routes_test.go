package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"log/slog"

	stockdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/stock"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

type fakeStockService struct {
	listResult stockdomain.ListResult
	listErr    error
	stock      stockdomain.Stock
	stockErr   error
	dailyBars  []stockdomain.DailyBar
	dailyErr   error
}

func (s *fakeStockService) ListStocks(ctx context.Context, params stockdomain.ListParams) (stockdomain.ListResult, error) {
	return s.listResult, s.listErr
}

func (s *fakeStockService) GetStock(ctx context.Context, tsCode string) (stockdomain.Stock, error) {
	return s.stock, s.stockErr
}

func (s *fakeStockService) ListDailyBars(ctx context.Context, params stockdomain.DailyParams) ([]stockdomain.DailyBar, error) {
	return s.dailyBars, s.dailyErr
}

func TestListStocksRoute(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithStockService(logger, &fakeStockService{
		listResult: stockdomain.ListResult{
			Items: []stockdomain.Stock{{
				TSCode:   "000001.SZ",
				Symbol:   "000001",
				Name:     "平安银行",
				Industry: "银行",
				Exchange: "SZ",
				IsActive: true,
			}},
			Page:     1,
			PageSize: 20,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/stocks?keyword=%E5%B9%B3%E5%AE%89&page=1&page_size=20", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":0,"errmsg":"","toast":"","data":{"items":[{"ts_code":"000001.SZ","symbol":"000001","name":"平安银行","industry":"银行","exchange":"SZ","is_active":true}],"page":1,"page_size":20}}`)
}

func TestGetStockRoute(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithStockService(logger, &fakeStockService{
		stock: stockdomain.Stock{
			TSCode:   "000001.SZ",
			Symbol:   "000001",
			Name:     "平安银行",
			Industry: "银行",
			Exchange: "SZ",
			IsActive: true,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/stocks/000001.SZ", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":0,"errmsg":"","toast":"","data":{"ts_code":"000001.SZ","symbol":"000001","name":"平安银行","industry":"银行","exchange":"SZ","is_active":true}}`)
}

func TestGetStockNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithStockService(logger, &fakeStockService{
		stockErr: apperror.New(apperror.CodeNotFound, pgx.ErrNoRows),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/stocks/000001.SZ", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":404001,"errmsg":"not found","toast":"数据不存在","data":{}}`)
}

func TestListDailyBarsRoute(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithStockService(logger, &fakeStockService{
		dailyBars: []stockdomain.DailyBar{{
			TSCode:    "000001.SZ",
			TradeDate: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
			Open:      "10.10",
			High:      "10.50",
			Low:       "10.00",
			Close:     "10.40",
			PctChg:    "4.00",
			Vol:       "100000",
			Amount:    "1000000",
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/stocks/000001.SZ/daily?start_date=2026-04-27&end_date=2026-04-28", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":0,"errmsg":"","toast":"","data":[{"ts_code":"000001.SZ","trade_date":"2026-04-27","open":"10.10","high":"10.50","low":"10.00","close":"10.40","pct_chg":"4.00","vol":"100000","amount":"1000000"}]}`)
}

func TestListDailyBarsBadRequest(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithStockService(logger, &fakeStockService{})

	req := httptest.NewRequest(http.MethodGet, "/api/stocks/000001.SZ/daily?start_date=2026-04-28&end_date=2026-04-27", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":400001,"errmsg":"bad request","toast":"请求参数不正确","data":{}}`)
}

func TestStockRoutesNotMountedWithoutService(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithDependencies(logger, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/stocks", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func assertResponseJSON(t *testing.T, recorder *httptest.ResponseRecorder, expected string) {
	t.Helper()

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var gotBody response.Body
	if err := json.Unmarshal(recorder.Body.Bytes(), &gotBody); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	var wantBody response.Body
	if err := json.Unmarshal([]byte(expected), &wantBody); err != nil {
		t.Fatalf("unmarshal expected body: %v", err)
	}

	gotJSON, err := json.Marshal(gotBody)
	if err != nil {
		t.Fatalf("marshal actual body: %v", err)
	}
	wantJSON, err := json.Marshal(wantBody)
	if err != nil {
		t.Fatalf("marshal expected body: %v", err)
	}

	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("response body = %s, want %s", gotJSON, wantJSON)
	}
}
