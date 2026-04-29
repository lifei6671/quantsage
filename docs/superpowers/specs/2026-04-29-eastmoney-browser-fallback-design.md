# EastMoney Browser Fallback Design

## 1. 目标

为 QuantSage 的 EastMoney 迁移链路补齐浏览器回退能力：

1. 抽象一个可复用的 `chromedp` 公共模块，供后续其他数据源复用。
2. 让 EastMoney 默认先走 HTTP，命中反爬或异常页面时自动回退到浏览器取 Cookie 后重试。
3. 保持现有 `datasource/eastmoney` 与 `marketdata/eastmoney` 的领域边界不变，不把浏览器细节泄漏到业务调用方。

本次设计只覆盖后端服务内部能力，不新增对外 HTTP API。

## 2. 设计结论

采用两层抽象：

1. **公共浏览器抓取层**
   放在 `apps/server/internal/infra/browserfetch`，只负责浏览器启动、页面访问、Cookie 获取、缓存和可扩展 action，不带任何 EastMoney 语义。
2. **EastMoney 请求策略层**
   放在 `apps/server/internal/domain/datasource/eastmoney`，负责判断什么时候应从纯 HTTP 回退到浏览器辅助模式，并把回退后的重试结果继续暴露为现有领域对象。

这是本次推荐方案，因为它把“浏览器怎么跑”和“EastMoney 什么时候回退”分开，后续别的数据源只需复用第一层，不会被 EastMoney 的业务规则绑死。

## 3. 范围

### 3.1 本次纳入范围

- `datasource/eastmoney` 的股票列表、交易日历、全市场日线
- `marketdata/eastmoney` 的 richer K 线查询
- 配置模型、运行说明、测试与实现计划同步更新

### 3.2 本次不纳入范围

- 对外 HTTP quote API
- 非 EastMoney 数据源的浏览器接入
- 通用“多数据源统一抓取 DSL”
- 真实浏览器 E2E 测试

## 4. 公共模块设计

### 4.1 目录

新增目录：

```text
apps/server/internal/infra/browserfetch/
```

建议包含：

- `config.go`
  - 浏览器配置定义与默认值归一化
- `runner.go`
  - `chromedp` 运行器创建、生命周期、基础执行入口
- `cookies.go`
  - 页面 Cookie 获取与 Cookie Header 拼装
- `cache.go`
  - 基于页面 URL 和浏览器配置的 Cookie 缓存
- `runner_test.go`
  - 配置和缓存等纯单测

### 4.2 对外能力

公共层只暴露基础浏览器能力，不暴露 EastMoney 业务方法。

建议对外接口形状：

```go
type Config struct {
	Enabled              bool
	BrowserPath          string
	Headless             bool
	Timeout              time.Duration
	CookieTTL            time.Duration
	UserAgentMode        string
	WindowWidth          int
	WindowHeight         int
}

type Runner interface {
	FetchCookieHeader(ctx context.Context, pageURL string) (string, error)
	Run(ctx context.Context, pageURL string, opts ...RunOption) error
	InvalidateCookies()
}
```

`RunOption` 用于后续扩展 `navigate / wait / custom action`。本次 EastMoney 只会直接使用 `FetchCookieHeader`，但模块要保留后续复用空间。

### 4.3 Cookie 缓存语义

- 缓存键包含：
  - 浏览器路径
  - 页面 URL 的规范化路径
  - headless 与 UA 模式等关键浏览器特征
- Cookie 缓存只缓存页面访问得到的 Cookie Header，不缓存业务数据响应
- 过期后重新拉取
- 提供显式失效方法，便于调试和未来热更新配置

## 5. EastMoney 请求策略设计

### 5.1 配置扩展

在现有 `datasource.eastmoney` 下新增：

```yaml
datasource:
  eastmoney:
    fetch_mode: auto
    browser_path: ""
    browser_timeout_seconds: 60
    browser_cookie_ttl_seconds: 720
    browser_headless: true
    browser_user_agent_mode: stable
```

### 5.2 配置语义

- `fetch_mode=http`
  - 只走 HTTP，不回退浏览器
- `fetch_mode=auto`
  - 默认走 HTTP
  - 命中回退条件时，用浏览器获取 Cookie 后重试一次 HTTP
- `fetch_mode=chromedp`
  - 每次请求都先通过浏览器拿 Cookie，再发业务 HTTP 请求
  - 仍由标准库 HTTP 发真正的数据请求，不直接在浏览器里抓 K 线 JSON

默认值：

- `fetch_mode=auto`
- `browser_timeout_seconds=60`
- `browser_cookie_ttl_seconds=720`
- `browser_headless=true`
- `browser_user_agent_mode=stable`

### 5.3 EastMoney 内部结构调整

在 `apps/server/internal/domain/datasource/eastmoney` 内新增一层请求策略抽象：

- `client.go`
  - 负责纯 HTTP 请求
- `fallback_client.go`
  - 负责 `http -> browser cookie refresh -> retry http`
- `browser_bridge.go`
  - EastMoney 对公共 `browserfetch.Runner` 的轻量接线

调用链保持不变：

- `Source.ListStocks`
- `Source.ListTradeCalendar`
- `Source.ListDailyBars`
- `marketdata/eastmoney.Service.ListKLines`

这些调用方都只依赖统一的 EastMoney 请求接口，不直接感知浏览器模块。

## 6. 回退判定规则

### 6.1 触发回退的情况

当 `fetch_mode=auto` 时，首次 HTTP 请求命中以下任一情况，触发一次浏览器辅助重试：

1. 响应为 HTML 页面
2. 响应包含明显反爬/验证码/机器人校验特征
3. HTTP 状态码异常，且具备反爬特征
4. JSON 解码失败，同时 body 更像页面内容而不是正常 JSON

### 6.2 不触发回退的情况

以下情况直接返回错误，不做浏览器回退：

1. 上下文取消或超时
2. 请求参数本身非法
3. EastMoney 明确返回业务错误且响应格式正常
4. 浏览器功能被配置禁用且模式为 `http`

### 6.3 重试规则

- 每个请求只允许一次浏览器回退重试
- 回退时流程为：
  1. 访问 `quote.eastmoney.com` 或指定页面
  2. 获取 Cookie Header
  3. 带 Cookie 重新发起原始 HTTP 请求
- 重试仍失败，则返回 `apperror.CodeDatasourceUnavailable`

## 7. 业务影响

### 7.1 对 `datasource/eastmoney` 的影响

- 股票列表接口改为使用统一 fallback client
- 交易日历推导链路中的指数日线请求改为使用统一 fallback client
- 全市场日线逐证券抓取请求改为使用统一 fallback client

### 7.2 对 `marketdata/eastmoney` 的影响

- richer K 线查询统一走 fallback client
- 均线和聚合仍保持纯领域函数，不与浏览器能力耦合

### 7.3 对运行时的影响

- `server_runtime` 只负责注入 EastMoney 配置
- 不在运行时层写任何浏览器业务逻辑

## 8. 错误处理

- 浏览器模块内部错误统一包装为带上下文的普通 `error`
- EastMoney 领域层负责将浏览器失败、HTTP 失败和二次重试失败统一包装成 `apperror.CodeDatasourceUnavailable`
- 所有错误消息都要显式区分：
  - 首次 HTTP 失败
  - 浏览器取 Cookie 失败
  - 带 Cookie 重试失败

这样后续排查时能快速知道是网络问题、反爬问题还是浏览器环境问题。

## 9. 测试策略

### 9.1 公共模块测试

覆盖：

- 配置默认值归一化
- Cookie 缓存命中
- Cookie 缓存过期
- URL 规范化缓存键
- 显式失效逻辑

这些测试全部使用 mock/stub，不依赖真实浏览器。

### 9.2 EastMoney fallback 测试

覆盖：

- HTTP 成功时不触发浏览器
- HTML 响应触发浏览器回退
- 浏览器拿到 Cookie 后重试成功
- 浏览器回退后仍失败
- `fetch_mode=http` 时即使命中 HTML 也不回退
- `fetch_mode=chromedp` 时直接先取 Cookie 再请求

### 9.3 领域集成测试

覆盖：

- `ListStocks`
- `ListTradeCalendar`
- `ListDailyBars`
- `ListKLines`

确认它们都复用了统一 fallback client，而不是各自散写回退逻辑。

## 10. 风险与约束

### 10.1 依赖变更

需要新增 `chromedp` 相关依赖。这属于明确依赖变更，实现前需要再次执行用户确认规则。

### 10.2 运行环境

- 本地或 CI 可能没有可用 Chrome/Chromium
- 因此测试必须以 mock 为主，不能依赖真实浏览器
- 文档中需要明确：启用 `auto/chromedp` 模式时，部署环境必须具备浏览器可执行文件

### 10.3 性能

- 成功路径仍以 HTTP 为主，性能影响可控
- 失败路径引入浏览器冷启动，单次请求延迟会明显增加
- 当前全市场日线本就采用逐证券抓取，若频繁命中回退，会进一步放大总耗时

## 11. 实施顺序

建议顺序：

1. 扩展配置模型和默认值
2. 新建 `infra/browserfetch`
3. 给 EastMoney 新增 fallback client
4. 让 `datasource/eastmoney` 接入 fallback client
5. 让 `marketdata/eastmoney` 接入 fallback client
6. 更新文档与运行说明
7. 运行格式化、单测、竞态测试、静态检查

## 12. 验收标准

满足以下条件才算完成：

1. EastMoney 支持 `http / auto / chromedp` 三种抓取模式
2. 公共浏览器模块不包含 EastMoney 业务语义
3. `datasource/eastmoney` 与 `marketdata/eastmoney` 都复用同一回退策略
4. 失败重试行为可测且被单测覆盖
5. 文档明确说明配置项、浏览器依赖和未覆盖风险
