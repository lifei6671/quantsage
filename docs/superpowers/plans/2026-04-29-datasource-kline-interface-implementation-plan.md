# Datasource K-Line Interface Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `datasource.Source` 增加统一的单股票多周期 K 线查询能力，并补齐公共流式接口 `StreamKLines`。当前 `eastmoney`、`tushare`、`sample` 先按能力矩阵分别实现一次性查询与显式报错；未来新浪这类页面驱动型数据源再接入真实流式能力。

**Architecture:** 保留现有 `ListStocks`、`ListTradeCalendar`、`ListDailyBars` 三个批量导入接口不变，在 `datasource/types.go` 上新增 `Interval`、`KLine`、`KLineQuery`、`KLineStreamItem`、`ListKLines` 与 `StreamKLines`。`sample` 和 `tushare` 第一版只支持日线并把 `DailyBar` 映射到 `KLine`；`eastmoney` 复用现有单股票多周期 K 线解析能力，把 richer K 线查询能力下沉到 `datasource/eastmoney.Source`。`sample`、`tushare`、`eastmoney` 三个已实现源当前统一返回显式“不支持流式 K 线”错误；`sina` 则在 `browserfetch` 与 `datasource/sina` 之间实现 `watcher -> parser -> collector` 内部分层，承接真实流式回推。

**Tech Stack:** Go 1.26、标准库 `context/time`、`shopspring/decimal`、现有 `apperror`、现有 `datasource/eastmoney` 与 `marketdata/eastmoney` 代码、现有 `infra/browserfetch`、`chromedp`、`go test`、`go test -race`、`gofmt`

---

## 1. 文件结构与职责

### 修改文件

```text
apps/server/internal/domain/datasource/types.go
apps/server/internal/domain/datasource/sample/source.go
apps/server/internal/domain/datasource/sample/source_test.go
apps/server/internal/domain/datasource/tushare/source.go
apps/server/internal/domain/datasource/tushare/source_test.go
apps/server/internal/domain/datasource/eastmoney/source.go
apps/server/internal/domain/datasource/eastmoney/source_test.go
apps/server/internal/domain/datasource/eastmoney/daily.go
apps/server/internal/domain/datasource/eastmoney/mapper.go
apps/server/internal/domain/marketdata/eastmoney/service.go
apps/server/internal/domain/marketdata/eastmoney/service_test.go
apps/server/internal/infra/browserfetch/runner.go
apps/server/internal/infra/browserfetch/runner_test.go
apps/server/internal/app/sample_runtime_test.go
apps/server/internal/domain/job/import_jobs_test.go
```

### 可选新增文件

如果 `eastmoney` 的单票 K 线逻辑在现有文件里放不下，再新增：

```text
apps/server/internal/domain/datasource/eastmoney/kline.go
apps/server/internal/domain/datasource/eastmoney/kline_test.go
```

优先保持现有文件结构；只有在 `source.go` / `daily.go` 明显过载时才拆。

### 未来页面驱动型数据源新增文件

当启动新浪这类页面驱动型数据源接入时，再新增以下文件：

```text
apps/server/internal/infra/browserfetch/observe.go
apps/server/internal/infra/browserfetch/observe_test.go
apps/server/internal/domain/datasource/sina/source.go
apps/server/internal/domain/datasource/sina/source_test.go
apps/server/internal/domain/datasource/sina/watcher.go
apps/server/internal/domain/datasource/sina/parser.go
apps/server/internal/domain/datasource/sina/collector.go
apps/server/internal/domain/datasource/sina/collector_test.go
```

职责约束：

1. `observe.go`
   放通用页面响应监听能力，只输出原始响应事件，不带新浪语义。
2. `watcher.go`
   放新浪页面 URL、监听过滤条件和 `browserfetch` 接线。
3. `parser.go`
   只负责把单条响应 body 解析成候选 K 线。
4. `collector.go`
   只负责去重、排序、`Limit / StartTime / EndTime` 收口。
5. `source.go`
   只负责组装 `watcher -> parser -> collector -> ListKLines / StreamKLines`。

## 2. 实施任务

### Task 1: 扩展 datasource 基础类型与接口

**Files:**
- Modify: `apps/server/internal/domain/datasource/types.go`

- [x] **Step 1: 在 `types.go` 里新增通用周期、K 线、流式事件与查询结构**

```go
type Interval string

const (
	Interval1Min    Interval = "1m"
	Interval5Min    Interval = "5m"
	Interval15Min   Interval = "15m"
	Interval30Min   Interval = "30m"
	Interval60Min   Interval = "60m"
	IntervalDay     Interval = "1d"
	IntervalWeek    Interval = "1w"
	IntervalMonth   Interval = "1mo"
	IntervalQuarter Interval = "1q"
	IntervalYear    Interval = "1y"
)

type KLine struct {
	TSCode    string
	TradeTime time.Time
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	PreClose  decimal.Decimal
	Change    decimal.Decimal
	PctChg    decimal.Decimal
	Vol       decimal.Decimal
	Amount    decimal.Decimal
	Source    string
}

type KLineQuery struct {
	TSCode    string
	Interval  Interval
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

type KLineStreamItem struct {
	Items []KLine
	Err   error
}
```

- [x] **Step 2: 扩展 `Source` 接口**

```go
type Source interface {
	ListStocks(ctx context.Context) ([]StockBasic, error)
	ListTradeCalendar(ctx context.Context, exchange string, startDate, endDate time.Time) ([]TradeDay, error)
	ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]DailyBar, error)
	ListKLines(ctx context.Context, query KLineQuery) ([]KLine, error)
	StreamKLines(ctx context.Context, query KLineQuery) (<-chan KLineStreamItem, error)
}
```

- [x] **Step 3: 运行一次最小编译检查，收集所有因接口变更导致的编译错误**

Run: `go test -timeout 120s ./internal/domain/datasource/... ./internal/app ./internal/domain/job`

Expected: FAIL，报出 `sample`、`tushare`、`eastmoney` 与测试桩缺少 `ListKLines` / `StreamKLines` 的编译错误

- [ ] **Step 4: Commit**

```bash
git add apps/server/internal/domain/datasource/types.go
git commit -m "feat: extend datasource source with kline query types"
```

### Task 2: 让 sample 与测试桩先满足新接口

**Files:**
- Modify: `apps/server/internal/domain/datasource/sample/source.go`
- Modify: `apps/server/internal/domain/datasource/sample/source_test.go`
- Modify: `apps/server/internal/app/sample_runtime_test.go`
- Modify: `apps/server/internal/domain/job/import_jobs_test.go`

- [x] **Step 1: 先在 `sample/source_test.go` 写成功用例，锁定日线映射**

```go
func TestSourceListKLinesSupportsDayInterval(t *testing.T) {
	t.Parallel()

	source := New("../../../../testdata/sample")
	endDate := time.Date(2026, 4, 29, 15, 0, 0, 0, time.UTC)

	items, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.IntervalDay,
		EndTime:  endDate,
		Limit:    2,
	})
	if err != nil {
		t.Fatalf("ListKLines() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want %d", len(items), 2)
	}
	if items[0].TSCode != "000001.SZ" {
		t.Fatalf("items[0].TSCode = %q, want %q", items[0].TSCode, "000001.SZ")
	}
}
```

- [x] **Step 2: 再写不支持周期的失败用例**

```go
func TestSourceListKLinesRejectsUnsupportedInterval(t *testing.T) {
	t.Parallel()

	source := New("../../../../testdata/sample")

	_, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.Interval5Min,
		Limit:    10,
	})
	if err == nil {
		t.Fatal("ListKLines() error = nil, want non-nil")
	}
}
```

- [x] **Step 3: 在 `sample/source.go` 实现最小日线支持**

```go
func (s *Source) ListKLines(ctx context.Context, query datasource.KLineQuery) ([]datasource.KLine, error) {
	if strings.TrimSpace(query.TSCode) == "" {
		return nil, fmt.Errorf("list sample klines: ts_code is required")
	}
	if query.Interval != datasource.IntervalDay {
		return nil, fmt.Errorf("sample datasource does not support interval %s", query.Interval)
	}

	bars, err := s.ListDailyBars(ctx, query.StartTime, queryEndForDayQuery(query))
	if err != nil {
		return nil, fmt.Errorf("list sample klines: %w", err)
	}

	result := make([]datasource.KLine, 0, len(bars))
	for _, bar := range bars {
		if bar.TSCode != query.TSCode {
			continue
		}
		result = append(result, datasource.KLine{
			TSCode:    bar.TSCode,
			TradeTime: bar.TradeDate,
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

	return trimKLinesByLimit(result, query.Limit), nil
}
```

- [x] **Step 4: 让 sample 与测试桩一起实现流式占位接口，先保证其他包恢复编译**

```go
func (s *noopSampleSource) ListKLines(ctx context.Context, query datasource.KLineQuery) ([]datasource.KLine, error) {
	return nil, nil
}

func (s *noopSampleSource) StreamKLines(ctx context.Context, query datasource.KLineQuery) (<-chan datasource.KLineStreamItem, error) {
	return nil, datasource.UnsupportedStreamError("sample")
}
```

```go
func (s *fakeSource) ListKLines(ctx context.Context, query datasource.KLineQuery) ([]datasource.KLine, error) {
	return nil, nil
}

func (s *fakeSource) StreamKLines(ctx context.Context, query datasource.KLineQuery) (<-chan datasource.KLineStreamItem, error) {
	return nil, datasource.UnsupportedStreamError("fake")
}
```

- [x] **Step 5: 为 sample 补一条流式不支持用例**

```go
func TestSourceStreamKLinesReturnsUnsupported(t *testing.T) {
	t.Parallel()

	source := New("../../../../testdata/sample")
	_, err := source.StreamKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.IntervalDay,
		Limit:    1,
	})
	if err == nil {
		t.Fatal("StreamKLines() error = nil, want non-nil")
	}
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("StreamKLines() code = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}
}
```

- [x] **Step 6: 运行 sample 与依赖它的测试**

Run: `go test -timeout 120s ./internal/domain/datasource/sample ./internal/app ./internal/domain/job`

Expected: PASS，`sample` 与测试桩恢复编译，新日线/不支持周期用例通过

- [ ] **Step 7: Commit**

```bash
git add apps/server/internal/domain/datasource/sample/source.go apps/server/internal/domain/datasource/sample/source_test.go apps/server/internal/app/sample_runtime_test.go apps/server/internal/domain/job/import_jobs_test.go
git commit -m "feat: add sample datasource kline support"
```

### Task 3: 为 tushare 增加仅支持日线的 K 线查询

**Files:**
- Modify: `apps/server/internal/domain/datasource/tushare/source.go`
- Modify: `apps/server/internal/domain/datasource/tushare/source_test.go`

- [x] **Step 1: 在 `tushare/source_test.go` 先写成功映射用例**

```go
func TestListKLinesSupportsDayInterval(t *testing.T) {
	t.Parallel()

	source := newTestSourceWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"msg":"","data":{"fields":["ts_code","trade_date","open","high","low","close","pre_close","change","pct_chg","vol","amount"],"items":[["000001.SZ","20260429","10.10","10.40","10.00","10.30","10.00","0.30","3.00","12345","67890"]]}}`))
	})

	items, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.IntervalDay,
		EndTime:  time.Date(2026, 4, 29, 15, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("ListKLines() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
}
```

- [x] **Step 2: 再写不支持分钟线的用例**

```go
func TestListKLinesRejectsUnsupportedMinuteInterval(t *testing.T) {
	t.Parallel()

	source := New("configured-token")

	_, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.Interval5Min,
		Limit:    10,
	})
	if err == nil {
		t.Fatal("ListKLines() error = nil, want non-nil")
	}
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListKLines() code = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}
}
```

- [x] **Step 3: 再补一条流式不支持用例**

```go
func TestStreamKLinesReturnsUnsupported(t *testing.T) {
	t.Parallel()

	source := New("configured-token")
	_, err := source.StreamKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.IntervalDay,
		Limit:    1,
	})
	if err == nil {
		t.Fatal("StreamKLines() error = nil, want non-nil")
	}
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("StreamKLines() code = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}
}
```

- [x] **Step 4: 在 `tushare/source.go` 实现仅支持 `1d` 的查询**

```go
func (s *Source) ListKLines(ctx context.Context, query datasource.KLineQuery) ([]datasource.KLine, error) {
	if strings.TrimSpace(query.TSCode) == "" {
		return nil, apperror.New(apperror.CodeBadRequest, errors.New("ts_code is required"))
	}
	if query.Interval != datasource.IntervalDay {
		return nil, apperror.New(
			apperror.CodeDatasourceUnavailable,
			fmt.Errorf("tushare datasource does not support interval %s", query.Interval),
		)
	}

	params, err := buildDailyKLineParams(query)
	if err != nil {
		return nil, err
	}
	data, err := s.query(ctx, "daily", params, dailyFields)
	if err != nil {
		return nil, err
	}

	return mapTushareDailyDataToKLines(strings.TrimSpace(query.TSCode), data)
}
```

- [x] **Step 5: 给实现补最小辅助函数，避免把 `ListKLines` 写成超长方法**

```go
func buildDailyKLineParams(query datasource.KLineQuery) (map[string]string, error) {
	if query.Limit > 0 {
		endTime := query.EndTime
		if endTime.IsZero() {
			endTime = time.Now()
		}
		return map[string]string{"trade_date": formatDate(endTime)}, nil
	}
	if query.StartTime.IsZero() || query.EndTime.IsZero() {
		return nil, apperror.New(apperror.CodeBadRequest, errors.New("start_time and end_time are required when limit <= 0"))
	}
	return map[string]string{
		"start_date": formatDate(query.StartTime),
		"end_date":   formatDate(query.EndTime),
	}, nil
}
```

- [x] **Step 6: 增加 `StreamKLines` 占位实现**

```go
func (s *Source) StreamKLines(context.Context, datasource.KLineQuery) (<-chan datasource.KLineStreamItem, error) {
	return nil, datasource.UnsupportedStreamError("tushare")
}
```

- [x] **Step 7: 运行 tushare 测试**

Run: `go test -timeout 120s ./internal/domain/datasource/tushare`

Expected: PASS，新日线映射和不支持周期错误用例通过

- [ ] **Step 8: Commit**

```bash
git add apps/server/internal/domain/datasource/tushare/source.go apps/server/internal/domain/datasource/tushare/source_test.go
git commit -m "feat: add tushare daily kline query"
```

### Task 4: 为 eastmoney 下沉单票多周期 K 线能力

**Files:**
- Modify: `apps/server/internal/domain/datasource/eastmoney/source.go`
- Modify: `apps/server/internal/domain/datasource/eastmoney/daily.go`
- Modify: `apps/server/internal/domain/datasource/eastmoney/mapper.go`
- Modify: `apps/server/internal/domain/datasource/eastmoney/source_test.go`
- Modify: `apps/server/internal/domain/marketdata/eastmoney/service.go`
- Modify: `apps/server/internal/domain/marketdata/eastmoney/service_test.go`
- Create or Modify: `apps/server/internal/domain/datasource/eastmoney/kline.go`
- Create or Modify: `apps/server/internal/domain/datasource/eastmoney/kline_test.go`

- [x] **Step 1: 先在 `source_test.go` 或 `kline_test.go` 写一组分钟线成功用例**

```go
func TestSourceListKLinesMapsMinuteRows(t *testing.T) {
	t.Parallel()

	source := newSourceWithClient(testClientConfig(), newClientWithRoundTripper(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(`{"rc":0,"data":{"klines":["2026-04-29 09:35,10.10,10.30,10.40,10.00,1000,2000,1.00,10.00,0.00,0.00"]}}`), nil
	}))

	items, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.Interval5Min,
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("ListKLines() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
}
```

- [x] **Step 2: 再写参数错误、空 `TSCode` 与流式不支持用例**

```go
func TestSourceListKLinesRejectsEmptyTSCode(t *testing.T) {
	t.Parallel()

	source := newSourceWithClient(testClientConfig(), nil)
	_, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		Interval: datasource.IntervalDay,
		Limit:    1,
	})
	if err == nil {
		t.Fatal("ListKLines() error = nil, want non-nil")
	}
	if code := apperror.CodeOf(err); code != apperror.CodeBadRequest {
		t.Fatalf("ListKLines() code = %d, want %d", code, apperror.CodeOf(err))
	}
}

func TestSourceStreamKLinesReturnsUnsupported(t *testing.T) {
	t.Parallel()

	source := newSourceWithClient(testClientConfig(), nil)
	_, err := source.StreamKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.IntervalDay,
		Limit:    1,
	})
	if err == nil {
		t.Fatal("StreamKLines() error = nil, want non-nil")
	}
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("StreamKLines() code = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}
}
```

- [x] **Step 3: 在 `datasource/eastmoney` 增加统一 `ListKLines` 实现**

```go
func (s *Source) ListKLines(ctx context.Context, query datasource.KLineQuery) ([]datasource.KLine, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	normalizedQuery, err := normalizeKLineQuery(query)
	if err != nil {
		return nil, err
	}
	secID, err := ConvertTSCodeToSecID(normalizedQuery.TSCode)
	if err != nil {
		return nil, datasourceUnavailable(fmt.Errorf("convert ts_code %s to secid: %w", normalizedQuery.TSCode, err))
	}

	body, err := s.fallbackClient.GetHistory(ctx, historyKLinePath, buildDatasourceKLineQuery(secID, normalizedQuery))
	if err != nil {
		return nil, err
	}

return decodeDatasourceKLines(normalizedQuery.TSCode, normalizedQuery.Interval, body)
}
```

- [x] **Step 3.1: 增加 `StreamKLines` 占位实现，明确当前东财源不支持流式接口**

```go
func (s *Source) StreamKLines(context.Context, datasource.KLineQuery) (<-chan datasource.KLineStreamItem, error) {
	return nil, datasource.UnsupportedStreamError("eastmoney")
}
```

- [x] **Step 4: 把 `marketdata/eastmoney/service.go` 切到复用新的 datasource helper，而不是保留两套独立拼参/解码逻辑**

```go
type queryExecutor interface {
	ListKLines(ctx context.Context, query datasource.KLineQuery) ([]datasource.KLine, error)
}

func (s *service) ListKLines(ctx context.Context, query Query) ([]KLine, error) {
	items, err := s.source.ListKLines(ctx, datasource.KLineQuery{
		TSCode:   query.TSCode,
		Interval: datasource.Interval(query.Interval),
		EndTime:  query.EndTime,
		Limit:    query.Limit,
	})
	if err != nil {
		return nil, err
	}

	result := make([]KLine, 0, len(items))
	for _, item := range items {
		result = append(result, KLine{
			TSCode:    item.TSCode,
			TradeTime: item.TradeTime,
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
	return result, nil
}
```

- [x] **Step 5: 让 `ListDailyBars` 可复用新的日线 helper，但不要改变它的“全市场导入”职责**

```go
items, itemErr := s.ListKLines(groupCtx, datasource.KLineQuery{
	TSCode:    stock.TSCode,
	Interval:  datasource.IntervalDay,
	StartTime: startDate,
	EndTime:   endDate,
})
```

随后把 `[]datasource.KLine` 映射回 `[]datasource.DailyBar`，只在 `daily.go` 内做这一层转换。

- [x] **Step 6: 跑 eastmoney 相关测试**

Run: `go test -timeout 120s ./internal/domain/datasource/eastmoney ./internal/domain/marketdata/eastmoney`

Expected: PASS，`datasource` 与 `marketdata` 的单票 K 线查询都复用同一条底层逻辑

- [ ] **Step 7: Commit**

```bash
git add apps/server/internal/domain/datasource/eastmoney/source.go apps/server/internal/domain/datasource/eastmoney/daily.go apps/server/internal/domain/datasource/eastmoney/mapper.go apps/server/internal/domain/datasource/eastmoney/source_test.go apps/server/internal/domain/marketdata/eastmoney/service.go apps/server/internal/domain/marketdata/eastmoney/service_test.go
git commit -m "feat: add eastmoney datasource kline query"
```

### Task 5: 为页面驱动型数据源预埋通用响应 watcher

**Files:**
- Create: `apps/server/internal/infra/browserfetch/observe.go`
- Create: `apps/server/internal/infra/browserfetch/observe_test.go`
- Modify: `apps/server/internal/infra/browserfetch/runner.go`
- Modify: `apps/server/internal/infra/browserfetch/runner_test.go`

- [x] **Step 1: 先写 watcher 的流式监听测试，锁定 chan 与 idle timeout 语义**

```go
func TestObserveResponsesEmitsMatchedBodiesAndStopsOnIdle(t *testing.T) {
	t.Parallel()

	stubObserveHooks(t)

	var listener func(any)
	listenTargetFunc = func(_ context.Context, fn func(any)) {
		listener = fn
	}
	getResponseBodyFunc = func(_ context.Context, requestID network.RequestID) ([]byte, error) {
		if requestID != network.RequestID("req-1") {
			t.Fatalf("requestID = %q, want %q", requestID, "req-1")
		}
		return []byte(`{"ok":true}`), nil
	}

	stream, err := New(Config{}).ObserveResponses(
		context.Background(),
		"https://quote.example.com/stock/000001",
		WithObserveIdleTimeout(10*time.Millisecond),
		WithObserveURLContains("/stock/kline"),
		WithObserveResourceTypes(network.ResourceTypeXHR, network.ResourceTypeFetch),
	)
	if err != nil {
		t.Fatalf("ObserveResponses() error = %v", err)
	}

	listener(&network.EventResponseReceived{
		RequestID: network.RequestID("req-1"),
		Type:      network.ResourceTypeXHR,
		Response: &network.Response{
			URL:      "https://api.example.com/stock/kline",
			Status:   200,
			MimeType: "application/json",
		},
	})
	listener(&network.EventLoadingFinished{RequestID: network.RequestID("req-1")})

	item := <-stream.Responses
	if got := string(item.Body); got != `{"ok":true}` {
		t.Fatalf("item.Body = %q, want %q", got, `{"ok":true}`)
	}
	if err := <-stream.Done; err != nil {
		t.Fatalf("stream.Done = %v, want nil", err)
	}
}
```

- [x] **Step 2: 再写过滤与主动取消测试，避免把噪音响应混进业务链路**

```go
func TestObserveResponsesSkipsNonMatchingResponses(t *testing.T) {
	t.Parallel()

	stream, err := New(Config{}).ObserveResponses(
		context.Background(),
		"https://quote.example.com/stock/000001",
		WithObserveIdleTimeout(10*time.Millisecond),
		WithObserveURLContains("/stock/kline"),
	)
	if err != nil {
		t.Fatalf("ObserveResponses() error = %v", err)
	}

	select {
	case item := <-stream.Responses:
		t.Fatalf("unexpected response item: %+v", item)
	case err := <-stream.Done:
		if err != nil {
			t.Fatalf("stream.Done = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not stop after idle timeout")
	}
}
```

- [x] **Step 3: 在 `observe.go` 实现通用监听类型与 `Runner` 扩展方法**

```go
type ResponseMetadata struct {
	URL          string
	Status       int
	MIMEType     string
	ResourceType string
}

type ResponseStreamItem struct {
	URL        string
	ReceivedAt time.Time
	Body       []byte
	Err        error
}

type ResponseStream struct {
	Responses <-chan ResponseStreamItem
	Done      <-chan error
	Close     func()
}

func (r *runner) ObserveResponses(ctx context.Context, pageURL string, opts ...ObserveOption) (*ResponseStream, error)
```

实现约束：

1. 不修改现有 `FetchCookieHeader` / `Run` 行为。
2. 监听停止条件必须覆盖：
   - `ctx` 取消
   - 页面关闭
   - idle timeout
3. 只在 `browserfetch` 输出原始响应 body，不在这里解析 K 线。

- [x] **Step 4: 在 `runner.go` 接入 `chromedp.ListenTarget` 与响应 body 抓取钩子**

```go
var listenTargetFunc = chromedp.ListenTarget
var getResponseBodyFunc = func(ctx context.Context, requestID network.RequestID) ([]byte, error) {
	return network.GetResponseBody(requestID).Do(ctx)
}
```

```go
func (r *runner) observeResponses(
	ctx context.Context,
	cfg normalizedConfig,
	pageURL string,
	options observeOptions,
	out chan<- ResponseStreamItem,
) error
```

该实现必须复用现有 `runInTab`，不要再复制一套浏览器生命周期管理逻辑。

- [x] **Step 5: 运行 browserfetch 定向测试**

Run: `go test -timeout 120s ./internal/infra/browserfetch`

Expected: PASS，新增 watcher 测试与现有 cookie/run 测试全部通过

- [ ] **Step 6: Commit**

```bash
git add apps/server/internal/infra/browserfetch/observe.go apps/server/internal/infra/browserfetch/observe_test.go apps/server/internal/infra/browserfetch/runner.go apps/server/internal/infra/browserfetch/runner_test.go
git commit -m "feat: add browser response watcher for page-driven datasource"
```

### Task 6: 未来新浪页面驱动型 K 线接入

> **Scope note:** 这一任务原本属于 design 已确认、实现延后的后续阶段任务；当前工作区已经按该任务落地了 `datasource/sina` 的最小浏览器驱动型实现，现阶段能力边界已明确为“latest-N only”。

**Files:**
- Create: `apps/server/internal/domain/datasource/sina/source.go`
- Create: `apps/server/internal/domain/datasource/sina/source_test.go`
- Create: `apps/server/internal/domain/datasource/sina/watcher.go`
- Create: `apps/server/internal/domain/datasource/sina/parser.go`
- Create: `apps/server/internal/domain/datasource/sina/collector.go`
- Create: `apps/server/internal/domain/datasource/sina/collector_test.go`

- [x] **Step 1: 先在 `source_test.go` 写流式消费成功用例，锁定 `watcher -> parser -> collector` 链路**

```go
func TestSourceListKLinesConsumesWatcherStream(t *testing.T) {
	t.Parallel()

	stream := make(chan browserfetch.ResponseStreamItem, 2)
	stream <- browserfetch.ResponseStreamItem{Body: []byte(`{"items":[{"time":"2026-04-29 09:30","open":"10.1","close":"10.2"}]}`)}
	stream <- browserfetch.ResponseStreamItem{Body: []byte(`{"items":[{"time":"2026-04-29 09:35","open":"10.2","close":"10.3"}]}`)}
	close(stream)

	source := newSourceWithWatcher(&stubWatcher{
		stream: stream,
		done:   nil,
	})

	items, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.Interval5Min,
		Limit:    2,
	})
	if err != nil {
		t.Fatalf("ListKLines() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want %d", len(items), 2)
	}
	if !items[0].TradeTime.Before(items[1].TradeTime) {
		t.Fatalf("items not sorted ascending: %+v", items)
	}
}
```

- [x] **Step 2: 在 `collector_test.go` 写去重、时间过滤和 `Limit` 收口测试**

```go
func TestCollectorFinalizesByQueryWindowAndLimit(t *testing.T) {
	t.Parallel()

	collector := newCollector(datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.Interval5Min,
		EndTime:  time.Date(2026, 4, 29, 15, 0, 0, 0, time.UTC),
		Limit:    2,
	})

	collector.Add([]datasource.KLine{
		{TSCode: "000001.SZ", TradeTime: time.Date(2026, 4, 29, 9, 30, 0, 0, time.UTC)},
		{TSCode: "000001.SZ", TradeTime: time.Date(2026, 4, 29, 9, 35, 0, 0, time.UTC)},
		{TSCode: "000001.SZ", TradeTime: time.Date(2026, 4, 29, 9, 35, 0, 0, time.UTC)},
	})

	items := collector.Finalize()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want %d", len(items), 2)
	}
}
```

- [x] **Step 3: 在 `watcher.go` 里只做新浪请求监听接线，不掺杂解析逻辑**

```go
type pageResponseWatcher interface {
	ObserveResponses(ctx context.Context, pageURL string, opts ...browserfetch.ObserveOption) (*browserfetch.ResponseStream, error)
}

func (s *Source) watchKLineResponses(ctx context.Context, query datasource.KLineQuery) (*browserfetch.ResponseStream, error) {
	requestURL := buildKLineRequestURL(query)
	return s.browser.ObserveResponses(
		ctx,
		requestURL,
		browserfetch.WithObserveIdleTimeout(5*time.Second),
		browserfetch.WithObserveMatch(func(meta browserfetch.ResponseMetadata) bool {
			return matchSinaKLineResponse(meta, query.Interval)
		}),
	)
}
```

实现要求：

1. `matchSinaKLineResponse` 只读响应元信息，不解析 body。
2. 请求 URL 与响应匹配规则集中放在 `watcher.go`，不要散到 `source.go`。
3. 当前实现直接访问新浪 JSONP K 线接口，支持按 `Limit` 驱动 `datalen`；不支持显式历史时间窗或自定义 `EndTime`。
4. 当真实接口请求模式变化时，只改这一层。

- [x] **Step 4: 在 `parser.go` 与 `collector.go` 分离解析和收口职责**

```go
func parseKLinesFromResponse(tsCode string, interval datasource.Interval, body []byte) ([]datasource.KLine, error)

type collector struct {
	query datasource.KLineQuery
	items map[string]datasource.KLine
}

func (c *collector) Add(items []datasource.KLine)
func (c *collector) Finalize() []datasource.KLine
```

实现要求：

1. `parser` 只把单条响应转换成候选 `[]datasource.KLine`。
2. `collector` 用 `TradeTime` 做去重主键，并在 `Finalize()` 时统一排序。
3. `Limit > 0` 时按“当前时刻前最近 N 条”截断；`Limit <= 0` 的历史时间窗模式当前显式返回“不支持”错误。

- [x] **Step 5: 在 `source.go` 里把流式 watcher 消费收口为标准 `ListKLines`**

```go
func (s *Source) ListKLines(ctx context.Context, query datasource.KLineQuery) ([]datasource.KLine, error) {
	normalizedQuery, err := normalizeQuery(query)
	if err != nil {
		return nil, err
	}

	stream, err := s.watchKLineResponses(ctx, normalizedQuery)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	collector := newCollector(normalizedQuery)
	for item := range stream.Responses {
		if item.Err != nil {
			return nil, fmt.Errorf("watch sina kline response: %w", item.Err)
		}
		parsed, err := parseKLinesFromResponse(normalizedQuery.TSCode, normalizedQuery.Interval, item.Body)
		if err != nil {
			return nil, fmt.Errorf("parse sina kline response: %w", err)
		}
		collector.Add(parsed)
	}
	if err := <-stream.Done; err != nil {
		return nil, err
	}

	return collector.Finalize(), nil
}
```

同时把 `ListStocks`、`ListTradeCalendar`、`ListDailyBars` 明确实现为“不支持当前 datasource 能力”的错误返回，避免假装自己是完整批量导入源。

- [x] **Step 6: 运行新浪数据源定向测试**

Run: `go test -timeout 120s ./internal/domain/datasource/sina`

Expected: PASS，流式监听、解析、去重和收口语义全部通过

- [ ] **Step 7: Commit**

```bash
git add apps/server/internal/domain/datasource/sina apps/server/internal/infra/browserfetch/observe.go apps/server/internal/infra/browserfetch/observe_test.go
git commit -m "feat: add sina page-driven kline datasource"
```

### Task 7: 全链路验证与收尾

**Files:**
- Modify: `docs/superpowers/specs/2026-04-29-datasource-kline-interface-design.md`（仅当实现与设计产生必要的小偏差时）

- [x] **Step 1: 运行定向测试，确认所有受影响包通过**

Run: `go test -timeout 120s ./internal/domain/datasource/... ./internal/domain/marketdata/eastmoney ./internal/app ./internal/domain/job`

Expected: PASS

- [x] **Step 2: 运行关键竞态测试**

Run: `go test -race -timeout 120s ./internal/domain/datasource/eastmoney ./internal/domain/datasource/tushare`

Expected: PASS

- [x] **Step 3: 格式化本次修改文件**

Run: `gofmt -w apps/server/internal/domain/datasource/types.go apps/server/internal/domain/datasource/sample/source.go apps/server/internal/domain/datasource/sample/source_test.go apps/server/internal/domain/datasource/tushare/source.go apps/server/internal/domain/datasource/tushare/source_test.go apps/server/internal/domain/datasource/eastmoney/source.go apps/server/internal/domain/datasource/eastmoney/daily.go apps/server/internal/domain/datasource/eastmoney/mapper.go apps/server/internal/domain/datasource/eastmoney/kline.go apps/server/internal/domain/datasource/eastmoney/source_test.go apps/server/internal/domain/marketdata/eastmoney/service.go apps/server/internal/domain/marketdata/eastmoney/service_test.go apps/server/internal/infra/browserfetch/runner.go apps/server/internal/infra/browserfetch/runner_test.go apps/server/internal/infra/browserfetch/observe.go apps/server/internal/infra/browserfetch/observe_test.go apps/server/internal/app/sample_runtime_test.go apps/server/internal/domain/job/import_jobs_test.go`

Expected: no output

- [x] **Step 4: 重新运行最关键的一组测试，防止 gofmt 后引入疏漏**

Run: `go test -timeout 120s ./internal/domain/datasource/... ./internal/domain/marketdata/eastmoney`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add apps/server/internal/domain/datasource apps/server/internal/domain/marketdata/eastmoney apps/server/internal/app/sample_runtime_test.go apps/server/internal/domain/job/import_jobs_test.go
git commit -m "test: verify datasource kline interface integration"
```

## 3. 自检

### Spec 覆盖检查

这份计划覆盖了 spec 的核心要求：

1. `Source` 同时扩展 `ListKLines` 与 `StreamKLines`
2. `Interval` / `KLine` / `KLineQuery` / `KLineStreamItem` 类型定义
3. `sample`、`tushare`、`eastmoney` 当前都显式实现“不支持流式接口”错误
4. `sample` 与 `tushare` 仅支持日线并显式报错
5. `eastmoney` 支持多周期
6. 现有导入接口语义保持不变
7. `ListKLines` 保持原始不复权语义
8. 页面驱动型数据源的 `watcher -> parser -> collector` 分层和 idle timeout 收口语义

### Placeholder 扫描

本计划没有使用 `TODO`、`TBD`、`implement later`、`similar to Task N` 等占位语句。所有代码步骤都给出了明确的函数签名、测试骨架或执行命令。

### 类型一致性检查

本计划统一使用以下命名：

1. `datasource.Interval`
2. `datasource.KLine`
3. `datasource.KLineQuery`
4. `datasource.KLineStreamItem`
5. `Source.ListKLines`
6. `Source.StreamKLines`
7. `browserfetch.ResponseStream`
8. `pageResponseWatcher`

后续任务中的 `eastmoney`、`tushare` 与 `sina` 实现都以这一组命名为准，不额外引入第二套并行命名。
