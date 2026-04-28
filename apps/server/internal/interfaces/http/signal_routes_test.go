package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	strategydomain "github.com/lifei6671/quantsage/apps/server/internal/domain/strategy"
)

type fakeSignalQueryService struct {
	result strategydomain.QueryResult
	err    error
}

func (s *fakeSignalQueryService) ListSignals(ctx context.Context, params strategydomain.QueryParams) (strategydomain.QueryResult, error) {
	return s.result, s.err
}

func TestListSignalsRoute(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithDependencies(logger, nil, nil, &fakeSignalQueryService{
		result: strategydomain.QueryResult{
			Items: []strategydomain.SignalResult{{
				StrategyCode:          strategydomain.StrategyCodeVolumeBreakout,
				StrategyVersion:       strategydomain.StrategyVersionV1,
				TSCode:                "000001.SZ",
				TradeDate:             time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
				SignalType:            "buy_signal",
				SignalStrength:        decimal.RequireFromString("85"),
				SignalLevel:           "A",
				BuyPriceRef:           decimal.RequireFromString("23.1000"),
				StopLossRef:           decimal.RequireFromString("21.5000"),
				TakeProfitRef:         decimal.RequireFromString("25.8720"),
				InvalidationCondition: "close < ma20",
				Reason:                "放量突破 20 日新高",
			}},
			Page:     1,
			PageSize: 20,
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/signals?trade_date=2026-04-27&strategy_code=volume_breakout_v1&page=1&page_size=20", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":0,"errmsg":"","toast":"","data":{"items":[{"strategy_code":"volume_breakout_v1","strategy_version":"v1","ts_code":"000001.SZ","trade_date":"2026-04-27","signal_type":"buy_signal","signal_strength":"85.0000","signal_level":"A","buy_price_ref":"23.1000","stop_loss_ref":"21.5000","take_profit_ref":"25.8720","invalidation_condition":"close < ma20","reason":"放量突破 20 日新高"}],"page":1,"page_size":20}}`)
}

func TestListSignalsRouteBadRequest(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithDependencies(logger, nil, nil, &fakeSignalQueryService{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/signals?trade_date=2026/04/27&page=1&page_size=20", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assertResponseJSON(t, recorder, `{"code":400001,"errmsg":"bad request","toast":"请求参数不正确","data":{}}`)
}

func TestListSignalsRouteNotMountedWithoutService(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouterWithDependencies(logger, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/signals?trade_date=2026-04-27&page=1&page_size=20", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
