# EastMoney Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将东方财富行情能力以 QuantSage 原生方式迁入 `apps/server`，同时保留现有 `tushare` 数据源，通过配置选择默认导入源，并新增一层 richer quote service 承接分钟线、周月线、复权、均线和批量查询。

**Architecture:** 保留现有 `internal/domain/datasource.Source` 作为导入任务最小契约，由新的 `datasource/eastmoney` 实现 `ListStocks`、`ListTradeCalendar`、`ListDailyBars`。在此之上新增东财专用 `marketdata/eastmoney` 查询服务，隔离 richer K 线能力，避免污染现有导入接口。配置和运行时只在 `config` 与 `server_runtime` 增量接线，不重构现有 stock DB 读服务。

**Tech Stack:** Go 1.22+、标准库 `net/http`、Gin 现有运行时、`shopspring/decimal`、现有 `apperror`、现有 `sample`/`tushare` datasource 测试风格。

**Status (2026-04-29):** 任务 1-5 已落地，其中 `marketdata/eastmoney` 已改为直接复用 `datasource/eastmoney` 导出的共享 history fallback 入口，不再保留本地 duplicated fallback 策略；任务 6 本轮仍明确跳过 HTTP 暴露；任务 7 的运行说明已同步，`gofmt` / `go test` / `go test -race` 已按影响范围完成，`golangci-lint run ./...` 受当前本地工具加载异常影响未跑通。

---

## 1. 已锁定决策

- 与 `tushare` 并存，不替换现有数据源能力。
- 本轮优先支持 A 股（沪深北）正式链路。
- 迁移目标不止日线导入，还包括分钟线、周月线、复权、均线、批量抓取。
- 不保留第三方项目 `backend/data` 的 API 形状；迁移后统一贴合 `quantsage` 领域边界。
- 不直接改现有 `stock.Service` 的数据库读职责；在线东财行情查询单独建服务。

## 2. 目标文件范围

预计涉及：

```text
apps/server/internal/config/config.go
apps/server/internal/config/config_test.go
configs/config.example.yaml
apps/server/internal/app/server_runtime.go
apps/server/internal/domain/datasource/types.go
apps/server/internal/domain/datasource/eastmoney/*
apps/server/internal/domain/marketdata/types.go
apps/server/internal/domain/marketdata/eastmoney/*
apps/server/internal/domain/job/import_jobs_test.go
apps/server/internal/domain/datasource/tushare/source_test.go
docs/architecture/v1-local-runbook.md
README.md
```

如果最终决定对外暴露东财 richer quote API，还会新增：

```text
apps/server/internal/interfaces/http/dto/quote.go
apps/server/internal/interfaces/http/handler/quote_handler.go
apps/server/internal/interfaces/http/router.go
apps/server/internal/interfaces/http/*_test.go
```

## 3. 目标接口与边界

### 3.1 导入层最小契约

保留现有：

```go
type Source interface {
	ListStocks(ctx context.Context) ([]StockBasic, error)
	ListTradeCalendar(ctx context.Context, exchange string, startDate, endDate time.Time) ([]TradeDay, error)
	ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]DailyBar, error)
}
```

### 3.2 新增 richer quote service 契约

建议新增：

```go
package eastmoney

type Interval string

const (
	Interval1Min  Interval = "1m"
	Interval5Min  Interval = "5m"
	Interval15Min Interval = "15m"
	Interval30Min Interval = "30m"
	Interval60Min Interval = "60m"
	IntervalDay   Interval = "1d"
	IntervalWeek  Interval = "1w"
	IntervalMonth Interval = "1mo"
	IntervalQuarter Interval = "1q"
	IntervalYear  Interval = "1y"
)

type AdjustType string

const (
	AdjustNone AdjustType = "none"
	AdjustQFQ  AdjustType = "qfq"
	AdjustHFQ  AdjustType = "hfq"
)

type KLine struct {
	TSCode       string
	TradeTime    time.Time
	Open         decimal.Decimal
	High         decimal.Decimal
	Low          decimal.Decimal
	Close        decimal.Decimal
	PreClose     decimal.Decimal
	Change       decimal.Decimal
	PctChg       decimal.Decimal
	Vol          decimal.Decimal
	Amount       decimal.Decimal
	TurnoverRate decimal.Decimal
	Source       string
}

type Query struct {
	TSCode   string
	Interval Interval
	Adjust   AdjustType
	Limit    int
	EndTime  time.Time
}

type Service interface {
	ListKLines(ctx context.Context, query Query) ([]KLine, error)
	GetLatestKLine(ctx context.Context, query Query) (KLine, error)
	BatchListKLines(ctx context.Context, queries []Query) (map[string][]KLine, error)
	ListKLinesWithMA(ctx context.Context, query Query, periods []int) ([]KLineWithMA, error)
}
```

## 4. 实施任务

### 任务 1：补配置和默认导入源选择

**文件：**

- 修改：`apps/server/internal/config/config.go`
- 修改：`apps/server/internal/config/config_test.go`
- 修改：`configs/config.example.yaml`
- 修改：`apps/server/internal/app/server_runtime.go`

- [x] **步骤 1：扩展配置模型**

在 `Config.Datasource` 下新增东财配置，并增加默认导入源选择：

```go
Datasource struct {
	DefaultSource string `yaml:"default_source"`
	Tushare struct {
		Token string `yaml:"token"`
	} `yaml:"tushare"`
	EastMoney struct {
		Endpoint       string `yaml:"endpoint"`
		QuoteEndpoint  string `yaml:"quote_endpoint"`
		TimeoutSeconds int    `yaml:"timeout_seconds"`
		MaxRetries     int    `yaml:"max_retries"`
		UserAgentMode  string `yaml:"user_agent_mode"`
	} `yaml:"eastmoney"`
}
```

- [x] **步骤 2：补环境变量覆盖与默认值**

增加：

- `QUANTSAGE_DATASOURCE_DEFAULT_SOURCE`
- `QUANTSAGE_EASTMONEY_ENDPOINT`
- `QUANTSAGE_EASTMONEY_QUOTE_ENDPOINT`
- `QUANTSAGE_EASTMONEY_TIMEOUT_SECONDS`
- `QUANTSAGE_EASTMONEY_MAX_RETRIES`
- `QUANTSAGE_EASTMONEY_USER_AGENT_MODE`

默认值建议：

- `default_source`: `tushare`
- `endpoint`: `https://push2his.eastmoney.com`
- `quote_endpoint`: `https://push2his.eastmoney.com`
- `timeout_seconds`: `30`
- `max_retries`: `2`
- `user_agent_mode`: `stable`

- [x] **步骤 3：调整运行时导入源选择逻辑**

将 `buildImportSource(cfg)` 改为显式选择：

```go
switch strings.ToLower(strings.TrimSpace(cfg.Datasource.DefaultSource)) {
case "eastmoney":
	return eastmoneyds.NewFromConfig(cfg.Datasource.EastMoney)
case "tushare", "":
	if strings.TrimSpace(cfg.Datasource.Tushare.Token) == "" {
		return nil
	}
	return tushareds.New(cfg.Datasource.Tushare.Token)
default:
	return nil
}
```

- [x] **步骤 4：补配置测试**

覆盖：

- YAML 解析 `eastmoney` 配置
- 环境变量覆盖
- `default_source=eastmoney`
- 未知 source 的降级行为

### 任务 2：实现东财底层 client 与代码映射

**文件：**

- 新建：`apps/server/internal/domain/datasource/eastmoney/client.go`
- 新建：`apps/server/internal/domain/datasource/eastmoney/codecs.go`
- 新建：`apps/server/internal/domain/datasource/eastmoney/types.go`
- 新建：`apps/server/internal/domain/datasource/eastmoney/client_test.go`

- [x] **步骤 1：抽离 client 配置与请求执行**

定义：

```go
type ClientConfig struct {
	Endpoint       string
	QuoteEndpoint  string
	Timeout        time.Duration
	MaxRetries     int
	UserAgentMode  string
}

type Client struct {
	httpClient *http.Client
	config     ClientConfig
}
```

要求：

- 使用标准库 `net/http`
- 支持 `context.Context`
- 自动处理 gzip
- 统一包装为 `apperror.CodeDatasourceUnavailable`

- [x] **步骤 2：迁移并重写 secid / 周期 / 复权映射**

从第三方代码吸收思路，但改成纯函数：

```go
func ConvertTSCodeToSecID(tsCode string) (string, error)
func MapIntervalToEastMoneyKLT(interval Interval) (string, error)
func MapAdjustType(adjust AdjustType) string
```

仅正式支持：

- `000001.SZ`
- `600000.SH`
- `430001.BJ`

不在本轮支持范围的市场直接返回明确错误，不做隐式兜底。

- [x] **步骤 3：补 client 单元测试**

覆盖：

- gzip 响应解压
- 非 200 响应包装
- HTML / 反爬页面识别
- `TSCode -> secid` 映射
- interval / adjust 参数转换

### 任务 3：实现 `datasource/eastmoney.Source`

**文件：**

- 新建：`apps/server/internal/domain/datasource/eastmoney/source.go`
- 新建：`apps/server/internal/domain/datasource/eastmoney/stocks.go`
- 新建：`apps/server/internal/domain/datasource/eastmoney/calendar.go`
- 新建：`apps/server/internal/domain/datasource/eastmoney/daily.go`
- 新建：`apps/server/internal/domain/datasource/eastmoney/mapper.go`
- 新建：`apps/server/internal/domain/datasource/eastmoney/source_test.go`

- [x] **步骤 1：先落 `ListDailyBars`**

实现：

```go
func (s *Source) ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]datasource.DailyBar, error)
```

要求：

- 单日同步优先走单日参数
- 区间同步支持 `start/end`
- 所有数值字段落为 `decimal.Decimal`
- 时间统一归一到 UTC date-only

- [x] **步骤 2：补 `ListStocks`**

要求：

- 返回 `TSCode`、`Symbol`、`Name`、`Exchange`
- `ListDate` 可为空时要有明确解析策略
- `Source` 固定为 `eastmoney`

说明：如果东财股票基础信息接口字段不稳定，需要在实现前先锁定稳定 endpoint，再补 fixture。

- [x] **步骤 3：补 `ListTradeCalendar`**

要求：

- 输出字段满足现有 `TradeDay`
- `exchange` 只支持 `SSE` / `SZSE` / `BSE`
- 如果东财无直接官方日历接口，允许在实现中通过可验证的行情交易日推导，但要把推导逻辑封装清楚并补中文注释

- [x] **步骤 4：补 `source_test.go`**

测试风格对齐 `tushare/source_test.go`，覆盖：

- 无效配置
- `ListDailyBars` 映射
- `ListStocks` 映射
- `ListTradeCalendar` 映射
- 业务错误包装

### 任务 4：实现 richer K 线查询服务

**文件：**

- 新建：`apps/server/internal/domain/marketdata/eastmoney/service.go`
- 新建：`apps/server/internal/domain/marketdata/eastmoney/ma.go`
- 新建：`apps/server/internal/domain/marketdata/eastmoney/aggregate.go`
- 新建：`apps/server/internal/domain/marketdata/eastmoney/service_test.go`

- [x] **步骤 1：定义查询 service**

封装从第三方项目迁来的：

- 日 / 周 / 月 / 季 / 年 K
- 1 / 5 / 15 / 30 / 60 分钟 K
- 最新一根
- 批量查询

- [x] **步骤 2：实现 MA 计算**

保留第三方 `GetKLineWithMA` 的思路，但改成纯领域函数：

```go
func AttachSimpleMovingAverages(items []KLine, periods []int) []KLineWithMA
```

禁止返回 `map[string]string`，改成明确结构：

```go
type MAValue struct {
	Period int
	Value  decimal.Decimal
}
```

- [x] **步骤 3：实现分钟线聚合**

迁移 `AggregateKLineEveryN` 的思路，改成：

```go
func AggregateKLines(items []KLine, size int) ([]KLine, error)
```

- [x] **步骤 4：补单元测试**

覆盖：

- 均线窗口不足
- 批量查询部分失败
- 聚合后 high/low/open/close/vol/amount 正确
- `GetLatestKLine` 空结果返回

### 任务 5：运行时接线与使用方式固化

**文件：**

- 修改：`apps/server/internal/app/server_runtime.go`
- 修改：`apps/server/internal/app/sample_runtime.go`
- 修改：`apps/server/internal/domain/job/import_jobs_test.go`

- [x] **步骤 1：保持导入任务零侵入**

要求：

- `sync_stock_basic`
- `sync_trade_calendar`
- `sync_daily_market`

继续只依赖 `datasource.Source`，不感知 richer quote service。

- [x] **步骤 2：补 `sample_runtime` 兼容性测试**

确保：

- 默认 sample 模式行为不变
- 切到 `eastmoney` 时不影响 runner 注册和任务调用

- [x] **步骤 3：补 source 选择测试**

在运行时层验证：

- 默认仍走 `tushare`
- 配置 `eastmoney` 后走东财
- `tushare token` 为空但 `eastmoney` 已配置时可正常构建 import source

### 任务 6：如需要，对外暴露 richer quote API

**文件：**

- 新建：`apps/server/internal/interfaces/http/dto/quote.go`
- 新建：`apps/server/internal/interfaces/http/handler/quote_handler.go`
- 修改：`apps/server/internal/interfaces/http/router.go`
- 新建：`apps/server/internal/interfaces/http/quote_routes_test.go`

- [x] **步骤 1：先确认是否需要本轮暴露 HTTP**

本轮结论：先不暴露东财 HTTP quote API，保持 richer quote service 只停留在后端领域层，避免过早扩展对外契约。

如果用户只需要后端内部服务，本任务整体跳过，不提前扩接口。

- [ ] **步骤 2：如暴露，则新增只读接口**

建议仅增加：

- `GET /api/quotes/:ts_code/kline`
- `GET /api/quotes/:ts_code/kline/latest`

查询参数：

- `interval`
- `adjust`
- `limit`
- `end_time`
- `ma`

- [ ] **步骤 3：补 handler 测试**

覆盖：

- 参数校验
- 非法 interval / adjust
- 空数据
- 成功响应 DTO

### 任务 7：文档同步与验证

**文件：**

- 修改：`README.md`
- 修改：`docs/architecture/v1-local-runbook.md`

- [x] **步骤 1：更新运行说明**

说明：

- 如何选择 `tushare` / `eastmoney`
- 东财配置项含义
- A 股支持范围
- richer quote service 当前是否暴露 HTTP

- [x] **步骤 2：执行验证命令**

当前状态：

- `gofmt -w internal/infra/browserfetch internal/config internal/app internal/domain/datasource/eastmoney internal/domain/marketdata/eastmoney`
- `go test -timeout 120s ./internal/config ./internal/app ./internal/domain/datasource/... ./internal/domain/job ./internal/domain/marketdata/...`
- `go test -timeout 120s ./internal/domain/datasource/eastmoney ./internal/domain/marketdata/eastmoney`
- `go test -timeout 120s ./...`
- `go test -race -timeout 120s ./internal/domain/datasource/... ./internal/domain/marketdata/...`
- `go test -race -timeout 120s ./...`
- `go build ./...`
- `go mod tidy`
- `golangci-lint run ./...` 当前在本地 `golangci-lint v1.64.8` 下报 `context loading failed: no go files to analyze`；执行 `go mod tidy` 后仍复现，属于工具加载异常，未定位到代码级 lint 错误

按影响范围至少执行：

```bash
cd apps/server
gofmt -w internal/config internal/app internal/domain/datasource internal/domain/marketdata internal/interfaces/http
go test -timeout 120s ./internal/config ./internal/app ./internal/domain/datasource/... ./internal/domain/job ./internal/interfaces/http/...
go test -race -timeout 120s ./internal/domain/datasource/... ./internal/domain/marketdata/...
```

- [x] **步骤 3：记录未覆盖风险**

至少明确：

- 东财接口是否存在反爬漂移
- 股票基础信息与交易日历 endpoint 的稳定性
- 是否仍需保留 `tushare` 作为 fallback

本轮补充风险：

- `ListDailyBars` 目前通过“先拉 A 股股票列表，再逐证券抓日 K”的方式适配现有导入接口，真实全市场同步成本偏高，后续可视实际运行情况增加分批/缓存/断点策略。
- `ListTradeCalendar` 当前使用上证综指日线推导沪深北统一 A 股交易日，若后续锁定更稳定的公开日历接口，应整体替换这一实现假设。
- EastMoney browser fallback 当前依赖 `chromedp` 默认可执行文件解析或显式 `browser_path`，项目内尚未做额外平台浏览器探测；部署环境需要自行保证 Chrome / Chromium 可用。
- `browserfetch.Runner` 已改为 Chrome Pool：每个 worker 持有一个 Chrome / Chromium 进程并在进程内复用多 tab；总并发 = `browser_count * browser_tabs_per_browser`，单进程累计打开 `browser_recycle_after_tabs` 个 tab 后在空闲时回收重启，`browser_max_concurrent_tabs` 仅保留为旧配置兼容字段。
- 浏览器 UA 优先使用 `github.com/lib4u/fake-useragent` 生成 Chrome UA，生成失败或结果为空时回退到项目内置固定 UA；反爬相关参数通过配置显式开放，默认保持保守。
- richer quote service 已落地在领域层，并与导入链路复用同一套 history fallback；若后续开放 HTTP 接口，需要补参数校验、DTO、handler 测试以及更高层的回退行为验证。

## 5. 实施顺序建议

建议按以下顺序落地：

1. 任务 1：配置与 source 选择
2. 任务 2：底层 client 与映射
3. 任务 3：`datasource/eastmoney.Source`
4. 任务 4：richer K 线查询服务
5. 任务 5：运行时接线
6. 任务 6：如有必要再暴露 HTTP
7. 任务 7：文档与验证收口

## 6. 关键风险与决策点

- 当前第三方源码已确认能直接复用的核心是 K 线链路，`ListStocks` / `ListTradeCalendar` 需要在实现前再次锁定稳定 endpoint。
- richer quote service 不应反向侵入 `datasource.Source`，否则会拖动 `sample` 与 `tushare` 一起扩接口。
- A 股范围内也要明确北交所编码规则，避免 secid 隐式兜底。
- 如果东财交易日历最终只能由行情推导，本轮必须把推导假设写进中文注释和运行文档。
