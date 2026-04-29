# EastMoney Browser Fallback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 QuantSage 的 EastMoney 数据链路增加可配置的 `http / auto / chromedp` 三种抓取模式，并抽出可复用的 `chromedp` 公共模块供后续其他数据源复用。

**Architecture:** 新增 `internal/infra/browserfetch` 作为浏览器抓取基础设施层，统一承接浏览器启动、Cookie 获取、缓存和可扩展 action。EastMoney 在 `datasource/eastmoney` 内新增 fallback client，默认先走 HTTP，遇到 HTML / 反爬页等异常时通过浏览器刷新 Cookie 后重试一次，再把结果继续映射回现有 `datasource` 与 `marketdata` 领域对象。

**Tech Stack:** Go 1.26、标准库 `net/http`、`chromedp`、现有 `apperror`、现有 `shopspring/decimal`、现有 `go test` / `go test -race` / `golangci-lint`

---

## 1. 文件结构与职责

### 新增文件

```text
apps/server/internal/infra/browserfetch/config.go
apps/server/internal/infra/browserfetch/cache.go
apps/server/internal/infra/browserfetch/cookies.go
apps/server/internal/infra/browserfetch/runner.go
apps/server/internal/infra/browserfetch/runner_test.go
apps/server/internal/domain/datasource/eastmoney/fallback_client.go
apps/server/internal/domain/datasource/eastmoney/fallback_client_test.go
```

### 修改文件

```text
apps/server/go.mod
apps/server/internal/config/config.go
apps/server/internal/config/config_test.go
apps/server/internal/app/server_runtime.go
apps/server/internal/app/server_runtime_test.go
apps/server/internal/domain/datasource/eastmoney/types.go
apps/server/internal/domain/datasource/eastmoney/client.go
apps/server/internal/domain/datasource/eastmoney/source.go
apps/server/internal/domain/datasource/eastmoney/source_test.go
apps/server/internal/domain/datasource/eastmoney/stocks.go
apps/server/internal/domain/datasource/eastmoney/calendar.go
apps/server/internal/domain/datasource/eastmoney/daily.go
apps/server/internal/domain/marketdata/eastmoney/service.go
apps/server/internal/domain/marketdata/eastmoney/service_test.go
configs/config.example.yaml
README.md
docs/architecture/v1-local-runbook.md
docs/superpowers/specs/ai_stock_analysis_database_technical_proposal.md
docs/superpowers/plans/2026-04-29-eastmoney-migration-implementation-plan.md
```

## 2. 实施任务

### Task 1: 扩展配置模型，声明浏览器回退模式

**Files:**

- Modify: `apps/server/internal/config/config.go`
- Modify: `apps/server/internal/config/config_test.go`
- Modify: `configs/config.example.yaml`

- [ ] **Step 1: 先写配置测试，锁定新增字段和默认值**

```go
func TestLoadAppliesEastMoneyBrowserDefaults(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
app:
  name: quantsage
database:
  dsn: postgres://demo
redis:
  addr: 127.0.0.1:6379
auth:
  session_secret: "***"
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Datasource.EastMoney.FetchMode != "auto" {
		t.Fatalf("FetchMode = %q, want %q", cfg.Datasource.EastMoney.FetchMode, "auto")
	}
	if cfg.Datasource.EastMoney.BrowserTimeoutSeconds != 60 {
		t.Fatalf("BrowserTimeoutSeconds = %d, want %d", cfg.Datasource.EastMoney.BrowserTimeoutSeconds, 60)
	}
	if cfg.Datasource.EastMoney.BrowserCookieTTLSeconds != 720 {
		t.Fatalf("BrowserCookieTTLSeconds = %d, want %d", cfg.Datasource.EastMoney.BrowserCookieTTLSeconds, 720)
	}
	if !cfg.Datasource.EastMoney.BrowserHeadless {
		t.Fatal("BrowserHeadless = false, want true")
	}
}
```

- [ ] **Step 2: 再补环境变量覆盖测试**

```go
func TestLoadAppliesEastMoneyBrowserEnvOverrides(t *testing.T) {
	t.Setenv("QUANTSAGE_EASTMONEY_FETCH_MODE", "chromedp")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_PATH", "/opt/google/chrome")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_TIMEOUT_SECONDS", "75")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_COOKIE_TTL_SECONDS", "900")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_HEADLESS", "false")
	t.Setenv("QUANTSAGE_EASTMONEY_BROWSER_USER_AGENT_MODE", "mobile")

	// 复用现有 Load 测试骨架，断言新字段被覆盖
}
```

- [ ] **Step 3: 实现配置字段、默认值和环境变量归一化**

```go
type EastMoneyConfig struct {
	Endpoint                 string `yaml:"endpoint"`
	QuoteEndpoint            string `yaml:"quote_endpoint"`
	TimeoutSeconds           int    `yaml:"timeout_seconds"`
	MaxRetries               int    `yaml:"max_retries"`
	UserAgentMode            string `yaml:"user_agent_mode"`
	FetchMode                string `yaml:"fetch_mode"`
	BrowserPath              string `yaml:"browser_path"`
	BrowserTimeoutSeconds    int    `yaml:"browser_timeout_seconds"`
	BrowserCookieTTLSeconds  int    `yaml:"browser_cookie_ttl_seconds"`
	BrowserHeadless          bool   `yaml:"browser_headless"`
	BrowserUserAgentMode     string `yaml:"browser_user_agent_mode"`
}
```

```go
const (
	defaultEastMoneyFetchMode               = "auto"
	defaultEastMoneyBrowserTimeoutSeconds   = 60
	defaultEastMoneyBrowserCookieTTLSeconds = 720
	defaultEastMoneyBrowserHeadless         = true
)
```

- [ ] **Step 4: 更新示例配置**

```yaml
datasource:
  eastmoney:
    endpoint: https://push2his.eastmoney.com
    quote_endpoint: https://push2his.eastmoney.com
    timeout_seconds: 30
    max_retries: 2
    user_agent_mode: stable
    fetch_mode: auto
    browser_path: ""
    browser_timeout_seconds: 60
    browser_cookie_ttl_seconds: 720
    browser_headless: true
    browser_user_agent_mode: stable
```

- [ ] **Step 5: 运行配置测试**

Run: `go test -timeout 120s ./internal/config`

Expected: PASS，新增默认值和环境变量覆盖断言通过

- [ ] **Step 6: Commit**

```bash
git add apps/server/internal/config/config.go apps/server/internal/config/config_test.go configs/config.example.yaml
git commit -m "feat: add eastmoney browser fallback config"
```

### Task 2: 新增通用 browserfetch 模块

**Files:**

- Create: `apps/server/internal/infra/browserfetch/config.go`
- Create: `apps/server/internal/infra/browserfetch/cache.go`
- Create: `apps/server/internal/infra/browserfetch/cookies.go`
- Create: `apps/server/internal/infra/browserfetch/runner.go`
- Create: `apps/server/internal/infra/browserfetch/runner_test.go`
- Modify: `apps/server/go.mod`

- [ ] **Step 1: 先写配置和缓存测试**

```go
func TestNormalizeConfigAppliesDefaults(t *testing.T) {
	t.Parallel()

	cfg := normalizeConfig(Config{})
	if !cfg.Enabled {
		t.Fatal("Enabled = false, want true")
	}
	if cfg.Timeout != 60*time.Second {
		t.Fatalf("Timeout = %v, want %v", cfg.Timeout, 60*time.Second)
	}
	if cfg.CookieTTL != 720*time.Second {
		t.Fatalf("CookieTTL = %v, want %v", cfg.CookieTTL, 720*time.Second)
	}
}

func TestCookieCacheUsesNormalizedPageURL(t *testing.T) {
	t.Parallel()

	cache := newCookieCache(10 * time.Minute)
	cache.Store(cacheKey{
		BrowserPath: "/chrome",
		PageURL:     "https://quote.eastmoney.com/sz000001.html",
		Headless:    true,
		UserAgent:   "stable",
	}, "a=b")

	header, ok := cache.Load(cacheKey{
		BrowserPath: "/chrome",
		PageURL:     "https://quote.eastmoney.com/sz000001.html?foo=bar#top",
		Headless:    true,
		UserAgent:   "stable",
	})
	if !ok || header != "a=b" {
		t.Fatalf("Load() = (%q, %v), want (%q, true)", header, ok, "a=b")
	}
}
```

- [ ] **Step 2: 引入 chromedp 依赖**

```bash
cd apps/server
go get github.com/chromedp/chromedp github.com/chromedp/cdproto/network
```

Expected: `go.mod` 与 `go.sum` 自动更新，不手动编辑 `go.sum`

- [ ] **Step 3: 实现 browserfetch 配置与缓存**

```go
type Config struct {
	Enabled       bool
	BrowserPath   string
	Headless      bool
	Timeout       time.Duration
	CookieTTL     time.Duration
	UserAgentMode string
	WindowWidth   int
	WindowHeight  int
}

type Runner interface {
	FetchCookieHeader(ctx context.Context, pageURL string) (string, error)
	Run(ctx context.Context, pageURL string, opts ...RunOption) error
	InvalidateCookies()
}
```

- [ ] **Step 4: 实现最小运行器和 Cookie Header 组装**

```go
func (r *runner) FetchCookieHeader(ctx context.Context, pageURL string) (string, error) {
	if header, ok := r.cache.Load(buildCacheKey(r.config, pageURL)); ok {
		return header, nil
	}

	cookies, err := r.fetchCookiesOnce(ctx, pageURL)
	if err != nil {
		return "", fmt.Errorf("fetch browser cookies for %s: %w", pageURL, err)
	}

	header := joinCookies(cookies)
	r.cache.Store(buildCacheKey(r.config, pageURL), header)
	return header, nil
}
```

- [ ] **Step 5: 用 stub 测试，不依赖真实浏览器**

```go
func TestRunnerInvalidateCookies(t *testing.T) {
	t.Parallel()

	runner := &runner{
		config: normalizeConfig(Config{}),
		cache:  newCookieCache(10 * time.Minute),
	}
	runner.cache.Store(buildCacheKey(runner.config, "https://quote.eastmoney.com/"), "a=b")
	runner.InvalidateCookies()

	if _, ok := runner.cache.Load(buildCacheKey(runner.config, "https://quote.eastmoney.com/")); ok {
		t.Fatal("cache still contains cookie after InvalidateCookies")
	}
}
```

- [ ] **Step 6: 运行 browserfetch 测试**

Run: `go test -timeout 120s ./internal/infra/browserfetch`

Expected: PASS，且不需要本机真的启动浏览器

- [ ] **Step 7: Commit**

```bash
git add apps/server/go.mod apps/server/go.sum apps/server/internal/infra/browserfetch
git commit -m "feat: add reusable browserfetch runner"
```

### Task 3: 为 EastMoney 新增 fallback client

**Files:**

- Create: `apps/server/internal/domain/datasource/eastmoney/fallback_client.go`
- Create: `apps/server/internal/domain/datasource/eastmoney/fallback_client_test.go`
- Modify: `apps/server/internal/domain/datasource/eastmoney/types.go`
- Modify: `apps/server/internal/domain/datasource/eastmoney/client.go`

- [ ] **Step 1: 先写回退策略测试**

```go
func TestFallbackClientUsesHTTPWhenResponseIsJSON(t *testing.T) {
	t.Parallel()

	httpClient := &fakeHTTPClient{
		body: []byte(`{"rc":0,"data":{"klines":[]}}`),
	}
	browser := &fakeBrowserRunner{}
	client := newFallbackClient(httpClient, browser, FallbackConfig{Mode: "auto"})

	_, err := client.GetHistory(context.Background(), "/api/qt/stock/kline/get", url.Values{})
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if browser.calls != 0 {
		t.Fatalf("browser.calls = %d, want %d", browser.calls, 0)
	}
}

func TestFallbackClientRetriesWithBrowserCookieWhenHTMLDetected(t *testing.T) {
	t.Parallel()

	httpClient := &fakeHTTPClient{
		bodies: [][]byte{
			[]byte("<html>captcha</html>"),
			[]byte(`{"rc":0,"data":{"klines":[]}}`),
		},
	}
	browser := &fakeBrowserRunner{cookieHeader: "st_si=abc"}
	client := newFallbackClient(httpClient, browser, FallbackConfig{Mode: "auto", QuotePageURL: "https://quote.eastmoney.com/"})

	_, err := client.GetHistory(context.Background(), "/api/qt/stock/kline/get", url.Values{})
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if browser.calls != 1 {
		t.Fatalf("browser.calls = %d, want %d", browser.calls, 1)
	}
}
```

- [ ] **Step 2: 定义 EastMoney 抓取模式与 fallback 配置**

```go
type FetchMode string

const (
	FetchModeHTTP     FetchMode = "http"
	FetchModeAuto     FetchMode = "auto"
	FetchModeChromedp FetchMode = "chromedp"
)

type FallbackConfig struct {
	Mode         FetchMode
	QuotePageURL string
}
```

- [ ] **Step 3: 实现统一 fallback client**

```go
type historyRequester interface {
	GetHistory(ctx context.Context, path string, query url.Values) ([]byte, error)
	GetQuote(ctx context.Context, path string, query url.Values) ([]byte, error)
}

type browserRunner interface {
	FetchCookieHeader(ctx context.Context, pageURL string) (string, error)
}
```

```go
func (c *fallbackClient) GetHistory(ctx context.Context, path string, query url.Values) ([]byte, error) {
	body, err := c.httpClient.GetHistory(ctx, path, query)
	if err == nil && !shouldFallback(body, nil) {
		return body, nil
	}

	switch c.config.Mode {
	case FetchModeHTTP:
		return body, err
	case FetchModeChromedp:
		return c.retryWithBrowserCookie(ctx, historyTarget, path, query)
	case FetchModeAuto:
		if shouldFallback(body, err) {
			return c.retryWithBrowserCookie(ctx, historyTarget, path, query)
		}
		return body, err
	default:
		return body, err
	}
}
```

- [ ] **Step 4: 在 client.go 中补支持带 Cookie 的重试入口**

```go
func (c *Client) GetHistoryWithHeaders(ctx context.Context, path string, query url.Values, headers http.Header) ([]byte, error) {
	return c.getWithHeaders(ctx, c.config.Endpoint, path, query, headers)
}
```

- [ ] **Step 5: 覆盖回退判定**

```go
func shouldFallback(body []byte, err error) bool {
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			return strings.Contains(strings.ToLower(err.Error()), "html") ||
				strings.Contains(strings.ToLower(err.Error()), "anti-bot")
		}
	}
	return looksLikeHTML("", body) || looksLikeBotPage(body)
}
```

- [ ] **Step 6: 运行 EastMoney client 测试**

Run: `go test -timeout 120s ./internal/domain/datasource/eastmoney`

Expected: PASS，新增 fallback client 用例通过

- [ ] **Step 7: Commit**

```bash
git add apps/server/internal/domain/datasource/eastmoney/client.go apps/server/internal/domain/datasource/eastmoney/types.go apps/server/internal/domain/datasource/eastmoney/fallback_client.go apps/server/internal/domain/datasource/eastmoney/fallback_client_test.go
git commit -m "feat: add eastmoney browser fallback client"
```

### Task 4: 让 datasource/eastmoney 接入 fallback client

**Files:**

- Modify: `apps/server/internal/domain/datasource/eastmoney/source.go`
- Modify: `apps/server/internal/domain/datasource/eastmoney/stocks.go`
- Modify: `apps/server/internal/domain/datasource/eastmoney/calendar.go`
- Modify: `apps/server/internal/domain/datasource/eastmoney/daily.go`
- Modify: `apps/server/internal/domain/datasource/eastmoney/source_test.go`
- Modify: `apps/server/internal/app/server_runtime.go`
- Modify: `apps/server/internal/app/server_runtime_test.go`

- [ ] **Step 1: 先补运行时与 source 选择测试**

```go
func TestBuildEastMoneyDatasourceConfigIncludesBrowserFallback(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Datasource.EastMoney.FetchMode = "auto"
	cfg.Datasource.EastMoney.BrowserPath = "/opt/google/chrome"
	cfg.Datasource.EastMoney.BrowserTimeoutSeconds = 60
	cfg.Datasource.EastMoney.BrowserCookieTTLSeconds = 720
	cfg.Datasource.EastMoney.BrowserHeadless = true

	got := buildEastMoneyDatasourceConfig(cfg)
	if got.FetchMode != "auto" {
		t.Fatalf("FetchMode = %q, want %q", got.FetchMode, "auto")
	}
}
```

- [ ] **Step 2: 改 Source 构造，注入 fallback client**

```go
type Source struct {
	config         Config
	clientConfig   ClientConfig
	client         *Client
	fallbackClient *FallbackClient
}
```

```go
func NewFromConfig(cfg Config) *Source {
	clientConfig := normalizeClientConfig(ClientConfig{...})
	browserConfig := browserfetch.Config{...}
	httpClient := NewClient(clientConfig)
	return newSourceWithClients(clientConfig, httpClient, NewFallbackClient(httpClient, browserfetch.NewRunner(browserConfig), buildFallbackConfig(cfg)))
}
```

- [ ] **Step 3: 把 stocks/calendar/daily 改为统一走 fallback client**

```go
body, err := s.fallbackClient.GetQuote(ctx, stockListPath, buildStockListQuery())
```

```go
body, err := s.fallbackClient.GetHistory(ctx, historyKLinePath, buildKLineQuery(...))
```

- [ ] **Step 4: 补 source_test 中的回退覆盖**

```go
func TestSourceListDailyBarsFallsBackFromHTML(t *testing.T) {
	t.Parallel()

	// 第一轮返回 HTML，第二轮返回有效 JSON
	// 断言最终 ListDailyBars 成功且浏览器 cookie 路径被调用一次
}
```

- [ ] **Step 5: 运行 app 与 datasource 测试**

Run: `go test -timeout 120s ./internal/app ./internal/domain/datasource/eastmoney`

Expected: PASS，source 选择和 EastMoney source 回退用例都通过

- [ ] **Step 6: Commit**

```bash
git add apps/server/internal/app/server_runtime.go apps/server/internal/app/server_runtime_test.go apps/server/internal/domain/datasource/eastmoney/source.go apps/server/internal/domain/datasource/eastmoney/stocks.go apps/server/internal/domain/datasource/eastmoney/calendar.go apps/server/internal/domain/datasource/eastmoney/daily.go apps/server/internal/domain/datasource/eastmoney/source_test.go
git commit -m "feat: wire browser fallback into eastmoney datasource"
```

### Task 5: 让 marketdata/eastmoney 接入 fallback client

**Files:**

- Modify: `apps/server/internal/domain/marketdata/eastmoney/service.go`
- Modify: `apps/server/internal/domain/marketdata/eastmoney/service_test.go`

- [ ] **Step 1: 先写 richer K 线回退测试**

```go
func TestServiceListKLinesFallsBackFromHTML(t *testing.T) {
	t.Parallel()

	service := newTestServiceWithFallback(...)

	items, err := service.ListKLines(context.Background(), Query{
		TSCode:   "000001.SZ",
		Interval: Interval5Min,
	})
	if err != nil {
		t.Fatalf("ListKLines() error = %v", err)
	}
	if len(items) == 0 {
		t.Fatal("len(items) = 0, want non-zero")
	}
}
```

- [ ] **Step 2: 修改 service 依赖统一的 fallback history client**

```go
type historyClient interface {
	GetHistory(ctx context.Context, path string, query url.Values) ([]byte, error)
}
```

```go
func NewFromClients(httpClient historyClient) Service {
	return &service{client: httpClient}
}
```

- [ ] **Step 3: 保持 MA 和聚合函数不受影响**

```go
// ma.go 与 aggregate.go 不改接口；仅校验现有测试继续通过
```

- [ ] **Step 4: 运行 marketdata 测试**

Run: `go test -timeout 120s ./internal/domain/marketdata/eastmoney`

Expected: PASS，HTML 回退和 기존 richer K 线测试都通过

- [ ] **Step 5: Commit**

```bash
git add apps/server/internal/domain/marketdata/eastmoney/service.go apps/server/internal/domain/marketdata/eastmoney/service_test.go
git commit -m "feat: wire browser fallback into eastmoney marketdata"
```

### Task 6: 文档与运行说明收口

**Files:**

- Modify: `README.md`
- Modify: `docs/architecture/v1-local-runbook.md`
- Modify: `docs/superpowers/specs/ai_stock_analysis_database_technical_proposal.md`
- Modify: `docs/superpowers/plans/2026-04-29-eastmoney-migration-implementation-plan.md`

- [ ] **Step 1: 更新 README，说明三种抓取模式**

```md
- `datasource.eastmoney.fetch_mode=http`：只走 HTTP
- `datasource.eastmoney.fetch_mode=auto`：默认 HTTP，遇到 HTML / 反爬页自动回退浏览器
- `datasource.eastmoney.fetch_mode=chromedp`：每次先取浏览器 Cookie 再发 HTTP 请求
```

- [ ] **Step 2: 更新运行手册，说明浏览器依赖**

```md
- 启用 `auto` 或 `chromedp` 模式时，部署环境必须具备 Chrome/Chromium 可执行文件
- 若配置 `browser_path` 为空，当前依赖 `chromedp` 默认可执行文件解析；解析失败会回退为数据源不可用错误
- `browserfetch.Runner` 使用 Chrome Pool：每个 worker 复用一个 Chrome 进程，并在进程内创建独立 tab
- 总并发 = `browser_count * browser_tabs_per_browser`；`browser_max_concurrent_tabs` 仅保留为旧配置兼容字段
- 单进程累计打开 `browser_recycle_after_tabs` 个 tab 后，在该 worker 空闲时回收重启，并通过 `Close(ctx)` 释放所有进程资源
- 可通过 `browser_disable_images`、`browser_blocked_url_patterns`、`browser_extra_flags` 等参数控制资源开销与运行特征
```

- [ ] **Step 3: 在技术方案与迁移计划中记录限制**

```md
- 公共 `browserfetch` 模块当前只承接 Cookie 获取和基础页面运行能力
- EastMoney 交易日历仍使用指数日线推导，浏览器回退只改变抓取 transport，不改变推导语义
- 浏览器 UA 优先通过 `github.com/lib4u/fake-useragent` 生成，生成失败时回退到项目内置固定 UA
- `golangci-lint` 若继续报环境级加载异常，需要在文档中单独记录
```

- [ ] **Step 4: Commit**

```bash
git add README.md docs/architecture/v1-local-runbook.md docs/superpowers/specs/ai_stock_analysis_database_technical_proposal.md docs/superpowers/plans/2026-04-29-eastmoney-migration-implementation-plan.md
git commit -m "docs: describe eastmoney browser fallback modes"
```

### Task 7: 验证与收尾

**Files:**

- Reference: `apps/server/internal/infra/browserfetch/*`
- Reference: `apps/server/internal/domain/datasource/eastmoney/*`
- Reference: `apps/server/internal/domain/marketdata/eastmoney/*`

- [ ] **Step 1: 运行格式化**

Run:

```bash
cd apps/server
gofmt -w internal/config internal/app internal/infra/browserfetch internal/domain/datasource/eastmoney internal/domain/marketdata/eastmoney
```

Expected: 无输出，文件被格式化

- [ ] **Step 2: 运行定向单测**

Run:

```bash
cd apps/server
go test -timeout 120s ./internal/config ./internal/app ./internal/infra/browserfetch ./internal/domain/datasource/eastmoney ./internal/domain/marketdata/eastmoney
```

Expected: PASS

- [ ] **Step 3: 运行全量单测**

Run:

```bash
cd apps/server
go test -timeout 120s ./...
```

Expected: PASS

- [ ] **Step 4: 运行竞态测试**

Run:

```bash
cd apps/server
go test -race -timeout 120s ./internal/infra/browserfetch ./internal/domain/datasource/eastmoney ./internal/domain/marketdata/eastmoney
```

Expected: PASS

- [ ] **Step 5: 运行静态检查**

Run:

```bash
cd apps/server
golangci-lint run ./...
```

Expected: PASS；若继续出现 `context loading failed: no go files to analyze`，记录为工具环境问题而非代码缺陷

- [ ] **Step 6: 记录未覆盖风险**

```md
- 未用真实浏览器做 E2E，只验证了 mock/stub 路径
- 部署环境浏览器路径自动探测仍需上线前人工确认
- 全市场日线若频繁命中回退，整体耗时会显著增长
```

- [ ] **Step 7: Commit**

```bash
git add apps/server
git commit -m "test: verify eastmoney browser fallback implementation"
```

## 3. 自检结论

- 配置、公共模块、EastMoney fallback、datasource 接线、marketdata 接线、文档和验证都已覆盖到任务
- 无 `TODO/TBD/implement later` 占位
- `browserfetch` 只承接基础浏览器能力，未混入 EastMoney 业务语义
- richer K 线与导入层都统一落在同一回退策略下，没有重复散写回退逻辑
