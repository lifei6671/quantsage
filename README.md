# quantsage

QuantSage 是一个面向本地研究与验证场景的 AI 量化工作台，当前 `V1` 提供：

- `apps/server`：Gin API、样例数据运行时、手动任务与本地 worker 调度
- `apps/web`：带登录态的股票、日线、策略信号、任务、自选股和持仓工作台
- `deployments/docker-compose`：本地 PostgreSQL / Redis / MinIO 依赖

当前后端已同时支持 `tushare` 和 `eastmoney` 两类导入源：

- `datasource.default_source=tushare`：继续走 Tushare Pro 导入股票基础、交易日历和日线
- `datasource.default_source=eastmoney`：切到东财公开行情导入同一批任务
- 未配置 `tushare token` 且处于 `local` 模式时，仍会回退到 `apps/server/testdata/sample`

东财链路当前支持三种抓取模式：

- `datasource.eastmoney.fetch_mode=http`：只走标准库 HTTP 请求
- `datasource.eastmoney.fetch_mode=auto`：默认先走 HTTP，命中 HTML / 人机验证页后自动取浏览器 Cookie 再重试一次
- `datasource.eastmoney.fetch_mode=chromedp`：每次先取浏览器 Cookie，再发真正的数据 HTTP 请求

启用 `auto` 或 `chromedp` 时，运行环境需要具备可用的 Chrome / Chromium：

- 可显式配置 `datasource.eastmoney.browser_path`
- 若 `browser_path` 为空，当前实现依赖 `chromedp` 的默认可执行文件解析，不做项目内额外平台探测
- 浏览器回退使用 Chrome Pool：`browser_count` 控制 Chrome 进程数，`browser_tabs_per_browser` 控制单进程并发 tab，总并发为二者乘积；单进程累计打开 `browser_recycle_after_tabs` 个 tab 后会在空闲时回收重启，降低长时间运行的内存泄露风险
- `browser_max_concurrent_tabs` 保留为旧配置兼容字段；新配置优先使用 `browser_tabs_per_browser`
- `browser_user_agent_mode=stable/mobile/custom` 会优先使用 `github.com/lib4u/fake-useragent` 生成 Chrome UA，生成失败时回退到内置固定 UA；`default` 则保留浏览器原生 UA
- 常用浏览器参数已开放：`browser_wait_ready_selector`、`browser_accept_language`、`browser_disable_images`、`browser_no_sandbox`、`browser_window_width`、`browser_window_height`、`browser_blocked_url_patterns`、`browser_extra_flags`
- richer K 线领域服务也复用同一套 history fallback 策略，但仍未暴露独立 HTTP 路由

## 仓库结构

- `apps/server`：后端服务与 worker
- `apps/web`：React + Vite 前端
- `configs/config.example.yaml`：本地默认配置
- `deployments/docker-compose/docker-compose.yml`：本地基础依赖
- `docs/architecture/v1-local-runbook.md`：V1 本地启动与冒烟手册

## 常用命令

```bash
make build
make test
make race
make lint
cd apps/web && npm run build
```

## 本地运行入口

完整步骤见 [docs/architecture/v1-local-runbook.md](/home/lifei6671/src/github.com/lifei6671/quantsage/docs/architecture/v1-local-runbook.md)。

最短路径：

```bash
docker compose -f deployments/docker-compose/docker-compose.yml up -d
make migrate-up
go run ./apps/server/cmd/quantsage-server -config configs/config.example.yaml
go run ./apps/server/cmd/quantsage-worker -config configs/config.example.yaml
cd apps/web && npm install && npm run dev
```

前端默认运行在 `http://127.0.0.1:4173/#/stocks`，并通过 Vite proxy 请求同源 `/api/*`。开发期后端代理目标默认是 `http://127.0.0.1:8080`，可用 `VITE_DEV_API_PROXY_TARGET` 覆盖。

东财 richer K 线能力当前已经在后端领域层落地，支持分钟线、周月季年线、复权、批量查询和均线/聚合计算；这部分复用 EastMoney browser fallback，但目前还没有对外 HTTP 路由。

## 本地默认账号

当前 `configs/config.example.yaml` 会在服务启动时同步一个预置管理员账号：

- 用户名：`admin`
- 密码：`admin123`

这个默认密码只用于本地开发和冒烟验证；若在共享环境使用，请先替换 `bootstrap_users.password_hash`。

如果前后端分开部署、需要跨域使用登录态：

- 后端必须显式配置 `auth.allowed_origins`
- 若浏览器需要跨站点携带 session cookie，还必须同时配置 `auth.session_same_site: none` 和 `auth.session_secure: true`
