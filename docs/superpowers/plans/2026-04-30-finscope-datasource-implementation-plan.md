# Finscope Datasource Skeleton Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 初始化一个浏览器驱动型 `finscope` 数据源骨架，并实现 `ListStocks` 的第一条串行子方法：抓取上证指数成分股。

**Architecture:** 新增 `apps/server/internal/domain/datasource/finscope/` 目录，并拆成 `source.go`、`types.go`、`watcher.go`、`parser.go`、`source_test.go` 五个文件。首版提供构造函数、浏览器依赖接口、统一未实现错误和接口空实现，同时让 `ListStocks` 通过“子方法串行执行”的方式先接入百度财经上证指数成分股真实页面抓取链路：在独立浏览器进程主页面 target 上用底层 `page.Navigate` 打开页面、预注入页面脚本拦截响应、滚动触发后续分页。

**Tech Stack:** Go、现有 `datasource.Source` 契约、`apperror`、`browserfetch.RunWithActions`、页面内 `fetch/XMLHttpRequest` 劫持脚本、主页面 target 导航模式

---

### Task 1: 新增 Finscope 骨架代码

**Files:**
- Create: `apps/server/internal/domain/datasource/finscope/source.go`
- Create: `apps/server/internal/domain/datasource/finscope/types.go`
- Create: `apps/server/internal/domain/datasource/finscope/watcher.go`
- Create: `apps/server/internal/domain/datasource/finscope/parser.go`

- [ ] **Step 1: 新建 `types.go`，定义 sourceName、Config、Option 和内部常量**

- [ ] **Step 2: 新建 `watcher.go`，定义 `pageResponseWatcher` 接口和浏览器未配置错误**

- [ ] **Step 3: 新建 `parser.go`，定义占位解析函数并返回未实现错误**

- [ ] **Step 4: 新建 `source.go`，实现 `Source`、`New(...)`、5 个 `datasource.Source` 方法和统一未实现错误**

- [ ] **Step 5: 运行 `gofmt` 格式化新增文件**

### Task 2: 新增骨架级测试

**Files:**
- Create: `apps/server/internal/domain/datasource/finscope/source_test.go`

- [ ] **Step 1: 写接口满足性测试，断言 `Source` 实现 `datasource.Source`**

- [ ] **Step 2: 写构造函数测试，断言 `New(nil)` 返回非空实例**

- [ ] **Step 3: 写空实现行为测试，断言 5 个公开方法返回显式错误而不是成功结果**

- [ ] **Step 4: 运行 `go test -timeout 120s ./internal/domain/datasource/finscope`**

- [ ] **Step 5: 扩大验证到 `go test -timeout 120s ./internal/domain/datasource/...`，确认没有破坏现有数据源**

### Task 3: 实现上证指数成分股真实页面抓取子方法

**Files:**
- Modify: `apps/server/internal/domain/datasource/finscope/source.go`
- Modify: `apps/server/internal/domain/datasource/finscope/types.go`
- Modify: `apps/server/internal/domain/datasource/finscope/watcher.go`
- Modify: `apps/server/internal/domain/datasource/finscope/parser.go`
- Modify: `apps/server/internal/domain/datasource/finscope/source_test.go`

- [ ] **Step 1: 在 `source.go` 中把 `ListStocks` 改为串行执行子方法列表，首个子方法命名为“上证指数成分股”**

- [ ] **Step 2: 在 `types.go` 中定义百度财经成分股页面常量、接口查询结构和滚动收口参数**

- [ ] **Step 3: 在 `browserfetch` 中补充页面前后附加动作能力，并支持“独立进程主页面 target + 原始 `page.Navigate`”模式，让抓取流程能在同页里预注入脚本并执行滚动等页面行为**

- [ ] **Step 4: 在 `watcher.go` 中实现“打开指数成分股页面 -> 预注入 `fetch/XMLHttpRequest` 拦截脚本 -> 触发滚动 -> 收集多批响应体”的封装**

- [ ] **Step 5: 在 `parser.go` 中实现百度财经成分股 JSON 到 `[]datasource.StockBasic` 的映射，补齐 `TSCode`、`Symbol`、`Name`、`Exchange`、`Market`、`Source`**

- [ ] **Step 6: 在 `source.go` 中把多批响应顺序解析并合并去重，维持 `ListStocks` 的串行子方法结构**

- [ ] **Step 7: 在 `source_test.go` 和 `browserfetch` 测试中新增页面驱动场景覆盖，验证入口页面、响应聚合以及页面前后附加动作执行**

- [ ] **Step 8: 运行 `go test -timeout 120s ./internal/domain/datasource/finscope ./internal/domain/datasource/... ./internal/infra/browserfetch`**
