# QuantSage Agent Guide

## 1. 仓库定位

QuantSage 是一个面向本地研究与验证场景的 AI 量化工作台，当前仓库由三个主要部分组成：

- `apps/server`：Gin API、worker、调度、数据源、领域服务
- `apps/web`：React + Vite 前端工作台
- `deployments/docker-compose`：本地 PostgreSQL / Redis / MinIO 依赖

根目录不是 Go 模块。后端 Go 模块根在 `apps/server`。

## 2. 工作目录约定

### 2.1 后端

涉及 Go 代码、测试、依赖整理、lint、编译时：

- 优先在 `apps/server` 下执行命令
- 除非明确需要根目录 `make` 封装，否则不要在仓库根误跑 `go test ./...`

### 2.2 前端

涉及前端页面、组件、Vite 配置时：

- 在 `apps/web` 下执行 `npm` 命令
- 保持前端继续通过 Vite proxy 访问 `/api/*`

### 2.3 根目录

涉及跨应用联调、迁移、docker compose、本地启动文档时：

- 优先使用根目录 `Makefile`
- 参考 `README.md` 和 `docs/architecture/v1-local-runbook.md`

## 3. 常用命令

优先使用项目已经封装好的根命令：

```bash
make build
make test
make race
make lint
make fmt
make tidy
```

等价后端命令都在 `apps/server` 下执行，例如：

```bash
go build ./...
go test -timeout 120s ./...
go test -race -timeout 120s ./...
golangci-lint run ./...
```

前端常用命令：

```bash
cd apps/web && npm install
cd apps/web && npm run dev
cd apps/web && npm run build
```

## 4. 后端代码约定

### 4.1 基本风格

- 以 Go 1.26.x 为准
- 新增或修改公开函数/接口时，同步补中文注释
- 错误使用 `fmt.Errorf("...: %w", err)` 包装，保留上下文
- IO、数据库、浏览器抓取、外部调用函数，第一个参数保持 `context.Context`

### 4.2 数据源代码

当前后端已经有 `tushare`、`eastmoney`、`sina`、`sample`、`finscope` 等数据源实现或骨架。

处理数据源相关任务时：

- 先看 `apps/server/internal/domain/datasource`
- 优先复用已有查询模型、错误码和 browserfetch 基础设施
- 不要在 parser 里混入浏览器动作
- `List*` / `Stream*` 的职责边界要保持清晰

### 4.3 浏览器抓取

浏览器抓取基础设施在 `apps/server/internal/infra/browserfetch`。

处理站点抓取时：

- 优先复用 `browserfetch.Run`、`RunWithActions`、`ObserveResponses`
- 站点如果对页面加载、子 tab、Cookie 或懒加载敏感，先补 smoke test 再改实现
- 修改浏览器抓取行为时，优先补对应单测和最小真实烟雾测试

百度财经这类站点已经踩过一轮真实坑，后续处理同类页面时，优先参考下面的四元组：

- **Scene**：抓取百度财经页面并监听页面内 `fetch/XMLHttpRequest` 响应。
  **Wrong**：先做一次空 `chromedp.Run(...)` 预热，再执行真正导航。
  **Right**：直接让第一次真实页面动作启动浏览器，不要额外做预热初始化。
  **Reason**：百度财经在这条链路下会把后续导航立刻打成 `context canceled`，导致页面还没开始抓取就失败。

- **Scene**：打开百度财经指数成分股页这类真实业务页面。
  **Wrong**：在浏览器已创建后的第二个子 tab 里再开业务页面。
  **Right**：优先使用独立浏览器进程的主页面 target 打开业务页面。
  **Reason**：该站点对二级 tab 非常敏感，子 tab 导航会秒退 `context canceled`，而主页面 target 可以稳定进入 `interactive` 并继续执行脚本与滚动。

- **Scene**：页面能打开，但 `chromedp.Navigate(...)` 长时间卡住或超时。
  **Wrong**：默认认为是 `WaitReady`、脚本注入或解析器有问题，继续在后置动作里排查。
  **Right**：先对照验证底层 `page.Navigate` 是否可用，再决定是否切到原始导航模式。
  **Reason**：百度财经页面可以在 `page.Navigate` 下快速进入 `interactive`，但 `chromedp.Navigate` 会等待更重的页面加载语义，误导排障方向。

- **Scene**：需要在百度财经页面里拦截分页接口并持续滚动收集多批数据。
  **Wrong**：直接依赖 CDP `Network` 域抓包，或者跳过真实页面流程去手拼接口请求。
  **Right**：在导航前注入页面脚本，拦截页面自身 `fetch/XMLHttpRequest` 返回体，再在同页上下文里滚动触发懒加载。
  **Reason**：这条路径更贴近真实站点行为，也更能规避百度财经页面对抓包时序、Cookie 和前端懒加载的耦合问题。

## 5. 前端代码约定

- 保持现有 React + Vite 结构，不擅自引入新框架
- 新页面或大改交互时，优先延续现有工作台风格，而不是做营销页式设计
- 影响 API 契约、登录态、代理目标时，要同步检查前后端联调路径

## 6. 文档同步约定

以下变更完成后，通常要同步文档：

- 新增或修改数据源
- 修改后端 API、任务流、配置项、调度行为
- 修改浏览器抓取策略、验证方式或本地运行方式
- 新增真实 smoke test、联调入口或关键排障结论

优先更新这些位置：

- `README.md`
- `docs/architecture/*.md`
- `docs/superpowers/specs/*.md`
- `docs/superpowers/plans/*.md`

如果本次变更不需要更新文档，交付时应明确说明原因。

## 7. 验证要求

完成代码后，至少按变更范围执行对应验证：

- 后端改动：`make test`，必要时加 `make race`、`make build`
- 前端改动：`cd apps/web && npm run build`
- 浏览器抓取或外部站点改动：补跑最小 smoke test
- 迁移或运行入口改动：检查 `README.md` / runbook 是否仍然可用

如果某项验证因为环境限制无法执行，要在结果里明确说明“没跑什么、为什么没跑”。

## 8. 不要做的事

- 不要在根目录手写与现有 `Makefile` 冲突的新验证约定
- 不要手动编辑 `go.sum`
- 不要为了单次任务引入新的第三方依赖，除非用户明确确认
- 不要把调试残留、临时日志、取消掉的 `t.Skip(...)` 留在测试里

## 9. 交付时建议说明

交付结果最好覆盖这几件事：

- 改了什么
- 为什么这么改
- 跑了哪些验证
- 哪些东西没验证
- 是否更新了文档，若没有，为什么
