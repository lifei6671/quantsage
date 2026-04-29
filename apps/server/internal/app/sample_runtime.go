package app

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
	sampleds "github.com/lifei6671/quantsage/apps/server/internal/domain/datasource/sample"
	"github.com/lifei6671/quantsage/apps/server/internal/domain/indicator"
	jobdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/job"
	"github.com/lifei6671/quantsage/apps/server/internal/domain/marketdata"
	stockdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/stock"
	strategydomain "github.com/lifei6671/quantsage/apps/server/internal/domain/strategy"
)

// SampleRuntime 封装本地 sample 模式下的运行时依赖。
type SampleRuntime struct {
	source       datasource.Source
	store        *memoryStore
	runner       *jobdomain.Registry
	stockService stockdomain.Service
	signalQuery  strategydomain.QueryService
	jobQuery     jobdomain.QueryService
}

type closeableDatasource interface {
	Close(context.Context) error
}

// NewSampleRuntime 创建 sample + 内存态运行时。
func NewSampleRuntime(sampleDir string) (*SampleRuntime, error) {
	return NewSampleRuntimeWithImportSource(sampleDir, nil)
}

// NewSampleRuntimeWithImportSource 创建 sample 预加载运行时，并允许导入任务使用外部数据源。
func NewSampleRuntimeWithImportSource(sampleDir string, importSource datasource.Source) (*SampleRuntime, error) {
	resolvedDir, err := resolveSampleDataDir(sampleDir)
	if err != nil {
		return nil, err
	}

	source := sampleds.New(resolvedDir)
	if importSource == nil {
		importSource = source
	}
	store := newMemoryStore()
	if err := preloadSampleData(context.Background(), source, store); err != nil {
		return nil, err
	}

	runner := jobdomain.NewRegistry()
	runtime := &SampleRuntime{
		source:       importSource,
		store:        store,
		runner:       runner,
		stockService: &sampleStockService{store: store},
		signalQuery:  strategydomain.NewQueryService(store),
		jobQuery:     jobdomain.NewQueryService(store),
	}

	if err := runtime.registerJobs(); err != nil {
		return nil, err
	}
	if err := runtime.seedPreloadedJobState(); err != nil {
		return nil, err
	}

	return runtime, nil
}

// StockService 返回股票查询服务。
func (r *SampleRuntime) StockService() stockdomain.Service {
	return r.stockService
}

// SignalQueryService 返回信号查询服务。
func (r *SampleRuntime) SignalQueryService() strategydomain.QueryService {
	return r.signalQuery
}

// JobQueryService 返回任务记录查询服务。
func (r *SampleRuntime) JobQueryService() jobdomain.QueryService {
	return r.jobQuery
}

// Close 释放 sample runtime 持有的外部数据源资源。
func (r *SampleRuntime) Close() {
	if r == nil || r.source == nil {
		return
	}
	closeable, ok := r.source.(closeableDatasource)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	closeErr := closeable.Close(ctx)
	if closeErr != nil {
		return
	}
}

// Runner 返回任务执行器。
func (r *SampleRuntime) Runner() jobdomain.Runner {
	return r.runner
}

func (r *SampleRuntime) registerJobs() error {
	if err := r.runner.Register("sync_stock_basic", func(ctx context.Context, bizDate time.Time) error {
		return jobdomain.ImportStockBasic(ctx, r.store, r.store, r.source, bizDate)
	}); err != nil {
		return err
	}
	if err := r.runner.Register("sync_trade_calendar", func(ctx context.Context, bizDate time.Time) error {
		return jobdomain.ImportTradeCalendar(ctx, r.store, r.store, r.source, "SSE", bizDate, bizDate, bizDate)
	}); err != nil {
		return err
	}
	if err := r.runner.Register("sync_daily_market", func(ctx context.Context, bizDate time.Time) error {
		return jobdomain.ImportStockDaily(ctx, r.store, r.store, r.source, bizDate, bizDate, bizDate)
	}); err != nil {
		return err
	}
	if err := r.runner.Register(
		"calc_daily_factor",
		func(ctx context.Context, bizDate time.Time) error {
			return jobdomain.CalcDailyFactor(ctx, r.store, r.store, r.store, time.Time{}, bizDate, bizDate)
		},
		jobdomain.WithDependencies("sync_daily_market"),
	); err != nil {
		return err
	}
	if err := r.runner.Register(
		"generate_strategy_signals",
		func(ctx context.Context, bizDate time.Time) error {
			return jobdomain.GenerateStrategySignals(ctx, r.store, r.store, r.store, bizDate, bizDate, bizDate)
		},
		jobdomain.WithDependencies("sync_daily_market", "calc_daily_factor"),
	); err != nil {
		return err
	}

	return nil
}

func (r *SampleRuntime) seedPreloadedJobState() error {
	for _, tradeDate := range r.store.tradeDatesWithDailyBars() {
		if err := r.runner.MarkCompleted("sync_daily_market", tradeDate); err != nil {
			return err
		}
	}
	for _, tradeDate := range r.store.tradeDatesWithFactors() {
		if err := r.runner.MarkCompleted("calc_daily_factor", tradeDate); err != nil {
			return err
		}
	}

	return nil
}

func preloadSampleData(ctx context.Context, source datasource.Source, store *memoryStore) error {
	stocks, err := source.ListStocks(ctx)
	if err != nil {
		return fmt.Errorf("preload sample stocks: %w", err)
	}
	if err := store.UpsertStockBasics(ctx, stocks); err != nil {
		return fmt.Errorf("preload sample stock basics: %w", err)
	}

	startDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	dailyBars, err := source.ListDailyBars(ctx, startDate, endDate)
	if err != nil {
		return fmt.Errorf("preload sample daily bars: %w", err)
	}
	if err := store.UpsertStockDaily(ctx, dailyBars); err != nil {
		return fmt.Errorf("preload sample stock daily: %w", err)
	}
	if err := preloadSampleFactors(ctx, store, startDate, endDate); err != nil {
		return fmt.Errorf("preload sample factors: %w", err)
	}

	return nil
}

func preloadSampleFactors(ctx context.Context, store *memoryStore, startDate, endDate time.Time) error {
	bars, err := store.ListStockDaily(ctx, startDate, endDate)
	if err != nil {
		return fmt.Errorf("list stock daily for preload factors: %w", err)
	}

	grouped := groupDailyBarsByTSCode(bars)
	factors := make([]indicator.DailyFactor, 0, len(bars))
	for _, tsCode := range sortedDailyBarKeys(grouped) {
		items := grouped[tsCode]
		dailyFactors, calcErr := indicator.CalculateDailyFactors(items)
		if calcErr != nil {
			return fmt.Errorf("calculate preload factors for %s: %w", tsCode, calcErr)
		}
		factors = append(factors, dailyFactors...)
	}
	if err := store.UpsertDailyFactors(ctx, factors); err != nil {
		return fmt.Errorf("upsert preload factors: %w", err)
	}

	return nil
}

func groupDailyBarsByTSCode(items []marketdata.DailyBar) map[string][]marketdata.DailyBar {
	grouped := make(map[string][]marketdata.DailyBar, len(items))
	for _, item := range items {
		grouped[item.TSCode] = append(grouped[item.TSCode], item)
	}

	return grouped
}

func sortedDailyBarKeys(grouped map[string][]marketdata.DailyBar) []string {
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}

func collectTradeDatesFromDailyBars(grouped map[string][]marketdata.DailyBar) []time.Time {
	seen := make(map[string]time.Time)
	for _, bars := range grouped {
		for _, item := range bars {
			key := dateKey(item.TradeDate)
			seen[key] = dateOnly(item.TradeDate)
		}
	}

	return sortedTradeDates(seen)
}

func collectTradeDatesFromFactors(grouped map[string][]indicator.DailyFactor) []time.Time {
	seen := make(map[string]time.Time)
	for _, factors := range grouped {
		for _, item := range factors {
			key := dateKey(item.TradeDate)
			seen[key] = dateOnly(item.TradeDate)
		}
	}

	return sortedTradeDates(seen)
}

func sortedTradeDates(items map[string]time.Time) []time.Time {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]time.Time, 0, len(keys))
	for _, key := range keys {
		result = append(result, items[key])
	}

	return result
}

func resolveSampleDataDir(sampleDir string) (string, error) {
	candidates := []string{
		sampleDir,
		"apps/server/testdata/sample",
		"testdata/sample",
		"../testdata/sample",
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("resolve sample data dir: no candidate exists in %v", candidates)
}

type sampleStockService struct {
	store *memoryStore
}

func (s *sampleStockService) ListStocks(ctx context.Context, params stockdomain.ListParams) (stockdomain.ListResult, error) {
	_ = ctx
	return s.store.listStocks(params), nil
}

func (s *sampleStockService) GetStock(ctx context.Context, tsCode string) (stockdomain.Stock, error) {
	_ = ctx
	return s.store.getStock(tsCode)
}

func (s *sampleStockService) ListDailyBars(ctx context.Context, params stockdomain.DailyParams) ([]stockdomain.DailyBar, error) {
	_ = ctx
	return s.store.listStockDaily(params), nil
}

type memoryStore struct {
	mu sync.RWMutex

	stocks        map[string]datasource.StockBasic
	stockOrder    []string
	tradeCalendar []datasource.TradeDay
	dailyBars     map[string][]marketdata.DailyBar
	factors       map[string][]indicator.DailyFactor
	signals       []strategydomain.SignalResult
	jobRuns       []jobdomain.JobRun
	nextJobRunID  int64
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		stocks:    make(map[string]datasource.StockBasic),
		dailyBars: make(map[string][]marketdata.DailyBar),
		factors:   make(map[string][]indicator.DailyFactor),
	}
}

func (s *memoryStore) Start(ctx context.Context, jobName string, bizDate time.Time) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	targetDate := dateOnly(bizDate)
	for _, item := range s.jobRuns {
		if item.JobName == jobName && item.BizDate.Equal(targetDate) && item.Status == "running" {
			return fmt.Errorf("start job run %s: another run is already in progress for %s", jobName, dateKey(targetDate))
		}
	}

	s.nextJobRunID++
	now := time.Now().UTC()
	s.jobRuns = append(s.jobRuns, jobdomain.JobRun{
		ID:        s.nextJobRunID,
		JobName:   jobName,
		BizDate:   targetDate,
		Status:    "running",
		StartedAt: now,
		CreatedAt: now,
	})
	return nil
}

func (s *memoryStore) Success(ctx context.Context, jobName string, bizDate time.Time) error {
	_ = ctx
	return s.finishJobRun(jobName, bizDate, "success", 0, "")
}

func (s *memoryStore) Fail(ctx context.Context, jobName string, bizDate time.Time, err error) error {
	_ = ctx
	return s.finishJobRun(jobName, bizDate, "failed", 1, err.Error())
}

func (s *memoryStore) finishJobRun(jobName string, bizDate time.Time, status string, errorCode int, errorMessage string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	targetDate := dateOnly(bizDate)
	for i := len(s.jobRuns) - 1; i >= 0; i-- {
		item := &s.jobRuns[i]
		if item.JobName == jobName && item.BizDate.Equal(targetDate) && item.Status == "running" {
			item.Status = status
			item.FinishedAt = time.Now().UTC()
			item.ErrorCode = errorCode
			item.ErrorMessage = errorMessage
			return nil
		}
	}

	return fmt.Errorf("finish job run %s: running record not found", jobName)
}

func (s *memoryStore) UpsertStockBasics(ctx context.Context, items []datasource.StockBasic) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]struct{}, len(s.stockOrder))
	for _, code := range s.stockOrder {
		seen[code] = struct{}{}
	}
	for _, item := range items {
		s.stocks[item.TSCode] = item
		if _, ok := seen[item.TSCode]; !ok {
			s.stockOrder = append(s.stockOrder, item.TSCode)
			seen[item.TSCode] = struct{}{}
		}
	}
	sort.Strings(s.stockOrder)
	return nil
}

func (s *memoryStore) UpsertTradeCalendar(ctx context.Context, items []datasource.TradeDay) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tradeCalendar = append([]datasource.TradeDay(nil), items...)
	return nil
}

func (s *memoryStore) UpsertStockDaily(ctx context.Context, items []datasource.DailyBar) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	grouped := make(map[string][]marketdata.DailyBar)
	for _, item := range items {
		grouped[item.TSCode] = append(grouped[item.TSCode], marketdata.DailyBar{
			TSCode:    item.TSCode,
			TradeDate: dateOnly(item.TradeDate),
			Open:      item.Open,
			High:      item.High,
			Low:       item.Low,
			Close:     item.Close,
			PreClose:  item.PreClose,
			Change:    item.Change,
			PctChg:    item.PctChg,
			Vol:       item.Vol,
			Amount:    item.Amount,
			Source:    item.Source,
		})
	}
	for tsCode, bars := range grouped {
		s.dailyBars[tsCode] = mergeDailyBars(s.dailyBars[tsCode], bars)
	}
	return nil
}

func (s *memoryStore) ListStockDaily(ctx context.Context, startDate, endDate time.Time) ([]marketdata.DailyBar, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()

	startDate, endDate = normalizeDayRange(startDate, endDate)
	result := make([]marketdata.DailyBar, 0)
	for _, tsCode := range s.sortedDailyBarCodes() {
		for _, item := range s.dailyBars[tsCode] {
			if item.TradeDate.Before(startDate) || item.TradeDate.After(endDate) {
				continue
			}
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *memoryStore) UpsertDailyFactors(ctx context.Context, items []indicator.DailyFactor) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	grouped := make(map[string][]indicator.DailyFactor)
	for _, item := range items {
		grouped[item.TSCode] = append(grouped[item.TSCode], item)
	}
	for tsCode, factors := range grouped {
		s.factors[tsCode] = mergeDailyFactors(s.factors[tsCode], factors)
	}
	return nil
}

func (s *memoryStore) ListStrategyContexts(ctx context.Context, startDate, endDate time.Time) ([]strategydomain.MarketContext, error) {
	_ = ctx

	s.mu.RLock()
	defer s.mu.RUnlock()

	startDate, endDate = normalizeDayRange(startDate, endDate)
	items := make([]strategydomain.MarketContext, 0)
	for _, tsCode := range s.sortedDailyBarCodes() {
		bars := s.dailyBars[tsCode]
		if len(bars) == 0 {
			continue
		}

		factorByDate := make(map[string]indicator.DailyFactor, len(s.factors[tsCode]))
		for _, factor := range s.factors[tsCode] {
			factorByDate[dateKey(factor.TradeDate)] = factor
		}

		for i, bar := range bars {
			if bar.TradeDate.Before(startDate) || bar.TradeDate.After(endDate) {
				continue
			}

			factor, ok := factorByDate[dateKey(bar.TradeDate)]
			if !ok {
				continue
			}

			items = append(items, strategydomain.MarketContext{
				CurrentBar:    bar,
				CurrentFactor: factor,
				RecentBars:    recentBarsWindow(bars, i, 21),
			})
		}
	}

	return items, nil
}

func (s *memoryStore) UpsertStrategySignals(ctx context.Context, items []strategydomain.SignalResult) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	s.signals = append([]strategydomain.SignalResult(nil), items...)
	sort.Slice(s.signals, func(i, j int) bool {
		if s.signals[i].TradeDate.Equal(s.signals[j].TradeDate) {
			if s.signals[i].TSCode == s.signals[j].TSCode {
				return s.signals[i].StrategyCode < s.signals[j].StrategyCode
			}
			return s.signals[i].TSCode < s.signals[j].TSCode
		}
		return s.signals[i].TradeDate.After(s.signals[j].TradeDate)
	})
	return nil
}

// ReplaceStrategySignals 按日期区间覆盖重算后的策略信号。
func (s *memoryStore) ReplaceStrategySignals(ctx context.Context, startDate, endDate time.Time, items []strategydomain.SignalResult) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	startDate, endDate = normalizeDayRange(startDate, endDate)
	filtered := make([]strategydomain.SignalResult, 0, len(s.signals)+len(items))
	for _, item := range s.signals {
		if item.TradeDate.Before(startDate) || item.TradeDate.After(endDate) {
			filtered = append(filtered, item)
		}
	}
	filtered = append(filtered, items...)
	s.signals = filtered
	sort.Slice(s.signals, func(i, j int) bool {
		if s.signals[i].TradeDate.Equal(s.signals[j].TradeDate) {
			if s.signals[i].TSCode == s.signals[j].TSCode {
				return s.signals[i].StrategyCode < s.signals[j].StrategyCode
			}
			return s.signals[i].TSCode < s.signals[j].TSCode
		}
		return s.signals[i].TradeDate.After(s.signals[j].TradeDate)
	})

	return nil
}

func (s *memoryStore) ListSignals(ctx context.Context, params strategydomain.QueryParams) ([]strategydomain.SignalResult, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]strategydomain.SignalResult, 0, len(s.signals))
	targetDate := dateOnly(params.TradeDate)
	for _, item := range s.signals {
		if !item.TradeDate.Equal(targetDate) {
			continue
		}
		if params.StrategyCode != "" && item.StrategyCode != params.StrategyCode {
			continue
		}
		filtered = append(filtered, item)
	}

	start, end := pageWindow(params.Page, params.PageSize, len(filtered))
	return filtered[start:end], nil
}

func (s *memoryStore) ListJobRuns(ctx context.Context, params jobdomain.QueryParams) ([]jobdomain.JobRun, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]jobdomain.JobRun, 0, len(s.jobRuns))
	targetDate := dateOnly(params.BizDate)
	for i := len(s.jobRuns) - 1; i >= 0; i-- {
		item := s.jobRuns[i]
		if params.JobName != "" && item.JobName != params.JobName {
			continue
		}
		if !targetDate.IsZero() && !item.BizDate.Equal(targetDate) {
			continue
		}
		filtered = append(filtered, item)
	}

	start, end := pageWindow(params.Page, params.PageSize, len(filtered))
	return filtered[start:end], nil
}

func (s *memoryStore) listStocks(params stockdomain.ListParams) stockdomain.ListResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	page, pageSize := sanitizeStockPage(params.Page, params.PageSize)
	keyword := strings.ToLower(strings.TrimSpace(params.Keyword))
	items := make([]stockdomain.Stock, 0, len(s.stockOrder))
	for _, code := range s.stockOrder {
		item := s.stocks[code]
		if keyword != "" {
			targets := []string{
				strings.ToLower(item.TSCode),
				strings.ToLower(item.Symbol),
				strings.ToLower(item.Name),
			}
			matched := false
			for _, target := range targets {
				if strings.Contains(target, keyword) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		items = append(items, stockdomain.Stock{
			TSCode:   item.TSCode,
			Symbol:   item.Symbol,
			Name:     item.Name,
			Industry: item.Industry,
			Exchange: item.Exchange,
			IsActive: true,
		})
	}

	start, end := pageWindow(page, pageSize, len(items))
	return stockdomain.ListResult{
		Items:    items[start:end],
		Page:     page,
		PageSize: pageSize,
	}
}

func (s *memoryStore) getStock(tsCode string) (stockdomain.Stock, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.stocks[tsCode]
	if !ok {
		return stockdomain.Stock{}, fmt.Errorf("stock %s not found", tsCode)
	}

	return stockdomain.Stock{
		TSCode:   item.TSCode,
		Symbol:   item.Symbol,
		Name:     item.Name,
		Industry: item.Industry,
		Exchange: item.Exchange,
		IsActive: true,
	}, nil
}

func (s *memoryStore) listStockDaily(params stockdomain.DailyParams) []stockdomain.DailyBar {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]stockdomain.DailyBar, 0)
	for _, item := range s.dailyBars[params.TSCode] {
		if item.TradeDate.Before(params.StartDate) || item.TradeDate.After(params.EndDate) {
			continue
		}
		items = append(items, stockdomain.DailyBar{
			TSCode:    item.TSCode,
			TradeDate: item.TradeDate,
			Open:      item.Open.String(),
			High:      item.High.String(),
			Low:       item.Low.String(),
			Close:     item.Close.String(),
			PctChg:    item.PctChg.String(),
			Vol:       item.Vol.String(),
			Amount:    item.Amount.String(),
		})
	}

	return items
}

// sortedDailyBarCodes 返回当前已写入日线数据的股票代码列表。
func (s *memoryStore) sortedDailyBarCodes() []string {
	keys := make([]string, 0, len(s.dailyBars))
	for tsCode := range s.dailyBars {
		keys = append(keys, tsCode)
	}
	sort.Strings(keys)

	return keys
}

func (s *memoryStore) tradeDatesWithDailyBars() []time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return collectTradeDatesFromDailyBars(s.dailyBars)
}

func (s *memoryStore) tradeDatesWithFactors() []time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return collectTradeDatesFromFactors(s.factors)
}

func pageWindow(page, pageSize, total int) (int, int) {
	start := (page - 1) * pageSize
	if start > total {
		return total, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}

func sanitizeStockPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func dateOnly(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func normalizeDayRange(startDate, endDate time.Time) (time.Time, time.Time) {
	return dateOnly(startDate), dateOnly(endDate)
}

// recentBarsWindow 截取当前交易日前最多 window 条升序日线。
func recentBarsWindow(items []marketdata.DailyBar, index, window int) []marketdata.DailyBar {
	start := index - window + 1
	if start < 0 {
		start = 0
	}

	return append([]marketdata.DailyBar(nil), items[start:index+1]...)
}

// mergeDailyBars 按交易日覆盖合并单只股票的日线历史。
func mergeDailyBars(existing []marketdata.DailyBar, incoming []marketdata.DailyBar) []marketdata.DailyBar {
	merged := make(map[string]marketdata.DailyBar, len(existing)+len(incoming))
	for _, item := range existing {
		merged[dateKey(item.TradeDate)] = item
	}
	for _, item := range incoming {
		merged[dateKey(item.TradeDate)] = item
	}

	items := make([]marketdata.DailyBar, 0, len(merged))
	for _, item := range merged {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].TradeDate.Before(items[j].TradeDate)
	})

	return items
}

// mergeDailyFactors 按交易日覆盖合并单只股票的因子历史。
func mergeDailyFactors(existing []indicator.DailyFactor, incoming []indicator.DailyFactor) []indicator.DailyFactor {
	merged := make(map[string]indicator.DailyFactor, len(existing)+len(incoming))
	for _, item := range existing {
		merged[dateKey(item.TradeDate)] = item
	}
	for _, item := range incoming {
		merged[dateKey(item.TradeDate)] = item
	}

	items := make([]indicator.DailyFactor, 0, len(merged))
	for _, item := range merged {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].TradeDate.Before(items[j].TradeDate)
	})

	return items
}

// dateKey 将时间归一化为 UTC 日期键。
func dateKey(value time.Time) string {
	return dateOnly(value).Format("2006-01-02")
}
