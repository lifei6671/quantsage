package app

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
	"github.com/lifei6671/quantsage/apps/server/internal/domain/indicator"
	jobdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/job"
	"github.com/lifei6671/quantsage/apps/server/internal/domain/marketdata"
	strategydomain "github.com/lifei6671/quantsage/apps/server/internal/domain/strategy"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/consts"
)

func TestUpsertStockDailyMergesHistory(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	ctx := context.Background()
	if err := store.UpsertStockDaily(ctx, []datasource.DailyBar{
		buildDatasourceBar("000001.SZ", "2026-04-27", "10.10", "100000"),
		buildDatasourceBar("000001.SZ", "2026-04-28", "10.20", "110000"),
	}); err != nil {
		t.Fatalf("UpsertStockDaily() first call error = %v", err)
	}

	if err := store.UpsertStockDaily(ctx, []datasource.DailyBar{
		buildDatasourceBar("000001.SZ", "2026-04-28", "10.25", "120000"),
		buildDatasourceBar("000001.SZ", "2026-04-29", "10.30", "130000"),
	}); err != nil {
		t.Fatalf("UpsertStockDaily() second call error = %v", err)
	}

	items, err := store.ListStockDaily(
		ctx,
		time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("ListStockDaily() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want %d", len(items), 3)
	}
	if !items[1].Close.Equal(decimal.RequireFromString("10.25")) {
		t.Fatalf("items[1].Close = %s, want %s", items[1].Close, "10.25")
	}
}

func TestSampleRuntimeUsesImportSourceForSyncJobs(t *testing.T) {
	t.Parallel()

	importSource := &noopSampleSource{
		stocks: []datasource.StockBasic{
			{
				TSCode:   "688001.SH",
				Symbol:   "688001",
				Name:     "测试股票",
				Industry: "软件服务",
				Exchange: "SSE",
				ListDate: time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC),
				Source:   consts.DatasourceTushare,
			},
		},
	}
	runtime, err := NewSampleRuntimeWithImportSource("../../testdata/sample", importSource)
	if err != nil {
		t.Fatalf("NewSampleRuntimeWithImportSource() error = %v", err)
	}

	if err := runtime.Runner().Run(context.Background(), "sync_stock_basic", time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Run(sync_stock_basic) error = %v", err)
	}

	stock, err := runtime.StockService().GetStock(context.Background(), "688001.SH")
	if err != nil {
		t.Fatalf("GetStock() error = %v", err)
	}
	if stock.Name != "测试股票" || stock.Exchange != "SSE" {
		t.Fatalf("stock = %+v, want import source stock", stock)
	}
}

func TestSampleRuntimeCloseClosesImportSource(t *testing.T) {
	t.Parallel()

	importSource := &noopSampleSource{}
	runtime, err := NewSampleRuntimeWithImportSource("../../testdata/sample", importSource)
	if err != nil {
		t.Fatalf("NewSampleRuntimeWithImportSource() error = %v", err)
	}

	runtime.Close()
	if importSource.closeCalls != 1 {
		t.Fatalf("closeCalls = %d, want 1", importSource.closeCalls)
	}
}

func TestListStockDailyUsesTradeDateGranularity(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	ctx := context.Background()
	if err := store.UpsertStockDaily(ctx, []datasource.DailyBar{
		buildDatasourceBar("000001.SZ", "2026-04-27", "10.10", "100000"),
		buildDatasourceBar("000001.SZ", "2026-04-28", "10.20", "110000"),
	}); err != nil {
		t.Fatalf("UpsertStockDaily() error = %v", err)
	}

	items, err := store.ListStockDaily(
		ctx,
		time.Date(2026, 4, 27, 18, 5, 0, 0, time.UTC),
		time.Date(2026, 4, 27, 18, 5, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("ListStockDaily() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
}

func TestListStrategyContextsBuildsFromStore(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	ctx := context.Background()
	if err := store.UpsertStockBasics(ctx, []datasource.StockBasic{{
		TSCode: "688001.SH",
		Name:   "示例科技",
	}}); err != nil {
		t.Fatalf("UpsertStockBasics() error = %v", err)
	}

	inputBars := make([]marketdata.DailyBar, 0, 21)
	sourceBars := make([]datasource.DailyBar, 0, 21)
	for i := 0; i < 21; i++ {
		tradeDate := time.Date(2026, 4, 7+i, 0, 0, 0, 0, time.UTC)
		closePrice := decimal.NewFromInt(10 + int64(i))
		bar := marketdata.DailyBar{
			TSCode:    "688001.SH",
			TradeDate: tradeDate,
			Open:      closePrice.Sub(decimal.RequireFromString("0.1")),
			High:      closePrice.Add(decimal.RequireFromString("0.2")),
			Low:       closePrice.Sub(decimal.RequireFromString("0.2")),
			Close:     closePrice,
			PreClose:  closePrice.Sub(decimal.RequireFromString("0.1")),
			Change:    decimal.RequireFromString("0.1"),
			PctChg:    decimal.RequireFromString("1.0"),
			Vol:       decimal.RequireFromString("100000"),
			Amount:    decimal.RequireFromString("1000000"),
			Source:    "sample",
		}
		inputBars = append(inputBars, bar)
		sourceBars = append(sourceBars, datasource.DailyBar{
			TSCode:    bar.TSCode,
			TradeDate: bar.TradeDate,
			Open:      bar.Open,
			High:      bar.High,
			Low:       bar.Low,
			Close:     bar.Close,
			PreClose:  bar.PreClose,
			Change:    bar.Change,
			PctChg:    bar.PctChg,
			Vol:       bar.Vol,
			Amount:    bar.Amount,
			Source:    bar.Source,
		})
	}

	if err := store.UpsertStockDaily(ctx, sourceBars); err != nil {
		t.Fatalf("UpsertStockDaily() error = %v", err)
	}

	factors, err := indicator.CalculateDailyFactors(inputBars)
	if err != nil {
		t.Fatalf("CalculateDailyFactors() error = %v", err)
	}
	if err := store.UpsertDailyFactors(ctx, factors); err != nil {
		t.Fatalf("UpsertDailyFactors() error = %v", err)
	}

	contexts, err := store.ListStrategyContexts(
		ctx,
		time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("ListStrategyContexts() error = %v", err)
	}
	if len(contexts) != 1 {
		t.Fatalf("len(contexts) = %d, want %d", len(contexts), 1)
	}
	if contexts[0].CurrentBar.TSCode != "688001.SH" {
		t.Fatalf("contexts[0].CurrentBar.TSCode = %q, want %q", contexts[0].CurrentBar.TSCode, "688001.SH")
	}
	if len(contexts[0].RecentBars) != 21 {
		t.Fatalf("len(contexts[0].RecentBars) = %d, want %d", len(contexts[0].RecentBars), 21)
	}
	if contexts[0].CurrentFactor.MA20 == nil {
		t.Fatal("contexts[0].CurrentFactor.MA20 = nil, want non-nil")
	}
}

func TestListStrategyContextsUsesTradeDateGranularity(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	ctx := context.Background()
	inputBars := make([]marketdata.DailyBar, 0, 21)
	sourceBars := make([]datasource.DailyBar, 0, 21)
	for i := 0; i < 21; i++ {
		tradeDate := time.Date(2026, 4, 7+i, 0, 0, 0, 0, time.UTC)
		closePrice := decimal.NewFromInt(10 + int64(i))
		bar := marketdata.DailyBar{
			TSCode:    "688001.SH",
			TradeDate: tradeDate,
			Open:      closePrice.Sub(decimal.RequireFromString("0.1")),
			High:      closePrice.Add(decimal.RequireFromString("0.2")),
			Low:       closePrice.Sub(decimal.RequireFromString("0.2")),
			Close:     closePrice,
			PreClose:  closePrice.Sub(decimal.RequireFromString("0.1")),
			Change:    decimal.RequireFromString("0.1"),
			PctChg:    decimal.RequireFromString("1.0"),
			Vol:       decimal.RequireFromString("100000"),
			Amount:    decimal.RequireFromString("1000000"),
			Source:    "sample",
		}
		inputBars = append(inputBars, bar)
		sourceBars = append(sourceBars, datasource.DailyBar{
			TSCode:    bar.TSCode,
			TradeDate: bar.TradeDate,
			Open:      bar.Open,
			High:      bar.High,
			Low:       bar.Low,
			Close:     bar.Close,
			PreClose:  bar.PreClose,
			Change:    bar.Change,
			PctChg:    bar.PctChg,
			Vol:       bar.Vol,
			Amount:    bar.Amount,
			Source:    bar.Source,
		})
	}

	if err := store.UpsertStockDaily(ctx, sourceBars); err != nil {
		t.Fatalf("UpsertStockDaily() error = %v", err)
	}
	factors, err := indicator.CalculateDailyFactors(inputBars)
	if err != nil {
		t.Fatalf("CalculateDailyFactors() error = %v", err)
	}
	if err := store.UpsertDailyFactors(ctx, factors); err != nil {
		t.Fatalf("UpsertDailyFactors() error = %v", err)
	}

	contexts, err := store.ListStrategyContexts(
		ctx,
		time.Date(2026, 4, 27, 18, 10, 0, 0, time.UTC),
		time.Date(2026, 4, 27, 18, 10, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("ListStrategyContexts() error = %v", err)
	}
	if len(contexts) != 1 {
		t.Fatalf("len(contexts) = %d, want %d", len(contexts), 1)
	}
}

func TestSampleRuntimeGeneratesSignalsFromSampleData(t *testing.T) {
	t.Parallel()

	runtime, err := NewSampleRuntime("../../testdata/sample")
	if err != nil {
		t.Fatalf("NewSampleRuntime() error = %v", err)
	}

	bizDate := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	if err := runtime.Runner().Run(context.Background(), "generate_strategy_signals", bizDate); err != nil {
		t.Fatalf("Run(generate_strategy_signals) error = %v", err)
	}

	result, err := runtime.SignalQueryService().ListSignals(context.Background(), strategydomain.QueryParams{
		TradeDate: bizDate,
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("ListSignals() error = %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("len(result.Items) = %d, want %d", len(result.Items), 2)
	}

	got := map[string]string{}
	for _, item := range result.Items {
		got[item.StrategyCode] = item.TSCode
	}
	if got[strategydomain.StrategyCodeVolumeBreakout] != "000001.SZ" {
		t.Fatalf("volume breakout ts_code = %q, want %q", got[strategydomain.StrategyCodeVolumeBreakout], "000001.SZ")
	}
	if got[strategydomain.StrategyCodeTrendBreak] != "300750.SZ" {
		t.Fatalf("trend break ts_code = %q, want %q", got[strategydomain.StrategyCodeTrendBreak], "300750.SZ")
	}
}

func TestCalcDailyFactorJobUsesFullHistoryLookback(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	ctx := context.Background()
	sourceBars := make([]datasource.DailyBar, 0, 90)
	for i := 0; i < 90; i++ {
		tradeDate := time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC)
		closePrice := decimal.NewFromInt(10 + int64(i))
		sourceBars = append(sourceBars, datasource.DailyBar{
			TSCode:    "000001.SZ",
			TradeDate: tradeDate,
			Open:      closePrice.Sub(decimal.RequireFromString("0.1")),
			High:      closePrice.Add(decimal.RequireFromString("0.2")),
			Low:       closePrice.Sub(decimal.RequireFromString("0.2")),
			Close:     closePrice,
			PreClose:  closePrice.Sub(decimal.RequireFromString("0.1")),
			Change:    decimal.RequireFromString("0.1"),
			PctChg:    decimal.RequireFromString("1.0"),
			Vol:       decimal.RequireFromString("100000"),
			Amount:    decimal.RequireFromString("1000000"),
			Source:    "sample",
		})
	}
	if err := store.UpsertStockDaily(ctx, sourceBars); err != nil {
		t.Fatalf("UpsertStockDaily() error = %v", err)
	}

	fullBars, err := store.ListStockDaily(ctx, time.Time{}, sourceBars[len(sourceBars)-1].TradeDate)
	if err != nil {
		t.Fatalf("ListStockDaily() error = %v", err)
	}
	fullFactors, err := indicator.CalculateDailyFactors(fullBars)
	if err != nil {
		t.Fatalf("CalculateDailyFactors() error = %v", err)
	}
	if err := store.UpsertDailyFactors(ctx, fullFactors); err != nil {
		t.Fatalf("UpsertDailyFactors() error = %v", err)
	}

	runtime := &SampleRuntime{
		source: &noopSampleSource{},
		store:  store,
		runner: jobdomain.NewRegistry(),
	}
	if err := runtime.registerJobs(); err != nil {
		t.Fatalf("registerJobs() error = %v", err)
	}
	if err := runtime.seedPreloadedJobState(); err != nil {
		t.Fatalf("seedPreloadedJobState() error = %v", err)
	}

	bizDate := sourceBars[len(sourceBars)-1].TradeDate
	if err := runtime.Runner().Run(ctx, "calc_daily_factor", bizDate); err != nil {
		t.Fatalf("Run(calc_daily_factor) error = %v", err)
	}

	targetDate := bizDate.AddDate(0, 0, -30)
	var targetFactor *indicator.DailyFactor
	for index := range store.factors["000001.SZ"] {
		item := &store.factors["000001.SZ"][index]
		if item.TradeDate.Equal(targetDate) {
			targetFactor = item
			break
		}
	}
	if targetFactor == nil {
		t.Fatalf("factor for %s not found", targetDate.Format("2006-01-02"))
	}
	if targetFactor.MA60 == nil {
		t.Fatalf("targetFactor.MA60 = nil, want non-nil for %s", targetDate.Format("2006-01-02"))
	}
}

func TestReplaceStrategySignalsDropsStaleSignals(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	ctx := context.Background()
	store.signals = []strategydomain.SignalResult{
		{
			StrategyCode: strategydomain.StrategyCodeVolumeBreakout,
			TSCode:       "000001.SZ",
			TradeDate:    time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			StrategyCode: strategydomain.StrategyCodeTrendBreak,
			TSCode:       "300750.SZ",
			TradeDate:    time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC),
		},
	}

	if err := store.ReplaceStrategySignals(ctx,
		time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		nil,
	); err != nil {
		t.Fatalf("ReplaceStrategySignals() error = %v", err)
	}

	result, err := store.ListSignals(ctx, strategydomain.QueryParams{
		TradeDate: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("ListSignals() error = %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("len(result) = %d, want %d", len(result), 0)
	}

	nextDayResult, err := store.ListSignals(ctx, strategydomain.QueryParams{
		TradeDate: time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC),
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("ListSignals() next day error = %v", err)
	}
	if len(nextDayResult) != 1 {
		t.Fatalf("len(nextDayResult) = %d, want %d", len(nextDayResult), 1)
	}
}

func TestStartRejectsConcurrentRunOnSameJobAndDate(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	ctx := context.Background()
	bizDate := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	if err := store.Start(ctx, "sync_daily_market", bizDate); err != nil {
		t.Fatalf("Start() first call error = %v", err)
	}

	err := store.Start(ctx, "sync_daily_market", bizDate)
	if err == nil {
		t.Fatal("Start() second call error = nil, want non-nil")
	}
}

type noopSampleSource struct {
	stocks     []datasource.StockBasic
	days       []datasource.TradeDay
	bars       []datasource.DailyBar
	closeCalls int
}

func (s *noopSampleSource) ListStocks(ctx context.Context) ([]datasource.StockBasic, error) {
	return append([]datasource.StockBasic(nil), s.stocks...), nil
}

func (s *noopSampleSource) ListTradeCalendar(ctx context.Context, exchange string, startDate, endDate time.Time) ([]datasource.TradeDay, error) {
	return append([]datasource.TradeDay(nil), s.days...), nil
}

func (s *noopSampleSource) ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]datasource.DailyBar, error) {
	return append([]datasource.DailyBar(nil), s.bars...), nil
}

func (s *noopSampleSource) Close(context.Context) error {
	s.closeCalls++
	return nil
}

func buildDatasourceBar(tsCode, tradeDate, closePrice, volume string) datasource.DailyBar {
	dateValue, err := time.Parse("2006-01-02", tradeDate)
	if err != nil {
		panic(err)
	}

	closeValue := decimal.RequireFromString(closePrice)
	return datasource.DailyBar{
		TSCode:    tsCode,
		TradeDate: dateValue,
		Open:      closeValue.Sub(decimal.RequireFromString("0.1")),
		High:      closeValue.Add(decimal.RequireFromString("0.2")),
		Low:       closeValue.Sub(decimal.RequireFromString("0.2")),
		Close:     closeValue,
		PreClose:  closeValue.Sub(decimal.RequireFromString("0.1")),
		Change:    decimal.RequireFromString("0.1"),
		PctChg:    decimal.RequireFromString("1.0"),
		Vol:       decimal.RequireFromString(volume),
		Amount:    decimal.RequireFromString("1000000"),
		Source:    "sample",
	}
}
