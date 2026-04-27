# QuantSage V1 数据底座实施计划

> **面向 Agent 工作者：** 必需子技能：使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans`，按任务逐项实施本计划。步骤使用复选框（`- [ ]`）语法追踪进度。

**目标：** 将 QuantSage V1 落地为可运行的单仓库工程，包含 Gin 后端、PostgreSQL/TimescaleDB/pgvector 数据底座、样例数据源兜底、统一 API 响应/错误模型、日线行情 API、基础指标计算、固定策略信号生成，以及最小可用的 React 工作台。

**架构：** V1 在 `apps/server` 下使用一个 Go 模块，提供两个二进制：`quantsage-server` 负责 HTTP API，`quantsage-worker` 负责定时/手动数据任务。PostgreSQL 是系统事实数据源；TimescaleDB hypertable 存储时序行情数据；pgvector 为后续 RAG 表预启用。`apps/web` 下的前端只消费 Gin API 暴露的稳定 JSON 契约，不直接访问存储层。

**技术栈：** Go 1.22+、Gin、pgx/v5、sqlc、goose、Redis/go-redis、gin-contrib/sessions Redis 存储、log/slog、robfig/cron、PostgreSQL 16 + TimescaleDB + pgvector、React + Vite + shadcn/ui + TanStack Query + ECharts。

---

## 1. 已锁定决策

- 后端框架：Gin。
- 日志：标准库 `log/slog`，JSON 输出。每条请求日志包含 `request_id`、`method`、`path`、`status`、`latency_ms`。
- 缓存：Redis。V1 用于 session 存储、任务状态缓存，并预留热点 API 缓存。
- Session：`gin-contrib/sessions` + Redis 存储。V1 优先使用 session 管理后台访问；JWT 不进入 V1 主路径。
- 数据访问：`pgx/v5` + `sqlc`，SQL 文件作为领域查询契约。
- 数据迁移：`goose`，迁移文件放在 `migrations/postgres`。
- 任务调度：`robfig/cron`，所有任务都写入 `job_run_log`。
- 数据源：V1 默认使用样例数据源。Tushare HTTP 数据源只保留接口和配置接线；没有 Token 时系统仍可完整运行。
- 策略 DSL：V1 不实现通用表达式 DSL。先使用固定 Go 策略 + JSON 参数化策略定义。
- 回测默认语义：收盘后生成信号，下一交易日开盘买入，T+1 后允许卖出，价格使用前复权数据，手续费/印花税/滑点可配置，停牌/涨跌停/无成交量场景记录未成交原因。
- API 响应统一使用 `{"code":0,"errmsg":"","toast":"","data":{}}`。
- Docker Compose 密码只来自环境变量，不提交明文默认密码。

## 2. 依赖审批清单

执行任务 1 前，需要一次性确认新增依赖，因为实施会修改 `go.mod`、`go.sum` 和前端依赖文件。

Go 依赖：

```text
github.com/gin-gonic/gin
github.com/gin-contrib/sessions
github.com/gin-contrib/sessions/redis
github.com/jackc/pgx/v5
github.com/redis/go-redis/v9
github.com/pressly/goose/v3
github.com/robfig/cron/v3
github.com/shopspring/decimal
github.com/stretchr/testify
```

开发工具：

```text
github.com/sqlc-dev/sqlc/cmd/sqlc
github.com/golangci/golangci-lint/cmd/golangci-lint
```

前端依赖：

```text
react
react-dom
vite
typescript
@tanstack/react-query
echarts
echarts-for-react
zustand
react-router-dom
tailwindcss
shadcn/ui components
```

## 3. 目标文件结构

```text
quantsage/
├── apps/
│   ├── server/
│   │   ├── cmd/
│   │   │   ├── quantsage-server/main.go
│   │   │   └── quantsage-worker/main.go
│   │   ├── internal/
│   │   │   ├── app/
│   │   │   ├── config/
│   │   │   ├── domain/
│   │   │   │   ├── datasource/
│   │   │   │   ├── indicator/
│   │   │   │   ├── job/
│   │   │   │   ├── marketdata/
│   │   │   │   ├── stock/
│   │   │   │   └── strategy/
│   │   │   ├── infra/
│   │   │   │   ├── cache/
│   │   │   │   ├── db/
│   │   │   │   ├── log/
│   │   │   │   └── scheduler/
│   │   │   ├── interfaces/http/
│   │   │   └── pkg/                    # 工具类需要统一存放在这里，不允许在非pkg包之外，写过小的且无业务含义的私有方法
│   │   │       ├── apperror/
│   │   │       ├── response/
│   │   │       └── requestid/
│   │   ├── sql/
│   │   │   ├── queries/
│   │   │   └── schema/
│   │   ├── testdata/
│   │   │   └── sample/
│   │   ├── go.mod
│   │   ├── go.sum
│   │   └── sqlc.yaml
│   └── web/
│       ├── src/
│       │   ├── app/
│       │   ├── components/
│       │   ├── features/
│       │   ├── lib/
│       │   └── pages/
│       ├── package.json
│       └── vite.config.ts
├── configs/
│   └── config.example.yaml
├── deployments/
│   └── docker-compose/docker-compose.yml
├── docs/
│   ├── api/
│   ├── architecture/
│   └── superpowers/
├── migrations/
│   └── postgres/
└── Makefile
```

## 4. 数据库 Schema 契约

迁移文件：

- `migrations/postgres/000001_enable_extensions.sql`
- `migrations/postgres/000002_core_schema.sql`
- `migrations/postgres/000003_timescale_hypertables.sql`

### 4.1 扩展

```sql
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
```

### 4.2 核心表

```sql
CREATE TABLE stock_basic (
    ts_code TEXT PRIMARY KEY,
    symbol TEXT NOT NULL,
    name TEXT NOT NULL,
    area TEXT,
    industry TEXT,
    market TEXT,
    exchange TEXT NOT NULL,
    list_date DATE,
    delist_date DATE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    source TEXT NOT NULL,
    source_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE trade_calendar (
    exchange TEXT NOT NULL,
    cal_date DATE NOT NULL,
    is_open BOOLEAN NOT NULL,
    pretrade_date DATE,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (exchange, cal_date)
);

CREATE TABLE stock_daily (
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    trade_date DATE NOT NULL,
    open NUMERIC(18,4) NOT NULL,
    high NUMERIC(18,4) NOT NULL,
    low NUMERIC(18,4) NOT NULL,
    close NUMERIC(18,4) NOT NULL,
    pre_close NUMERIC(18,4),
    change NUMERIC(18,4),
    pct_chg NUMERIC(10,4),
    vol NUMERIC(20,4) NOT NULL DEFAULT 0,
    amount NUMERIC(20,4) NOT NULL DEFAULT 0,
    source TEXT NOT NULL,
    data_status TEXT NOT NULL DEFAULT 'clean',
    source_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (ts_code, trade_date)
);

CREATE TABLE adj_factor (
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    trade_date DATE NOT NULL,
    adj_factor NUMERIC(20,8) NOT NULL,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (ts_code, trade_date)
);

CREATE TABLE financial_indicator (
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    report_period DATE NOT NULL,
    ann_date DATE,
    end_date DATE NOT NULL,
    eps NUMERIC(18,6),
    diluted_eps NUMERIC(18,6),
    roe NUMERIC(18,6),
    roa NUMERIC(18,6),
    gross_margin NUMERIC(18,6),
    net_profit_margin NUMERIC(18,6),
    debt_to_assets NUMERIC(18,6),
    current_ratio NUMERIC(18,6),
    revenue_yoy NUMERIC(18,6),
    profit_yoy NUMERIC(18,6),
    source TEXT NOT NULL,
    version TEXT NOT NULL DEFAULT 'v1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (ts_code, report_period, version)
);

CREATE TABLE stock_factor_daily (
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    trade_date DATE NOT NULL,
    ma5 NUMERIC(18,4),
    ma10 NUMERIC(18,4),
    ma20 NUMERIC(18,4),
    ma60 NUMERIC(18,4),
    ema12 NUMERIC(18,6),
    ema26 NUMERIC(18,6),
    macd_dif NUMERIC(18,6),
    macd_dea NUMERIC(18,6),
    macd_hist NUMERIC(18,6),
    rsi6 NUMERIC(10,4),
    rsi12 NUMERIC(10,4),
    rsi24 NUMERIC(10,4),
    volume_ma5 NUMERIC(20,4),
    volume_ma20 NUMERIC(20,4),
    volume_ratio NUMERIC(10,4),
    amplitude NUMERIC(10,4),
    upper_shadow_ratio NUMERIC(10,4),
    lower_shadow_ratio NUMERIC(10,4),
    close_above_ma5 BOOLEAN,
    close_above_ma10 BOOLEAN,
    close_above_ma20 BOOLEAN,
    ma_bullish BOOLEAN,
    volume_breakout BOOLEAN,
    price_breakout_20 BOOLEAN,
    factor_version TEXT NOT NULL DEFAULT 'v1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (ts_code, trade_date, factor_version)
);

CREATE TABLE stock_event_tag (
    id BIGSERIAL PRIMARY KEY,
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    trade_date DATE NOT NULL,
    event_type TEXT NOT NULL,
    event_level TEXT NOT NULL DEFAULT 'info',
    score NUMERIC(10,4),
    description TEXT NOT NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    version TEXT NOT NULL DEFAULT 'v1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE strategy_definition (
    id BIGSERIAL PRIMARY KEY,
    strategy_code TEXT NOT NULL,
    strategy_name TEXT NOT NULL,
    strategy_type TEXT NOT NULL,
    description TEXT,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    version TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (strategy_code, version)
);

CREATE TABLE strategy_signal (
    id BIGSERIAL PRIMARY KEY,
    strategy_code TEXT NOT NULL,
    strategy_version TEXT NOT NULL,
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    trade_date DATE NOT NULL,
    signal_type TEXT NOT NULL,
    signal_strength NUMERIC(10,4) NOT NULL DEFAULT 0,
    signal_level TEXT NOT NULL DEFAULT 'D',
    buy_price_ref NUMERIC(18,4),
    stop_loss_ref NUMERIC(18,4),
    take_profit_ref NUMERIC(18,4),
    invalidation_condition TEXT,
    reason TEXT NOT NULL,
    input_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (strategy_code, strategy_version, ts_code, trade_date, signal_type)
);

CREATE TABLE job_run_log (
    id BIGSERIAL PRIMARY KEY,
    job_name TEXT NOT NULL,
    biz_date DATE,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    error_code INT NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    retry_count INT NOT NULL DEFAULT 0,
    progress_current INT NOT NULL DEFAULT 0,
    progress_total INT NOT NULL DEFAULT 0,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE watchlist (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (name, ts_code)
);

CREATE TABLE position (
    id BIGSERIAL PRIMARY KEY,
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    position_date DATE NOT NULL,
    quantity NUMERIC(20,4) NOT NULL,
    cost_price NUMERIC(18,4) NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 4.3 RAG/AI 预留表

```sql
CREATE TABLE announcement (
    id BIGSERIAL PRIMARY KEY,
    ts_code TEXT REFERENCES stock_basic(ts_code),
    title TEXT NOT NULL,
    announcement_type TEXT NOT NULL DEFAULT 'unknown',
    publish_time TIMESTAMPTZ,
    source TEXT NOT NULL,
    source_url TEXT,
    object_key TEXT,
    content_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE document_chunk (
    id BIGSERIAL PRIMARY KEY,
    document_type TEXT NOT NULL,
    document_id BIGINT NOT NULL,
    chunk_index INT NOT NULL,
    content TEXT NOT NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (document_type, document_id, chunk_index)
);

CREATE TABLE document_embedding (
    chunk_id BIGINT PRIMARY KEY REFERENCES document_chunk(id) ON DELETE CASCADE,
    embedding vector(1536),
    embedding_model TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ai_stock_analysis (
    id BIGSERIAL PRIMARY KEY,
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    analysis_date DATE NOT NULL,
    analysis_type TEXT NOT NULL,
    prompt_version TEXT NOT NULL,
    conclusion TEXT NOT NULL,
    evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
    risks JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 4.4 索引与 Hypertable

```sql
CREATE INDEX idx_stock_basic_symbol ON stock_basic(symbol);
CREATE INDEX idx_stock_basic_name_trgm ON stock_basic USING gin (name gin_trgm_ops);
CREATE INDEX idx_stock_daily_date ON stock_daily(trade_date);
CREATE INDEX idx_adj_factor_date ON adj_factor(trade_date);
CREATE INDEX idx_financial_indicator_period ON financial_indicator(report_period);
CREATE INDEX idx_stock_factor_daily_date ON stock_factor_daily(trade_date);
CREATE INDEX idx_stock_event_tag_code_date ON stock_event_tag(ts_code, trade_date);
CREATE INDEX idx_stock_event_tag_type_date ON stock_event_tag(event_type, trade_date);
CREATE INDEX idx_strategy_signal_date ON strategy_signal(trade_date, strategy_code, signal_type);
CREATE INDEX idx_job_run_log_name_date ON job_run_log(job_name, biz_date);
CREATE INDEX idx_announcement_code_time ON announcement(ts_code, publish_time DESC);
CREATE INDEX idx_document_embedding_vector ON document_embedding USING ivfflat (embedding vector_cosine_ops);

SELECT create_hypertable('stock_daily', 'trade_date', if_not_exists => TRUE);
SELECT create_hypertable('adj_factor', 'trade_date', if_not_exists => TRUE);
SELECT create_hypertable('stock_factor_daily', 'trade_date', if_not_exists => TRUE);
```

## 5. API 契约

### 5.1 统一响应

文件：`apps/server/internal/pkg/response/response.go`

```go
package response

type Body struct {
    Code   int         `json:"code"`
    Errmsg string      `json:"errmsg"`
    Toast  string      `json:"toast"`
    Data   any         `json:"data"`
}
```

### 5.2 错误码

文件：`apps/server/internal/pkg/apperror/codes.go`

```go
package apperror

const (
    CodeOK                    = 0
    CodeBadRequest            = 400001
    CodeUnauthorized          = 401001
    CodeForbidden             = 403001
    CodeNotFound              = 404001
    CodeValidationFailed      = 422001
    CodeDatasourceUnavailable = 503001
    CodeJobRunning            = 409001
    CodeDatabaseError         = 500101
    CodeInternal              = 500001
)
```

文件：`apps/server/internal/pkg/apperror/messages.go`

```go
package apperror

var Messages = map[int]struct {
    Errmsg string
    Toast  string
}{
    CodeOK:                    {"", ""},
    CodeBadRequest:            {"bad request", "请求参数不正确"},
    CodeUnauthorized:          {"unauthorized", "请先登录"},
    CodeForbidden:             {"forbidden", "没有操作权限"},
    CodeNotFound:              {"not found", "数据不存在"},
    CodeValidationFailed:      {"validation failed", "提交内容不符合要求"},
    CodeDatasourceUnavailable: {"datasource unavailable", "数据源暂时不可用，请稍后重试"},
    CodeJobRunning:            {"job already running", "任务正在执行，请稍后查看结果"},
    CodeDatabaseError:         {"database error", "数据服务异常，请稍后重试"},
    CodeInternal:              {"internal error", "系统异常，请稍后重试"},
}
```

### 5.3 V1 接口

```text
GET  /api/healthz
GET  /api/stocks?keyword=&page=1&page_size=20
GET  /api/stocks/{ts_code}
GET  /api/stocks/{ts_code}/daily?start_date=2025-01-01&end_date=2026-04-27
GET  /api/stocks/{ts_code}/factors?start_date=2025-01-01&end_date=2026-04-27
GET  /api/signals?trade_date=2026-04-27&strategy_code=volume_breakout_v1&page=1&page_size=20
GET  /api/jobs?job_name=&biz_date=&page=1&page_size=20
POST /api/jobs/{job_name}/run
```

响应 DTO 文件：

```text
apps/server/internal/interfaces/http/dto/common.go
apps/server/internal/interfaces/http/dto/stock.go
apps/server/internal/interfaces/http/dto/signal.go
apps/server/internal/interfaces/http/dto/job.go
```

## 6. 实施任务

### 任务 1：单仓库 脚手架与工具链

**文件：**

- 新建：`Makefile`
- 新建：`apps/server/go.mod`
- 新建：`apps/server/cmd/quantsage-server/main.go`
- 新建：`apps/server/cmd/quantsage-worker/main.go`
- 新建：`apps/server/sqlc.yaml`
- 新建：`configs/config.example.yaml`
- 新建：`deployments/docker-compose/docker-compose.yml`
- 新建：`.golangci.yml`

- [ ] **步骤 1：初始化 Go 模块**

运行：

```bash
cd apps/server
go mod init github.com/lifei6671/quantsage/apps/server
```

预期：`apps/server/go.mod` 存在，且 模块路径 与上方一致。

- [ ] **步骤 2：添加已审批的 Go 依赖**

依赖审批后运行：

```bash
cd apps/server
go get github.com/gin-gonic/gin github.com/gin-contrib/sessions github.com/gin-contrib/sessions/redis github.com/jackc/pgx/v5 github.com/redis/go-redis/v9 github.com/pressly/goose/v3 github.com/robfig/cron/v3 github.com/shopspring/decimal github.com/stretchr/testify
```

预期：依赖出现在 `go.mod` 中。

- [ ] **步骤 3：创建 Server 和 Worker 入口**

`apps/server/cmd/quantsage-server/main.go`:

```go
package main

import "fmt"

func main() {
    fmt.Println("quantsage server bootstrap")
}
```

`apps/server/cmd/quantsage-worker/main.go`:

```go
package main

import "fmt"

func main() {
    fmt.Println("quantsage worker bootstrap")
}
```

- [ ] **步骤 4：添加 Makefile**

`Makefile`:

```makefile
.PHONY: build test race lint fmt tidy migrate-up migrate-down

build:
	cd apps/server && go build ./...

test:
	cd apps/server && go test -timeout 120s ./...

race:
	cd apps/server && go test -race -timeout 120s ./...

lint:
	cd apps/server && golangci-lint run ./...

fmt:
	gofmt -w apps/server

tidy:
	cd apps/server && go mod tidy

migrate-up:
	cd apps/server && goose -dir ../../migrations/postgres postgres "$$QUANTSAGE_DATABASE_DSN" up

migrate-down:
	cd apps/server && goose -dir ../../migrations/postgres postgres "$$QUANTSAGE_DATABASE_DSN" down
```

- [ ] **步骤 5：添加使用环境变量的 Docker Compose**

`deployments/docker-compose/docker-compose.yml` 必须使用：

```yaml
services:
  postgres:
    image: timescale/timescaledb:latest-pg16
    environment:
      POSTGRES_DB: ${POSTGRES_DB}
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
  redis:
    image: redis:7
  minio:
    image: minio/minio
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
```

- [ ] **步骤 6：验证脚手架**

运行：

```bash
make fmt
make tidy
make build
make test
```

预期：所有命令成功完成。

### 任务 2：后端基础能力

**文件：**

- 新建：`apps/server/internal/config/config.go`
- 新建：`apps/server/internal/infra/log/logger.go`
- 新建：`apps/server/internal/pkg/apperror/codes.go`
- 新建：`apps/server/internal/pkg/apperror/messages.go`
- 新建：`apps/server/internal/pkg/apperror/error.go`
- 新建：`apps/server/internal/pkg/response/response.go`
- 新建：`apps/server/internal/pkg/requestid/requestid.go`
- 新建：`apps/server/internal/interfaces/http/router.go`
- 新建：`apps/server/internal/interfaces/http/middleware/recovery.go`
- 新建：`apps/server/internal/interfaces/http/middleware/requestlog.go`
- 修改：`apps/server/cmd/quantsage-server/main.go`

- [ ] **步骤 1：实现配置加载器**

`config.Config` 必须包含：

```go
type Config struct {
    App struct {
        Name string `yaml:"name"`
        Env  string `yaml:"env"`
        Addr string `yaml:"addr"`
    } `yaml:"app"`
    Database struct {
        DSN string `yaml:"dsn"`
    } `yaml:"database"`
    Redis struct {
        Addr     string `yaml:"addr"`
        Password string `yaml:"password"`
        DB       int    `yaml:"db"`
    } `yaml:"redis"`
}
```

环境变量覆盖 YAML：

```text
QUANTSAGE_DATABASE_DSN
QUANTSAGE_REDIS_ADDR
QUANTSAGE_REDIS_PASSWORD
```

- [ ] **步骤 2：实现 `apperror.AppError`**

契约：

```go
type AppError struct {
    Code int
    Err  error
}

func New(code int, err error) *AppError
func CodeOf(err error) int
func MessageOf(code int) (errmsg string, toast string)
```

`CodeOf(nil)` 返回 `CodeOK`；未知错误返回 `CodeInternal`。

- [ ] **步骤 3：实现响应辅助函数**

契约：

```go
func OK(c *gin.Context, data any)
func Fail(c *gin.Context, err error)
```

`Fail` 必须将 HTTP status 设置为 `200`，并在 JSON body 中编码业务错误。

- [ ] **步骤 4：实现 Gin router**

Router 必须注册：

```text
GET /api/healthz
```

响应：

```json
{"code":0,"errmsg":"","toast":"","data":{"status":"ok"}}
```

- [ ] **步骤 5：验证基础能力**

运行：

```bash
cd apps/server
go test -timeout 120s ./internal/pkg/apperror ./internal/pkg/response ./internal/interfaces/http
go build ./...
```

预期：测试通过，二进制编译成功。

### 任务 3：数据库迁移与 sqlc 设置

**文件：**

- 新建：`migrations/postgres/000001_enable_extensions.sql`
- 新建：`migrations/postgres/000002_core_schema.sql`
- 新建：`migrations/postgres/000003_timescale_hypertables.sql`
- 新建：`apps/server/sql/schema/schema.sql`
- 新建：`apps/server/sql/queries/stocks.sql`
- 新建：`apps/server/sql/queries/jobs.sql`
- 修改：`apps/server/sqlc.yaml`

- [ ] **步骤 1：添加迁移 DDL**

使用第 4 节中的精确 schema 契约。

- [ ] **步骤 2：添加 sqlc 配置**

`apps/server/sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "../../migrations/postgres"
    queries: "sql/queries"
    gen:
      go:
        package: "dbgen"
        out: "internal/infra/db/dbgen"
        sql_package: "pgx/v5"
```

- [ ] **步骤 3：添加股票查询 SQL**

`apps/server/sql/queries/stocks.sql`:

```sql
-- name: ListStocks :many
SELECT ts_code, symbol, name, area, industry, market, exchange, list_date, delist_date, is_active, source, updated_at
FROM stock_basic
WHERE ($1::text = '' OR ts_code ILIKE '%' || $1 || '%' OR symbol ILIKE '%' || $1 || '%' OR name ILIKE '%' || $1 || '%')
ORDER BY ts_code
LIMIT $2 OFFSET $3;

-- name: GetStock :one
SELECT ts_code, symbol, name, area, industry, market, exchange, list_date, delist_date, is_active, source, updated_at
FROM stock_basic
WHERE ts_code = $1;

-- name: ListStockDaily :many
SELECT ts_code, trade_date, open, high, low, close, pre_close, change, pct_chg, vol, amount, source, data_status
FROM stock_daily
WHERE ts_code = $1 AND trade_date BETWEEN $2 AND $3
ORDER BY trade_date;
```

- [ ] **步骤 4：生成 sqlc 代码**

运行：

```bash
cd apps/server
sqlc generate
```

预期：`apps/server/internal/infra/db/dbgen` 包含生成后的 Go 文件。

### 任务 4：样例数据源与导入任务

**文件：**

- 新建：`apps/server/internal/domain/datasource/types.go`
- 新建：`apps/server/internal/domain/datasource/sample/source.go`
- 新建：`apps/server/internal/domain/datasource/tushare/source.go`
- 新建：`apps/server/internal/domain/job/import_stock_basic.go`
- 新建：`apps/server/internal/domain/job/import_trade_calendar.go`
- 新建：`apps/server/internal/domain/job/import_stock_daily.go`
- 新建：`apps/server/testdata/sample/stock_basic.json`
- 新建：`apps/server/testdata/sample/trade_calendar.json`
- 新建：`apps/server/testdata/sample/stock_daily.json`

- [ ] **步骤 1：定义数据源契约**

```go
type StockBasic struct {
    TSCode   string
    Symbol   string
    Name     string
    Area     string
    Industry string
    Market   string
    Exchange string
    ListDate time.Time
    Source   string
}

type DailyBar struct {
    TSCode    string
    TradeDate time.Time
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

type Source interface {
    ListStocks(ctx context.Context) ([]StockBasic, error)
    ListTradeCalendar(ctx context.Context, exchange string, startDate, endDate time.Time) ([]TradeDay, error)
    ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]DailyBar, error)
}
```

- [ ] **步骤 2：实现样例数据源**

样例数据源读取本地 JSON 文件，并为以下股票返回确定性数据：

```text
000001.SZ 平安银行
600519.SH 贵州茅台
300750.SZ 宁德时代
```

- [ ] **步骤 3：实现 Tushare 未启用行为**

如果 `TUSHARE_TOKEN` 为空，Tushare 数据源返回 `CodeDatasourceUnavailable`，错误消息为 `tushare token is empty`。

- [ ] **步骤 4：实现导入任务**

每个导入任务必须：

1. 创建一条 `status='running'` 的 `job_run_log` 记录。
2. 写入或更新导入的数据行。
3. 完成后将状态设置为 `success`。
4. 失败时将状态设置为 `failed`，并记录包装后的错误。

- [ ] **步骤 5：验证数据源**

运行：

```bash
cd apps/server
go test -timeout 120s ./internal/domain/datasource/... ./internal/domain/job/...
```

预期：样例数据源测试通过，且不依赖外部网络。

### 任务 5：股票 API

**文件：**

- 新建：`apps/server/internal/interfaces/http/dto/common.go`
- 新建：`apps/server/internal/interfaces/http/dto/stock.go`
- 新建：`apps/server/internal/interfaces/http/handler/stock_handler.go`
- 新建：`apps/server/internal/domain/stock/service.go`
- 修改：`apps/server/internal/interfaces/http/router.go`

- [ ] **步骤 1：定义分页 DTO**

```go
type PageRequest struct {
    Page     int `form:"page"`
    PageSize int `form:"page_size"`
}

type PageResponse[T any] struct {
    Items    []T `json:"items"`
    Page     int `json:"page"`
    PageSize int `json:"page_size"`
}
```

- [ ] **步骤 2：定义股票 DTO**

```go
type StockItem struct {
    TSCode   string `json:"ts_code"`
    Symbol   string `json:"symbol"`
    Name     string `json:"name"`
    Industry string `json:"industry"`
    Exchange string `json:"exchange"`
    IsActive bool   `json:"is_active"`
}

type DailyBarItem struct {
    TSCode    string `json:"ts_code"`
    TradeDate string `json:"trade_date"`
    Open      string `json:"open"`
    High      string `json:"high"`
    Low       string `json:"low"`
    Close     string `json:"close"`
    PctChg    string `json:"pct_chg"`
    Vol       string `json:"vol"`
    Amount    string `json:"amount"`
}
```

- [ ] **步骤 3：注册接口**

```text
GET /api/stocks
GET /api/stocks/:ts_code
GET /api/stocks/:ts_code/daily
```

- [ ] **步骤 4：验证 API 测试**

运行：

```bash
cd apps/server
go test -timeout 120s ./internal/interfaces/http/...
```

预期：处理器测试断言精确 JSON 结构，包含 `code`、`errmsg`、`toast`、`data`。

### 任务 6：指标计算

**文件：**

- 新建：`apps/server/internal/domain/indicator/types.go`
- 新建：`apps/server/internal/domain/indicator/calculator.go`
- 新建：`apps/server/internal/domain/indicator/calculator_test.go`
- 新建：`apps/server/internal/domain/job/calc_daily_factor.go`

- [ ] **步骤 1：实现纯计算器**

公开方法：

```go
func CalculateDailyFactors(bars []marketdata.DailyBar) ([]DailyFactor, error)
```

规则：

1. 输入 K 线数据必须按 `TradeDate` 升序排序。
2. MA 使用简单移动平均。
3. EMA 使用标准平滑系数 `2/(n+1)`。
4. MACD 使用 EMA12、EMA26、DEA9。
5. 当分母为零时，RSI 返回空值。
6. 影线比例使用 `(high - max(open, close)) / (high - low)` 和 `(min(open, close) - low) / (high - low)`。

- [ ] **步骤 2：添加表驱动测试**

测试必须覆盖：

```text
MA 窗口数据不足
MA5 计算
volume_ratio 计算
振幅为零时的上下影线比例
输入未排序时返回错误
```

- [ ] **步骤 3：实现因子计算任务**

任务按日期范围读取 `stock_daily`，按股票计算因子，并写入或更新到 `stock_factor_daily`。

### 任务 7：固定策略信号

**文件：**

- 新建：`apps/server/internal/domain/strategy/types.go`
- 新建：`apps/server/internal/domain/strategy/volume_breakout.go`
- 新建：`apps/server/internal/domain/strategy/trend_break.go`
- 新建：`apps/server/internal/domain/strategy/scoring.go`
- 新建：`apps/server/internal/domain/strategy/volume_breakout_test.go`
- 新建：`apps/server/internal/domain/job/generate_strategy_signals.go`

- [ ] **步骤 1：定义策略结果**

```go
type SignalResult struct {
    StrategyCode          string
    StrategyVersion       string
    TSCode                string
    TradeDate             time.Time
    SignalType            string
    SignalStrength        decimal.Decimal
    SignalLevel           string
    BuyPriceRef           decimal.Decimal
    StopLossRef           decimal.Decimal
    TakeProfitRef         decimal.Decimal
    InvalidationCondition string
    Reason                string
    InputSnapshot         map[string]any
}
```

- [ ] **步骤 2：实现放量突破策略**

触发条件：

```text
close_today > highest(high, 20, exclude_today=true)
volume_today > volume_ma20 * 1.8
pct_chg_today > 3
close_today > ma20
upper_shadow_ratio < 0.25
```

信号等级：

```text
score >= 80 => A
score >= 60 => B
score >= 40 => C
otherwise => D
```

- [ ] **步骤 3：实现趋势破位策略**

触发条件：

```text
close < ma20
volume > volume_ma20 * 1.2
```

- [ ] **步骤 4：验证策略测试**

运行：

```bash
cd apps/server
go test -timeout 120s ./internal/domain/strategy/...
```

预期：测试覆盖命中、未命中、历史数据不足、分数边界等场景。

### 任务 8：任务调度器与手动任务 API

**文件：**

- 新建：`apps/server/internal/infra/scheduler/scheduler.go`
- 新建：`apps/server/internal/domain/job/runner.go`
- 新建：`apps/server/internal/interfaces/http/dto/job.go`
- 新建：`apps/server/internal/interfaces/http/handler/job_handler.go`
- 修改：`apps/server/cmd/quantsage-worker/main.go`
- 修改：`apps/server/internal/interfaces/http/router.go`

- [ ] **步骤 1：定义任务执行器契约**

```go
type Runner interface {
    Run(ctx context.Context, jobName string, bizDate time.Time) error
}
```

- [ ] **步骤 2：注册 V1 任务**

```text
sync_stock_basic
sync_trade_calendar
sync_daily_market
calc_daily_factor
generate_strategy_signals
```

- [ ] **步骤 3：添加手动触发接口**

```text
POST /api/jobs/:job_name/run
```

请求：

```json
{"biz_date":"2026-04-27"}
```

响应：

```json
{"code":0,"errmsg":"","toast":"","data":{"job_name":"sync_daily_market","status":"queued"}}
```

### 任务 9：信号 API

**文件：**

- 新建：`apps/server/internal/interfaces/http/dto/signal.go`
- 新建：`apps/server/internal/interfaces/http/handler/signal_handler.go`
- 新建：`apps/server/internal/domain/strategy/query_service.go`
- 修改：`apps/server/internal/interfaces/http/router.go`

- [ ] **步骤 1：定义信号 DTO**

```go
type SignalItem struct {
    StrategyCode          string `json:"strategy_code"`
    StrategyVersion       string `json:"strategy_version"`
    TSCode                string `json:"ts_code"`
    TradeDate             string `json:"trade_date"`
    SignalType            string `json:"signal_type"`
    SignalStrength        string `json:"signal_strength"`
    SignalLevel           string `json:"signal_level"`
    BuyPriceRef           string `json:"buy_price_ref"`
    StopLossRef           string `json:"stop_loss_ref"`
    TakeProfitRef         string `json:"take_profit_ref"`
    InvalidationCondition string `json:"invalidation_condition"`
    Reason                string `json:"reason"`
}
```

- [ ] **步骤 2：注册接口**

```text
GET /api/signals?trade_date=2026-04-27&strategy_code=volume_breakout_v1&page=1&page_size=20
```

### 任务 10：前端 V1 工作台

**文件：**

- 新建：`apps/web/package.json`
- 新建：`apps/web/src/lib/api.ts`
- 新建：`apps/web/src/lib/query.ts`
- 新建：`apps/web/src/app/App.tsx`
- 新建：`apps/web/src/pages/stocks/StockListPage.tsx`
- 新建：`apps/web/src/pages/stocks/StockDetailPage.tsx`
- 新建：`apps/web/src/pages/signals/SignalListPage.tsx`
- 新建：`apps/web/src/pages/jobs/JobListPage.tsx`

- [ ] **步骤 1：初始化 Vite React 应用**

运行：

```bash
cd apps/web
npm create vite@latest . -- --template react-ts
npm install
```

- [ ] **步骤 2：添加 API 客户端**

`api.ts` 只在 `code === 0` 时解包；否则抛出包含 `code`、`errmsg`、`toast` 的错误对象。

- [ ] **步骤 3：构建页面**

页面：

```text
/stocks       股票列表
/stocks/:id   个股日线
/signals      策略信号列表
/jobs         任务状态列表
```

- [ ] **步骤 4：验证前端**

运行：

```bash
cd apps/web
npm run build
```

预期：生产构建成功。

### 任务 11：本地端到端冒烟测试

**文件：**

- 新建：`docs/architecture/v1-local-runbook.md`
- 修改：`README.md`

- [ ] **步骤 1：记录本地环境变量**

必需变量：

```text
POSTGRES_DB=quantsage
POSTGRES_USER=quantsage
POSTGRES_PASSWORD=***
MINIO_ROOT_USER=quantsage
MINIO_ROOT_PASSWORD=***
QUANTSAGE_DATABASE_DSN=postgres://quantsage:***@127.0.0.1:5432/quantsage?sslmode=disable
QUANTSAGE_REDIS_ADDR=127.0.0.1:6379
```

- [ ] **步骤 2：冒烟流程**

运行：

```bash
docker compose -f deployments/docker-compose/docker-compose.yml up -d
make migrate-up
make test
make build
```

然后运行手动任务：

```bash
curl -X POST http://127.0.0.1:8080/api/jobs/sync_stock_basic/run -d '{"biz_date":"2026-04-27"}'
curl -X POST http://127.0.0.1:8080/api/jobs/sync_daily_market/run -d '{"biz_date":"2026-04-27"}'
curl -X POST http://127.0.0.1:8080/api/jobs/calc_daily_factor/run -d '{"biz_date":"2026-04-27"}'
curl -X POST http://127.0.0.1:8080/api/jobs/generate_strategy_signals/run -d '{"biz_date":"2026-04-27"}'
```

预期：

```text
GET /api/stocks 返回样例股票
GET /api/stocks/000001.SZ/daily 返回样例日线
GET /api/signals 在信号任务执行后返回确定性的策略信号数据
GET /api/jobs 返回 job_run_log 记录
```

## 7. 验证关口

每个任务都必须先通过本地验证，再进入下一个任务。

V1 最终验证：

```bash
make fmt
make tidy
make build
make test
make race
make lint
cd apps/web && npm run build
```

如果执行环境未安装 `golangci-lint`，先安装已审批的工具，再重新运行 `make lint`。

## 8. V1 范围外

- 通用策略 DSL 解析器。
- 真实 Tushare Token 验证。
- 公告 PDF 解析、Embedding、RAG 检索。
- 实时行情、WebSocket、盘中预警。
- 完整回测页面和 AI 复盘页面。
- 细粒度用户权限体系；V1 只保留基础 session 能力。

## 9. 自检记录

- 后端框架选型已从 GoFrame/Gin/Chi 收敛为 Gin。
- 没有 Tushare Token 的阻塞已通过样例数据源消除。
- API 错误码与 toast 消息集中维护在 `apperror`。
- 数据库表覆盖 V1 必需表，并为 RAG/AI 预留低耦合表。
- DSL 延后；V1 策略使用固定 Go 代码 + JSON 参数配置。
- Docker Compose 不包含明文密码。
