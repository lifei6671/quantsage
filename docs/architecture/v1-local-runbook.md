# QuantSage V1 本地运行与冒烟手册

## 1. 目标

本文档用于在本地启动 QuantSage V1，并完成一次最小端到端冒烟验证。

V1 冒烟范围包括：

- 启动基础依赖：PostgreSQL、Redis、MinIO
- 执行数据库迁移
- 构建并验证后端与前端
- 启动 `quantsage-server`、`quantsage-worker` 与前端工作台
- 手动触发数据任务，检查股票、日线、信号和任务状态接口

## 2. 本地环境变量

请先准备以下环境变量，示例中的敏感值统一使用 `***` 占位：

```text
POSTGRES_DB=quantsage
POSTGRES_USER=quantsage
POSTGRES_PASSWORD=***
MINIO_ROOT_USER=quantsage
MINIO_ROOT_PASSWORD=***
QUANTSAGE_DATABASE_DSN=postgres://quantsage:***@127.0.0.1:5432/quantsage?sslmode=disable
QUANTSAGE_REDIS_ADDR=127.0.0.1:6379
```

说明：

- `configs/config.example.yaml` 默认使用 `local` 模式，未覆盖时服务监听 `:8080`
- `QUANTSAGE_DATABASE_DSN` 与 `QUANTSAGE_REDIS_ADDR` 会覆盖 YAML 中的默认值
- V1 本地样例运行时依赖 `apps/server/testdata/sample`，无需真实 Tushare Token
- `datasource.default_source` 支持 `tushare` / `eastmoney`
- 配置 `datasource.tushare.token` 或 `QUANTSAGE_TUSHARE_TOKEN` 后，手动触发的 `sync_stock_basic`、`sync_trade_calendar`、`sync_daily_market` 会改用 Tushare Pro HTTP 数据源
- 若将 `datasource.default_source` 切到 `eastmoney`，同一批导入任务会改走东财公开行情接口
- 东财抓取模式支持：
  - `datasource.eastmoney.fetch_mode=http`
  - `datasource.eastmoney.fetch_mode=auto`
  - `datasource.eastmoney.fetch_mode=chromedp`
- `auto` / `chromedp` 模式下还应关注：
  - `datasource.eastmoney.browser_path`
  - `datasource.eastmoney.browser_timeout_seconds`
  - `datasource.eastmoney.browser_cookie_ttl_seconds`
  - `datasource.eastmoney.browser_headless`
  - `datasource.eastmoney.browser_user_agent_mode`
  - `datasource.eastmoney.browser_user_agent_platform`
  - `datasource.eastmoney.browser_count`
  - `datasource.eastmoney.browser_max_concurrent_tabs`
  - `datasource.eastmoney.browser_tabs_per_browser`
  - `datasource.eastmoney.browser_recycle_after_tabs`
  - `datasource.eastmoney.browser_wait_ready_selector`
  - `datasource.eastmoney.browser_accept_language`
  - `datasource.eastmoney.browser_disable_images`
  - `datasource.eastmoney.browser_no_sandbox`
  - `datasource.eastmoney.browser_window_width`
  - `datasource.eastmoney.browser_window_height`
  - `datasource.eastmoney.browser_blocked_url_patterns`
  - `datasource.eastmoney.browser_extra_flags`
- `browserfetch.Runner` 使用 Chrome Pool：每个 worker 维护一个 Chrome / Chromium 进程，进程内按 tab 并发抓取；总并发 = `browser_count * browser_tabs_per_browser`
- 单个 Chrome 进程累计打开 `browser_recycle_after_tabs` 个 tab 后，会在该 worker 没有活跃 tab 时回收重启；服务退出或测试清理时应调用 `Close(ctx)` 释放全部进程
- `browser_max_concurrent_tabs` 保留为旧配置兼容字段；新配置优先使用 `browser_tabs_per_browser`
- `browser_user_agent_mode=stable/mobile/custom` 会优先用 `github.com/lib4u/fake-useragent` 生成 Chrome UA，生成失败或结果为空时回退到项目内置固定 UA；`default` 保留浏览器原生 UA
- 东财交易日历当前通过上证综指日线推导沪深北统一 A 股交易日，属于显式实现假设，不是官方日历接口
- 若启用 `auto` / `chromedp`，本机还需要可用的 Chrome / Chromium；当 `browser_path` 为空时，当前实现依赖 `chromedp` 默认可执行文件解析，不做项目内额外平台探测

## 3. 启动基础依赖

在仓库根目录执行：

```bash
docker compose -f deployments/docker-compose/docker-compose.yml up -d
```

默认会拉起：

- PostgreSQL / TimescaleDB：`127.0.0.1:5432`
- Redis：`127.0.0.1:6379`
- MinIO：`127.0.0.1:9000`，控制台 `127.0.0.1:9001`

## 4. 执行迁移与基础验证

在仓库根目录执行：

```bash
make migrate-up
make build
make test
make race
make lint
cd apps/web && npm install && npm run build
```

预期：

- 后端可完成编译、单测、竞态检测和静态检查
- 前端生产构建成功

## 5. 启动服务

建议打开三个终端。

### 终端 A：启动 server

```bash
go run ./apps/server/cmd/quantsage-server -config configs/config.example.yaml
```

预期：

- 本地监听 `http://127.0.0.1:8080`
- `GET /api/healthz` 返回 `{"code":0,...,"data":{"status":"ok"}}`

### 终端 B：启动 worker

```bash
go run ./apps/server/cmd/quantsage-worker -config configs/config.example.yaml
```

说明：

- local worker 只负责 cron 调度
- 真正的 sample runtime 只保留在 server 进程里，避免 worker 与 API 各自维护独立内存态
- richer K 线查询服务当前只存在于后端领域层，尚未开放独立 HTTP 路由
- richer K 线与导入链路都复用同一套 EastMoney history browser fallback

### 终端 C：启动前端

```bash
cd apps/web
npm run dev
```

预期：

- 前端监听 `http://127.0.0.1:4173`
- 页面使用 `HashRouter`
- `/api/*` 请求会自动通过 Vite proxy 代理到 `http://127.0.0.1:8080`
- 如果后端监听地址不是默认值，可在前端启动时设置 `VITE_DEV_API_PROXY_TARGET`，例如 `VITE_DEV_API_PROXY_TARGET=http://127.0.0.1:8090 npm run dev`

当前默认预置账号：

- 用户名：`admin`
- 密码：`admin123`

补充说明：

- 本地 `npm run dev` / `npm run preview` 默认通过 Vite proxy 走同源 `/api/*`，不需要额外配置 CORS 白名单，也不需要设置 `VITE_API_BASE_URL`
- 只有在显式设置 `VITE_API_BASE_URL` 做分开部署时，才需要同步配置服务端 `auth.allowed_origins`
- 若属于跨站点部署，服务端还需要把 `auth.session_same_site` 设为 `none`，并同时开启 `auth.session_secure: true`

## 6. 手动任务冒烟

在仓库根目录或任意能访问本地 API 的终端执行：

```bash
COOKIE_JAR=/tmp/quantsage-cookie.txt
curl -c "$COOKIE_JAR" -X POST http://127.0.0.1:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}'

curl -X POST http://127.0.0.1:8080/api/jobs/sync_stock_basic/run \
  -b "$COOKIE_JAR" \
  -H 'Content-Type: application/json' \
  -d '{"biz_date":"2026-04-27"}'

curl -X POST http://127.0.0.1:8080/api/jobs/sync_daily_market/run \
  -b "$COOKIE_JAR" \
  -H 'Content-Type: application/json' \
  -d '{"biz_date":"2026-04-27"}'

curl -X POST http://127.0.0.1:8080/api/jobs/calc_daily_factor/run \
  -b "$COOKIE_JAR" \
  -H 'Content-Type: application/json' \
  -d '{"biz_date":"2026-04-27"}'

curl -X POST http://127.0.0.1:8080/api/jobs/generate_strategy_signals/run \
  -b "$COOKIE_JAR" \
  -H 'Content-Type: application/json' \
  -d '{"biz_date":"2026-04-27"}'
```

每次成功返回都应类似：

```json
{"code":0,"errmsg":"","toast":"","data":{"job_name":"sync_daily_market","status":"queued"}}
```

## 7. 接口验收点

任务执行完成后，至少检查以下接口：

```bash
curl -b "$COOKIE_JAR" 'http://127.0.0.1:8080/api/stocks?page=1&page_size=20'
curl -b "$COOKIE_JAR" 'http://127.0.0.1:8080/api/stocks/000001.SZ/daily?start_date=2026-04-01&end_date=2026-04-30'
curl -b "$COOKIE_JAR" 'http://127.0.0.1:8080/api/signals?trade_date=2026-04-27&page=1&page_size=20'
curl -b "$COOKIE_JAR" 'http://127.0.0.1:8080/api/jobs?biz_date=2026-04-27&page=1&page_size=20'
```

预期：

- `GET /api/stocks` 返回样例股票
- `GET /api/stocks/000001.SZ/daily` 返回样例日线
- `GET /api/signals` 在信号任务执行后返回确定性的策略信号数据
- `GET /api/jobs` 返回 `job_run_log` 风格的任务记录

如果将导入源切到 `eastmoney`，还应额外关注：

- 全市场日线导入会先拉取 A 股股票列表，再逐证券抓取东财日 K
- `fetch_mode=http` 时，东财返回 HTML / 人机验证页会直接以 `datasource unavailable` 失败
- `fetch_mode=auto` 时，命中 HTML / 人机验证页会先尝试浏览器 Cookie 回退一次；若浏览器不可用或回退后仍失败，同样会返回 `datasource unavailable`
- `fetch_mode=chromedp` 时，每次都会先取浏览器 Cookie，再发真正的数据 HTTP 请求
- 当前 richer K 线能力未暴露 HTTP，因此接口验收仍以上述导入链路为准

## 8. 前端验收点

打开 `http://127.0.0.1:4173/#/login`，先使用默认账号登录，再确认：

- 未登录时访问 `#/watchlists` 或 `#/positions` 会被自动跳转到 `#/login`
- 登录成功后会进入默认工作台，并显示当前登录用户
- `#/watchlists` 可以新增分组、修改分组、添加股票和删除股票
- `#/positions` 可以新增持仓、编辑持仓和删除持仓
- `#/jobs` 页面上半区可手动触发任务，提交成功后会刷新股票、日线、信号和任务状态缓存
- `#/jobs` 页面下半区可以看到任务状态列表，并按任务名或业务日期过滤

## 9. 常见问题

### 9.1 `make lint` 报缓存或加载错误

当前 Makefile 已显式设置：

- `GOCACHE=/tmp/quantsage-go-build`
- `GOLANGCI_LINT_CACHE=/tmp/golangci-lint`

如果仍异常，优先确认：

- `apps/server/go.mod` 存在且可读
- 本地 Go 版本与项目依赖兼容
- `golangci-lint` 已安装
- 当前本地已观察到 `golangci-lint run ./...` 可能报 `context loading failed: no go files to analyze`，这是工具加载异常，不代表已定位到代码级 lint 问题

### 9.2 前端打开后接口请求失败

优先检查：

- `quantsage-server` 是否已监听 `127.0.0.1:8080`
- 前端是否通过 `npm run dev` 或 `npm run preview` 启动
- 是否误设置了 `VITE_API_BASE_URL`

## 10. 收尾命令

验证完成后，如需停止本地依赖：

```bash
docker compose -f deployments/docker-compose/docker-compose.yml down
```
