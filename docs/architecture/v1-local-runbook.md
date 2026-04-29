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
