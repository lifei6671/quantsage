# QuantSage（量策智研）AI 股票分析系统技术方案（评审稿）

## 1. 文档信息

| 项目 | 内容 |
|---|---|
| 软件名 | QuantSage |
| 中文名 | 量策智研 |
| 仓库名 | quantsage |
| 项目代号 | QS |
| 文档名称 | QuantSage AI 股票分析系统技术方案 |
| 文档版本 | v1.1 |
| 技术方向 | 股票数据中台 + 策略信号引擎 + AI 投研分析助理 |
| 后端技术栈 | Golang |
| 前端技术栈 | React + shadcn/ui |
| 适用范围 | A 股行情数据、财务数据、公告资讯、技术指标、短线信号、AI 分析与复盘 |
| 目标形态 | 个人/小团队级股票数据中台 + AI 分析辅助系统 |
| 评审对象 | 架构设计、数据链路、策略信号、AI 分析、可扩展性、合规风险、落地成本 |

---

## 2. 名称与定位

### 2.1 项目命名

本项目命名为：

```text
软件名：QuantSage
中文名：量策智研
仓库名：quantsage
```

命名含义：

| 名称元素 | 含义 |
|---|---|
| Quant | 量化、数据、规则、策略、统计验证 |
| Sage | 理性分析、辅助判断、研究助理 |
| 量策 | 量化策略、规则化决策 |
| 智研 | AI 辅助研究、智能复盘、投研分析 |

### 2.2 产品定位

QuantSage 不是“自动荐股系统”，也不是“AI 预测股价系统”，而是：

> 一个以数据为基础、以规则为约束、以回测为验证、以 AI 为解释和归因能力的股票分析辅助系统。

系统核心定位：

```text
股票数据中台
+ 技术指标计算平台
+ 策略信号引擎
+ 回测验证系统
+ AI 投研分析助理
+ 持仓复盘与预警系统
```

### 2.3 核心原则

```text
买卖点由公式和规则计算；
信号有效性由回测验证；
AI 负责解释、归因、过滤、相似案例和交易计划；
最终交易决策由人工完成。
```

---

## 3. 背景与问题定义

### 3.1 背景

当前短线交易和股票分析主要依赖券商软件、同花顺、东方财富、通达信等平台。这些工具在行情展示、交易下单、基础资讯方面成熟，但在个性化数据沉淀、策略回测、AI 归因分析、相似案例检索、持仓复盘、交易计划生成等方面存在限制。

随着大语言模型、向量检索、时间序列数据库和本地数据分析能力逐渐成熟，可以构建一套面向个人投资者或小型研究团队的股票数据与 AI 分析系统。

QuantSage 旨在解决以下问题：

1. 将分散的行情、财务、公告、新闻、板块、技术指标数据统一沉淀。
2. 将短线交易中常见的“放量突破”“缩量回踩”“趋势破位”“放量滞涨”等经验规则公式化。
3. 通过回测验证规则信号的真实有效性。
4. 通过 AI 对信号进行解释、归因、风险识别和交易计划生成。
5. 建立持仓复盘、自选股预警、财报解读和相似案例分析能力。

### 3.2 当前痛点

| 痛点 | 表现 |
|---|---|
| 数据割裂 | 行情、财务、公告、新闻、板块、资金流分散在不同平台 |
| 数据不可沉淀 | 看盘判断难以结构化留存，历史复盘成本高 |
| 信号不可验证 | 常见技术形态缺少历史统计和回测验证 |
| AI 缺少数据底座 | 直接询问 AI 容易产生空泛判断或幻觉 |
| 短线决策情绪化 | 缺少规则化买卖条件、止损条件、失效条件 |
| 复盘低效 | 每日持仓、板块、个股异动依赖人工反复查看 |
| 策略不可复现 | 指标、参数、信号版本没有统一管理 |

### 3.3 方案目标

建设 QuantSage 的目标是形成一个：

> 数据可追溯、信号可回测、逻辑可解释、风险可控制的 AI 股票分析辅助系统。

系统不追求让 AI 直接给出确定性买卖结论，而是通过数据和规则帮助用户提高决策质量。

---

## 4. 建设目标与边界

### 4.1 建设目标

#### 4.1.1 数据目标

1. 建立统一股票主数据体系。
2. 存储 A 股日线、分钟线、复权因子、交易日历等基础行情数据。
3. 存储财务报表、财务指标、估值指标、行业板块、概念题材等结构化数据。
4. 存储公告、财报 PDF、新闻、研报摘要等非结构化文本数据。
5. 建立技术指标、事件标签、策略信号、AI 分析结果等衍生数据层。
6. 保留数据来源、更新时间、数据版本，保证可追溯。

#### 4.1.2 分析目标

1. 支持个股趋势、量价、均线、技术指标分析。
2. 支持短线买卖点候选信号计算。
3. 支持信号历史回测与统计。
4. 支持财报、公告、新闻的 AI 解读。
5. 支持相似案例检索。
6. 支持自选股和持仓每日 AI 复盘。
7. 支持盘中预警和盘后分析。

#### 4.1.3 工程目标

1. 后端以 Golang 为主语言，保证服务稳定性、并发能力和部署简洁性。
2. 前端采用 React + shadcn/ui，提供现代化管理台和分析工作台。
3. 数据同步、指标计算、策略信号、AI 分析均采用模块化设计。
4. 数据链路可追踪，任务可调度、可重跑、可补数。
5. 指标与信号规则可配置、可版本化。
6. AI 分析基于真实数据和检索结果，避免无依据输出。
7. 系统可从单机版平滑演进到分布式版本。

### 4.2 非目标

本阶段不实现以下能力：

1. 不做全自动交易。
2. 不做高频交易。
3. 不做毫秒级低延迟撮合级行情系统。
4. 不承诺预测股价。
5. 不直接替代人工交易决策。
6. 不做面向公众的投资建议服务。
7. 不存储非法或未授权的数据源。
8. 不在第一阶段接入全市场 Level-2 tick 数据。

---

## 5. Python 是否必须

### 5.1 结论

Python 不是必须的。

在 QuantSage 的主系统设计中，可以完全采用：

```text
Golang 后端
React + shadcn/ui 前端
PostgreSQL + TimescaleDB 数据库
pgvector 向量检索
DuckDB / Parquet 离线分析
```

完成数据同步、指标计算、策略信号、AI 分析、回测、预警和前端展示。

### 5.2 Python 的价值

Python 的优势主要在于：

| 场景 | Python 优势 |
|---|---|
| 数据源 SDK | Tushare、AKShare 等生态更成熟 |
| 量化研究 | pandas、numpy、vectorbt、backtrader 等工具丰富 |
| 机器学习 | scikit-learn、PyTorch、LightGBM 等生态成熟 |
| Notebook 分析 | Jupyter 适合探索式研究 |
| PDF/文本处理 | 一些 NLP 和文档处理库更丰富 |

### 5.3 Python 的问题

如果把 Python 作为主后端，会带来以下问题：

| 问题 | 影响 |
|---|---|
| 服务治理复杂度 | 长期运行的 API、Worker 稳定性需要更多约束 |
| 类型约束弱 | 大型工程长期演进容易产生隐性错误 |
| 部署一致性 | 包依赖和运行环境管理复杂度较高 |
| 性能边界 | 高并发 API、实时预警、流式处理不如 Go 简洁稳定 |

### 5.4 推荐决策

QuantSage 建议采用：

```text
主工程：Golang
研究工具：Python 可选
```

也就是说：

| 模块 | 推荐语言 |
|---|---|
| API 服务 | Go |
| 数据同步 Worker | Go |
| 指标计算 | Go |
| 策略信号 | Go |
| 回测引擎 | Go 优先，Python 可作为研究验证工具 |
| AI 工具调用 | Go |
| RAG 检索服务 | Go |
| 前端 | React + shadcn/ui |
| 探索式研究 | Python 可选 |
| 机器学习训练 | Python 可选 |

### 5.5 什么时候需要 Python

只有在以下场景中，建议引入 Python：

1. 快速使用 Tushare / AKShare SDK 做原型验证。
2. 使用 pandas / vectorbt 做探索式策略研究。
3. 训练机器学习模型。
4. 做 Notebook 式投研分析。
5. 某些文档解析、NLP 工具 Go 生态不够成熟。

### 5.6 架构建议

不要把 Python 放进主链路作为必需运行时。

更合理的形态是：

```text
Go 主系统：生产链路、任务调度、API、指标、信号、AI 分析
Python 辅助工具：研究脚本、Notebook、策略原型、模型训练
```

Python 可以存在于：

```text
research/
notebooks/
scripts/
ml/
```

但不应成为系统启动和核心功能运行的硬依赖。

---

## 6. 设计原则

### 6.1 数据优先原则

AI 分析必须基于结构化数据、文本检索结果、事件标签和历史统计，不允许单纯凭模型语言能力生成交易判断。

### 6.2 规则可解释原则

买卖点候选信号应优先由公式、规则、评分模型生成。AI 负责解释、过滤、归因和生成交易计划，而不是直接作为信号源。

### 6.3 回测验证原则

任何策略信号上线前必须经过历史回测，至少输出胜率、平均收益、盈亏比、最大回撤、交易次数、持有周期等指标。

### 6.4 原始数据保留原则

所有外部数据应保留 raw 层，清洗后的数据进入 cleaned 层，衍生指标进入 feature 层，AI 分析结果进入 analysis 层。

### 6.5 可重算原则

技术指标、事件标签、策略信号必须支持按股票、按日期、按策略版本重算。

### 6.6 Go 优先原则

主系统以 Golang 实现，尽量减少多语言运行时对部署、监控、运维和排障的影响。

### 6.7 合规边界原则

系统应区分个人研究、内部辅助和对外服务三种使用场景。未经授权的数据不得用于商业分发或对外服务。

---

## 7. 总体架构

### 7.1 架构概览

```text
┌──────────────────────────────────────────────┐
│                  数据源层                    │
│ Tushare / AKShare / QMT / 交易所公告 / 新闻  │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│                  采集层                      │
│ Go Data Worker / Realtime Worker / Parser     │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│                  数据存储层                  │
│ PostgreSQL / TimescaleDB / Parquet / pgvector │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│                  计算加工层                  │
│ Go 指标计算 / 事件标签 / 策略信号 / 回测      │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│                  AI 分析层                   │
│ LLM Tool Calling / RAG / SQL Tool / 归因分析  │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│                  应用层                      │
│ React + shadcn/ui / API / 自选股 / 每日复盘   │
└──────────────────────────────────────────────┘
```

### 7.2 后端逻辑架构

```text
quantsage-server
  ├─ API Gateway
  ├─ Auth Module
  ├─ Stock Master Service
  ├─ Market Data Service
  ├─ Financial Data Service
  ├─ Indicator Service
  ├─ Strategy Signal Service
  ├─ Backtest Service
  ├─ Document Service
  ├─ Vector Search Service
  ├─ AI Analysis Service
  ├─ Watchlist Service
  ├─ Alert Service
  └─ Job Scheduler / Worker
```

### 7.3 前端逻辑架构

```text
quantsage-web
  ├─ Dashboard
  ├─ Stock Detail
  ├─ Strategy Center
  ├─ Backtest Center
  ├─ AI Review
  ├─ Similar Cases
  ├─ Announcement & Report Search
  ├─ Watchlist
  ├─ Alert Center
  └─ System Tasks
```

### 7.4 分层职责

| 层级 | 责任 |
|---|---|
| 数据源层 | 接入外部行情、财务、公告、新闻、板块、实时行情等数据 |
| 采集层 | 定时同步、增量更新、补数、异常重试、原始数据落盘 |
| 存储层 | 存储结构化、时序、文本、向量、归档数据 |
| 计算层 | 计算技术指标、事件标签、买卖点信号、评分、回测 |
| AI 层 | 自然语言分析、公告财报解读、相似案例、交易计划生成 |
| 应用层 | Web 控制台、API、自选股预警、持仓复盘、策略管理 |

---

## 8. 技术选型

### 8.1 总体技术栈

| 模块 | 选型 | 说明 |
|---|---|---|
| 后端语言 | Golang | 主系统语言，负责 API、Worker、策略、回测、AI 工具调用 |
| 后端框架 | GoFrame / Gin / Chi 三选一 | 推荐 GoFrame 或 Gin，依据团队习惯选择 |
| 前端框架 | React | 构建分析工作台和管理台 |
| UI 组件 | shadcn/ui | 现代化 UI 组件体系，适合 dashboard 和管理系统 |
| 样式 | Tailwind CSS | 与 shadcn/ui 配套 |
| 图表 | ECharts / Recharts | K 线、指标、回测曲线、统计图表 |
| 状态管理 | Zustand / TanStack Query | 服务端状态和前端局部状态管理 |
| 构建工具 | Vite | React 前端构建 |
| 结构化数据库 | PostgreSQL | 股票主数据、财务、板块、策略配置、AI 结果 |
| 时序数据库 | TimescaleDB | 日线、分钟线、盘口快照等时间序列数据 |
| 向量检索 | pgvector | 初期直接集成 PostgreSQL，降低复杂度 |
| 离线分析 | DuckDB | 查询 Parquet，适合批量回测和离线统计 |
| 数据湖格式 | Parquet | 历史归档和训练数据 |
| 缓存 | Redis | 实时快照、任务状态、预警状态、热点查询缓存 |
| 对象存储 | MinIO / S3 | 存储 PDF、公告 HTML、原始 JSON、Parquet 文件 |
| 任务调度 | Go Cron / Asynq / Temporal 可选 | 初期 Go Cron，复杂后可引入 Temporal |
| 消息队列 | Redis Stream / NATS / Kafka 可选 | 初期 Redis Stream，后续按数据量升级 |
| AI 接入 | OpenAI-compatible API / 本地模型可选 | 通过统一 LLM Provider 抽象接入 |
| 日志 | zap / slog | 结构化日志 |
| 配置 | YAML + ENV | 支持本地和容器部署 |
| 部署 | Docker Compose | 初期单机部署 |

### 8.2 后端框架建议

#### 方案 A：GoFrame

适合：

1. 需要较完整工程框架。
2. 偏企业级模块化。
3. 希望集成配置、日志、ORM、路由、校验等能力。

#### 方案 B：Gin + 自选组件

适合：

1. 追求轻量灵活。
2. 团队已有 Gin 经验。
3. 希望自行选择 ORM、配置、日志、任务组件。

#### 方案 C：Chi + 标准库风格

适合：

1. 追求简洁。
2. 关注长期可维护性。
3. 希望尽量贴近 Go 原生生态。

本方案建议：

```text
如果你希望工程化程度高：选 GoFrame。
如果你希望灵活轻量：选 Gin。
```

考虑你已有 GoFrame 经验，QuantSage 推荐优先使用：

```text
GoFrame v2.x
```

### 8.3 前端技术选型

| 模块 | 选型 |
|---|---|
| 框架 | React |
| 构建 | Vite |
| UI | shadcn/ui |
| 样式 | Tailwind CSS |
| 图表 | ECharts 优先，Recharts 辅助 |
| 表格 | TanStack Table |
| 请求 | TanStack Query |
| 状态 | Zustand |
| 表单 | React Hook Form + Zod |
| 路由 | React Router / TanStack Router |
| 主题 | shadcn/ui theme + CSS variables |

### 8.4 分阶段选型

#### V1 单机研究版

```text
GoFrame / Gin
React + shadcn/ui
PostgreSQL + TimescaleDB + pgvector
Redis
MinIO
DuckDB
Parquet
Docker Compose
```

#### V2 盘中辅助版

增加：

```text
QMT / miniQMT
WebSocket
Redis Stream / NATS
实时信号引擎
行情订阅服务
```

#### V3 规模化分析版

增加：

```text
ClickHouse
Kafka / Redpanda
Milvus
Temporal
多实例 Worker
```

---

## 9. 仓库与工程结构

### 9.1 推荐仓库模式

第一阶段推荐单仓库 monorepo：

```text
quantsage/
  apps/
    server/                 # Go 后端服务
    web/                    # React + shadcn/ui 前端
  packages/
    strategy/               # 策略规则定义、DSL、文档
    indicators/             # 指标说明与测试样例
    prompts/                # AI Prompt 模板
  deployments/
    docker-compose/
    k8s/
  configs/
    config.example.yaml
  migrations/
    postgres/
  data/
    samples/
  docs/
    architecture/
    api/
    strategy/
    review/
  research/                 # 可选 Python / Notebook，不进入主链路
    notebooks/
    scripts/
  README.md
  AGENTS.md
```

### 9.2 Go 后端结构

```text
apps/server/
  cmd/
    quantsage-server/
    quantsage-worker/
    quantsage-cli/
  internal/
    app/
    config/
    domain/
      stock/
      marketdata/
      financial/
      indicator/
      strategy/
      backtest/
      document/
      vector/
      ai/
      watchlist/
      alert/
      job/
    infra/
      db/
      redis/
      objectstore/
      llm/
      datasource/
      scheduler/
    interfaces/
      http/
      websocket/
      cli/
    pkg/
      timeseries/
      formula/
      tradingcalendar/
      errors/
  api/
    openapi/
  go.mod
```

### 9.3 React 前端结构

```text
apps/web/
  src/
    app/
    pages/
      dashboard/
      stocks/
      strategies/
      backtests/
      ai-review/
      alerts/
      documents/
      settings/
    components/
      ui/                   # shadcn/ui components
      charts/
      kline/
      stock-card/
      signal-table/
    features/
      stock/
      strategy/
      backtest/
      ai/
      alert/
    lib/
      api.ts
      query.ts
      utils.ts
    hooks/
    stores/
    routes/
  package.json
```

### 9.4 是否拆多仓库

第一阶段不建议拆多仓库。原因：

1. API、前端、策略、数据库迁移迭代频繁。
2. 单人或小团队维护，多仓库会增加版本协调成本。
3. monorepo 更方便统一文档、部署、CI/CD。

后续如果系统规模扩大，可以拆分：

```text
quantsage-server
quantsage-web
quantsage-worker
quantsage-strategy
quantsage-docs
```

---

## 10. 数据源设计

### 10.1 数据源分类

| 数据类型 | 数据源候选 | 说明 |
|---|---|---|
| 股票基础信息 | Tushare / 交易所公开数据 | 股票代码、名称、上市日期、市场、行业 |
| 交易日历 | Tushare / 交易所 | 判断交易日、补数、回测 |
| 日线行情 | Tushare / AKShare / QMT | 开高低收、成交量、成交额 |
| 分钟行情 | AKShare / QMT / 券商行情 | 盘中短线分析 |
| 实时行情 | AKShare / QMT / 券商终端 | 盘中盯盘和预警 |
| 财务数据 | Tushare / 巨潮 / 交易所公告 | 三大报表、财务指标 |
| 公告数据 | 巨潮资讯 / 交易所公告 | 财报、减持、解禁、回购、监管函 |
| 新闻资讯 | 东方财富等公开来源 / 商业源 | 情绪与事件辅助 |
| 板块概念 | AKShare / 东方财富 / 同花顺概念 | 板块强弱与题材分析 |
| Level-2 数据 | 券商授权 / QMT / 专业行情源 | 十档、逐笔、委托队列 |

### 10.2 数据源接入策略

Go 主系统下的数据源接入分为三种方式：

#### 方式一：HTTP API 直接接入

适合：

```text
Tushare HTTP API
交易所公开接口
自建代理接口
商业数据源 REST API
```

优点：

1. Go 原生支持好。
2. 部署简单。
3. 不依赖 Python。
4. 易于限流、重试和日志追踪。

#### 方式二：HTML / JSON 公开页面解析

适合：

```text
公告列表
新闻列表
部分板块信息
```

注意：

1. 需要遵守 robots、访问频率和数据使用边界。
2. 不建议作为核心生产数据源。
3. 必须保留 source 和 fetched_at。

#### 方式三：Python 辅助桥接，可选

适合：

```text
Tushare SDK 原型验证
AKShare 快速取数
研究阶段脚本
```

该方式不进入主系统必需链路。若使用，可将结果写入 raw 文件或通过内部 HTTP 服务转发给 Go 系统。

### 10.3 数据源优先级

第一阶段建议：

```text
Tushare：主数据、日线、复权、财务
公开公告源：公告与财报文本
Go 直接 HTTP 接口：优先
Python 脚本：仅用于验证和补充
```

第二阶段增加：

```text
QMT / miniQMT：实时行情、分钟线、分笔数据
```

第三阶段根据成本增加：

```text
Level-2 授权行情源
商业新闻源
机构级数据终端
```

---

## 11. 数据模型设计

### 11.1 数据域划分

```text
stock_master      股票主数据域
market_data       行情数据域
financial_data    财务数据域
sector_data       行业板块域
text_data         公告新闻文本域
feature_data      特征指标域
signal_data       策略信号域
backtest_data     回测结果域
ai_analysis       AI 分析结果域
user_data         用户私有业务数据域
```

V2 从数据隔离边界上明确分成两层：

1. **共享底座数据**
   - `stock_basic`
   - `trade_calendar`
   - `stock_daily`
   - `adj_factor`
   - `financial_indicator`
   - `stock_factor_daily`
   - `strategy_signal`
   - `announcement`
2. **用户私有数据**
   - `app_user`
   - `watchlist_group`
   - `watchlist_item`
   - `user_position`
   - 后续用户级 AI 复盘与预警规则

也就是说，股票、行情、指标和信号按全站统一口径计算一次；用户只隔离“我关注哪些股票”和“我的持仓是什么”。

### 11.2 核心表清单

| 表名 | 类型 | 说明 |
|---|---|---|
| stock_basic | 主数据 | 股票基础信息 |
| trade_calendar | 主数据 | 交易日历 |
| stock_daily | 时序 | 日线行情 |
| stock_minute | 时序 | 分钟行情 |
| adj_factor | 时序 | 复权因子 |
| stock_realtime_snapshot | 时序 | 实时行情快照 |
| financial_indicator | 结构化 | 财务指标 |
| income_statement | 结构化 | 利润表 |
| balance_sheet | 结构化 | 资产负债表 |
| cashflow_statement | 结构化 | 现金流量表 |
| sector | 主数据 | 行业/板块 |
| stock_sector_mapping | 关系 | 个股与板块关系 |
| announcement | 文本 | 公告元数据 |
| document_chunk | 文本 | 公告/财报/新闻切片 |
| document_embedding | 向量 | 文本向量 |
| stock_factor_daily | 衍生 | 日频技术指标 |
| stock_event_tag | 衍生 | 事件标签 |
| strategy_definition | 配置 | 策略规则定义 |
| strategy_signal | 衍生 | 策略信号结果 |
| backtest_run | 结果 | 回测任务 |
| backtest_result | 结果 | 回测统计 |
| ai_stock_analysis | 结果 | AI 个股分析 |
| app_user | 业务 | 预置账号 |
| watchlist_group | 业务 | 用户自选分组 |
| watchlist_item | 业务 | 分组内自选股 |
| user_position | 业务 | 用户持仓记录 |
| alert_rule | 配置 | 预警规则 |
| alert_event | 结果 | 预警事件 |
| job_run_log | 运维 | 任务执行记录 |

### 11.3 核心表结构示例

#### 11.3.1 股票基础表

```sql
CREATE TABLE stock_basic (
    ts_code        TEXT PRIMARY KEY,
    symbol         TEXT NOT NULL,
    name           TEXT NOT NULL,
    area           TEXT,
    industry       TEXT,
    market         TEXT,
    exchange       TEXT,
    list_date      DATE,
    delist_date    DATE,
    is_active      BOOLEAN DEFAULT TRUE,
    source         TEXT,
    created_at     TIMESTAMPTZ DEFAULT now(),
    updated_at     TIMESTAMPTZ DEFAULT now()
);
```

#### 11.3.2 交易日历表

```sql
CREATE TABLE trade_calendar (
    exchange       TEXT NOT NULL,
    cal_date       DATE NOT NULL,
    is_open        BOOLEAN NOT NULL,
    pretrade_date  DATE,
    created_at     TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (exchange, cal_date)
);
```

#### 11.3.3 日线行情表

```sql
CREATE TABLE stock_daily (
    ts_code      TEXT NOT NULL,
    trade_date   DATE NOT NULL,
    open         NUMERIC(18,4),
    high         NUMERIC(18,4),
    low          NUMERIC(18,4),
    close        NUMERIC(18,4),
    pre_close    NUMERIC(18,4),
    change       NUMERIC(18,4),
    pct_chg      NUMERIC(10,4),
    vol          NUMERIC(20,4),
    amount       NUMERIC(20,4),
    source       TEXT NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT now(),
    updated_at   TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (ts_code, trade_date)
);
```

#### 11.3.4 分钟行情表

```sql
CREATE TABLE stock_minute (
    ts_code      TEXT NOT NULL,
    ts           TIMESTAMPTZ NOT NULL,
    period       TEXT NOT NULL,
    open         NUMERIC(18,4),
    high         NUMERIC(18,4),
    low          NUMERIC(18,4),
    close        NUMERIC(18,4),
    volume       NUMERIC(20,4),
    amount       NUMERIC(20,4),
    source       TEXT NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (ts_code, ts, period)
);
```

#### 11.3.5 日频技术因子表

```sql
CREATE TABLE stock_factor_daily (
    ts_code       TEXT NOT NULL,
    trade_date    DATE NOT NULL,

    ma5           NUMERIC(18,4),
    ma10          NUMERIC(18,4),
    ma20          NUMERIC(18,4),
    ma60          NUMERIC(18,4),

    ema12         NUMERIC(18,4),
    ema26         NUMERIC(18,4),
    macd_dif      NUMERIC(18,6),
    macd_dea      NUMERIC(18,6),
    macd_hist     NUMERIC(18,6),

    rsi6          NUMERIC(10,4),
    rsi12         NUMERIC(10,4),
    rsi24         NUMERIC(10,4),

    volume_ma5    NUMERIC(20,4),
    volume_ma20   NUMERIC(20,4),
    volume_ratio  NUMERIC(10,4),

    turnover_rate NUMERIC(10,4),
    amplitude     NUMERIC(10,4),
    upper_shadow_ratio NUMERIC(10,4),
    lower_shadow_ratio NUMERIC(10,4),

    close_above_ma5   BOOLEAN,
    close_above_ma10  BOOLEAN,
    close_above_ma20  BOOLEAN,
    ma_bullish        BOOLEAN,
    volume_breakout   BOOLEAN,
    price_breakout_20 BOOLEAN,

    factor_version TEXT NOT NULL DEFAULT 'v1',
    created_at    TIMESTAMPTZ DEFAULT now(),
    updated_at    TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (ts_code, trade_date, factor_version)
);
```

#### 11.3.6 事件标签表

```sql
CREATE TABLE stock_event_tag (
    id           BIGSERIAL PRIMARY KEY,
    ts_code      TEXT NOT NULL,
    trade_date   DATE NOT NULL,
    event_type   TEXT NOT NULL,
    event_level  TEXT,
    score        NUMERIC(10,4),
    description  TEXT,
    meta         JSONB,
    version      TEXT NOT NULL DEFAULT 'v1',
    created_at   TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_stock_event_tag_code_date
ON stock_event_tag(ts_code, trade_date);

CREATE INDEX idx_stock_event_tag_type_date
ON stock_event_tag(event_type, trade_date);
```

#### 11.3.7 策略定义表

```sql
CREATE TABLE strategy_definition (
    id              BIGSERIAL PRIMARY KEY,
    strategy_code   TEXT NOT NULL UNIQUE,
    strategy_name   TEXT NOT NULL,
    strategy_type   TEXT NOT NULL,
    description     TEXT,
    config          JSONB NOT NULL,
    enabled         BOOLEAN DEFAULT TRUE,
    version         TEXT NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);
```

#### 11.3.8 策略信号表

```sql
CREATE TABLE strategy_signal (
    id               BIGSERIAL PRIMARY KEY,
    strategy_code    TEXT NOT NULL,
    strategy_version TEXT NOT NULL,
    ts_code          TEXT NOT NULL,
    trade_date       DATE NOT NULL,
    signal_type      TEXT NOT NULL,
    signal_strength  NUMERIC(10,4),
    buy_price_ref    NUMERIC(18,4),
    stop_loss_ref    NUMERIC(18,4),
    take_profit_ref  NUMERIC(18,4),
    invalidation_condition TEXT,
    reason           TEXT,
    meta             JSONB,
    created_at       TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_strategy_signal_date
ON strategy_signal(trade_date, strategy_code, signal_type);
```

#### 11.3.9 用户与自选股结构

V2 不引入团队、组织和复杂 RBAC，只支持最简用户隔离。

```sql
CREATE TABLE app_user (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    role TEXT NOT NULL DEFAULT 'user',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE watchlist_group (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);

CREATE TABLE watchlist_item (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES watchlist_group(id) ON DELETE CASCADE,
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (group_id, ts_code)
);

CREATE TABLE user_position (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    position_date DATE NOT NULL,
    quantity NUMERIC(20,4) NOT NULL,
    cost_price NUMERIC(18,4) NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

当前项目仍处于开发阶段，V2 直接修改现有 schema，不额外新增迁移脚本；旧的扁平 `watchlist` / `position` 结构不做兼容双写。

---

## 12. 数据分层设计

### 12.1 数据分层

```text
raw       原始数据层
cleaned   清洗标准化层
feature   特征指标层
signal    策略信号层
analysis  AI 分析结果层
mart      应用数据集市层
```

### 12.2 分层说明

| 层级 | 说明 | 是否可重算 |
|---|---|---|
| raw | 外部接口原始返回、PDF、HTML、JSON | 不重算，只追加或覆盖 |
| cleaned | 标准化后的行情、财务、公告元数据 | 可重跑 |
| feature | 均线、MACD、RSI、量比、形态标签 | 可重算 |
| signal | 买卖点候选信号、风险信号 | 可重算 |
| analysis | AI 分析结论、复盘报告、归因结果 | 可重算并保留版本 |
| mart | 面向前端展示的聚合结果 | 可重算 |

### 12.3 数据湖目录

```text
data_lake/
  raw/
    tushare/
    akshare/
    qmt/
    announcements/
    news/
  cleaned/
    stock_daily/
      year=2026/month=04/
    stock_minute/
      period=1m/year=2026/month=04/
    financial/
    sector/
  feature/
    daily_factors/
    event_tags/
  signal/
    strategy_signals/
  analysis/
    ai_reports/
  training/
    short_term_signal_dataset/
```

---

## 13. 指标与买卖点信号设计

### 13.1 设计原则

买卖点信号由规则引擎计算，AI 不直接生成最终买卖点。

AI 的作用是：

1. 解释信号。
2. 过滤低质量信号。
3. 识别风险因素。
4. 生成交易计划。
5. 做相似案例统计。

### 13.2 原子指标

| 指标 | 说明 |
|---|---|
| MA | 简单移动均线，如 MA5、MA10、MA20、MA60 |
| EMA | 指数移动均线，用于 MACD |
| MACD | 趋势动能指标 |
| RSI | 超买超卖指标 |
| BOLL | 波动区间指标 |
| ATR | 波动率指标 |
| volume_ma | 成交量均线 |
| volume_ratio | 当前成交量 / 均量 |
| turnover_rate | 换手率 |
| amplitude | 振幅 |
| upper_shadow_ratio | 上影线比例 |
| lower_shadow_ratio | 下影线比例 |
| gap_pct | 跳空幅度 |

### 13.3 结构信号

| 信号 | 说明 |
|---|---|
| price_breakout_20 | 突破 20 日新高 |
| price_breakdown_20 | 跌破 20 日低点 |
| volume_breakout | 放量 |
| volume_shrink | 缩量 |
| ma_bullish | 均线多头排列 |
| ma_bearish | 均线空头排列 |
| long_upper_shadow | 长上影线 |
| long_lower_shadow | 长下影线 |
| trend_break | 趋势破位 |
| gap_up | 跳空高开 |
| gap_down | 跳空低开 |
| volume_stall | 放量滞涨 |
| pullback_to_ma10 | 回踩 10 日线 |
| pullback_to_ma20 | 回踩 20 日线 |

### 13.4 策略信号

#### 13.4.1 放量突破买点

触发条件：

```text
close_today > highest(high, 20, exclude_today=true)
AND volume_today > sma(volume, 20) * 1.8
AND pct_chg_today > 3
AND close_today > ma20
AND upper_shadow_ratio < 0.25
```

适用场景：

1. 平台整理后向上突破。
2. 板块同步走强。
3. 个股成交额明显放大。

失效条件：

```text
close < breakout_price
OR close < ma10
OR 放量长上影后次日低开低走
```

#### 13.4.2 强势股缩量回踩买点

触发条件：

```text
close > ma20
AND ma5 > ma10
AND ma10 > ma20
AND low <= ma10 * 1.01
AND close >= ma10
AND volume < sma(volume, 20) * 0.8
```

确认条件：

```text
next_close > today_high
AND next_volume > today_volume
```

适用场景：

1. 上升趋势未破。
2. 股价回踩短期均线。
3. 回调过程中抛压较轻。

失效条件：

```text
close < ma20
OR close < recent_swing_low
```

#### 13.4.3 超跌反弹买点

触发条件：

```text
close < ma20
AND rsi6 < 25
AND pct_chg_5d < -12
AND lower_shadow_ratio > 0.5
```

确认条件：

```text
next_close > today_high
AND next_pct_chg > 3
```

风险说明：

超跌反弹属于逆势模式，容易接飞刀，应降低仓位和持有周期。

#### 13.4.4 放量滞涨卖点

触发条件：

```text
volume > sma(volume, 20) * 2
AND pct_chg < 2
AND upper_shadow_ratio > 0.4
```

含义：

成交明显放大，但股价未能有效上涨，说明上方抛压较重，短线分歧放大。

#### 13.4.5 趋势破位卖点

触发条件：

```text
close < ma20
AND volume > sma(volume, 20) * 1.2
```

或更激进：

```text
close < ma10
```

### 13.5 信号评分模型

#### 13.5.1 放量突破评分

| 条件 | 分数 |
|---|---:|
| 收盘价突破 20 日新高 | +20 |
| 成交量大于 20 日均量 1.8 倍 | +20 |
| 收盘价在 5/10/20 日线上方 | +15 |
| 5日线 > 10日线 > 20日线 | +15 |
| 所属板块当日涨幅前 20% | +15 |
| 换手率 3%–15% | +10 |
| 上影线小于 25% | +10 |
| 前 20 日涨幅大于 40% | -15 |
| 高位放量长上影 | -30 |
| 财报、减持、解禁风险 | -30 |

#### 13.5.2 信号等级

| 分数 | 等级 | 操作建议 |
|---:|---|---|
| >= 80 | A | 高质量候选，可重点观察 |
| 60–79 | B | 普通候选，需结合板块和位置 |
| 40–59 | C | 弱候选，仅观察 |
| < 40 | D | 过滤 |

---

## 14. 策略规则 DSL 设计

### 14.1 设计目标

策略不直接硬编码在业务逻辑中，而是通过配置化规则定义。

目标：

1. 策略可配置。
2. 参数可调整。
3. 版本可管理。
4. 可用于信号生成。
5. 可用于回测。
6. 可由 AI 解释。

### 14.2 JSON 配置示例

```json
{
  "strategy_code": "volume_breakout_v1",
  "strategy_name": "放量突破20日新高",
  "strategy_type": "buy_signal",
  "version": "v1",
  "conditions": {
    "all": [
      { "left": "close", "op": ">", "right": "highest(high, 20, exclude_today=true)" },
      { "left": "volume", "op": ">", "right": "sma(volume, 20) * 1.8" },
      { "left": "pct_chg", "op": ">", "right": 3 },
      { "left": "close", "op": ">", "right": "ma20" },
      { "left": "upper_shadow_ratio", "op": "<", "right": 0.25 }
    ]
  },
  "risk_control": {
    "stop_loss": "min(breakout_price, ma10)",
    "max_holding_days": 10,
    "take_profit_pct": 12
  }
}
```

### 14.3 Go 规则执行模块

核心接口：

```go
type StrategyEngine interface {
    Evaluate(ctx context.Context, strategy StrategyDefinition, input MarketContext) (SignalResult, error)
}

type IndicatorEngine interface {
    CalculateDaily(ctx context.Context, tsCode string, tradeDate time.Time) (*DailyFactor, error)
}

type BacktestEngine interface {
    Run(ctx context.Context, req BacktestRequest) (*BacktestReport, error)
}
```

### 14.4 规则执行要求

1. 禁止未来函数。
2. 所有指标必须声明数据窗口。
3. 所有策略必须带 version。
4. 所有结果必须记录输入参数快照。
5. 回测与盘后信号使用同一套规则引擎。

---

## 15. 回测系统设计

### 15.1 回测目标

1. 验证策略信号是否具备统计优势。
2. 评估不同市场环境下的有效性。
3. 评估不同板块、市值、波动率条件下的差异。
4. 输出交易成本和滑点后的真实表现。

### 15.2 回测输入

```text
策略定义
股票池
回测时间区间
买入规则
卖出规则
持有周期
交易成本
滑点模型
仓位模型
复权方式
```

### 15.3 回测输出指标

| 指标 | 说明 |
|---|---|
| signal_count | 信号总数 |
| win_rate | 胜率 |
| avg_return | 平均收益 |
| median_return | 中位收益 |
| profit_loss_ratio | 盈亏比 |
| max_drawdown | 最大回撤 |
| max_consecutive_loss | 最大连续亏损 |
| avg_holding_days | 平均持有天数 |
| annualized_return | 年化收益 |
| sharpe_ratio | 夏普比率 |
| turnover | 换手率 |
| cost_adjusted_return | 扣费后收益 |

### 15.4 回测分层统计

| 维度 | 说明 |
|---|---|
| 市场环境 | 牛市、震荡、熊市 |
| 板块 | 行业、概念、题材 |
| 市值 | 大盘、中盘、小盘 |
| 成交额 | 高流动性、低流动性 |
| 波动率 | 高波动、低波动 |
| 信号分数 | A/B/C/D 分层 |
| 持有周期 | 1日、3日、5日、10日、20日 |

### 15.5 回测结果表

```sql
CREATE TABLE backtest_run (
    id               BIGSERIAL PRIMARY KEY,
    strategy_code    TEXT NOT NULL,
    strategy_version TEXT NOT NULL,
    start_date       DATE NOT NULL,
    end_date         DATE NOT NULL,
    stock_pool       TEXT,
    config           JSONB NOT NULL,
    status           TEXT NOT NULL,
    created_at       TIMESTAMPTZ DEFAULT now(),
    finished_at      TIMESTAMPTZ
);

CREATE TABLE backtest_result (
    id              BIGSERIAL PRIMARY KEY,
    run_id          BIGINT NOT NULL,
    metric_name     TEXT NOT NULL,
    metric_value    NUMERIC(20,6),
    dimension       JSONB,
    created_at      TIMESTAMPTZ DEFAULT now()
);
```

---

## 16. AI 分析层设计

### 16.1 AI 层定位

AI 不直接负责生成买卖点，而负责：

1. 解释规则信号。
2. 分析财报、公告、新闻。
3. 检索相似历史案例。
4. 归因个股异动。
5. 生成交易计划和风险清单。
6. 生成每日持仓复盘。

### 16.2 AI 工具调用架构

```text
用户问题
  ↓
意图识别
  ↓
工具路由
  ├─ SQL 查询工具
  ├─ 行情指标工具
  ├─ 财务数据工具
  ├─ 文本检索工具
  ├─ 相似案例工具
  ├─ 回测工具
  └─ 风险事件工具
  ↓
结构化结果
  ↓
AI 分析与解释
  ↓
输出报告
```

### 16.3 工具接口设计

```text
get_stock_daily(ts_code, start_date, end_date)
get_stock_factors(ts_code, trade_date)
get_financial_indicator(ts_code, period)
get_sector_strength(trade_date)
get_strategy_signals(ts_code, start_date, end_date)
get_similar_cases(condition)
get_announcements(ts_code, start_date, end_date)
search_documents(query, filters)
run_backtest(strategy_code, config)
compare_stocks(stock_list, metrics)
```

### 16.4 RAG 检索流程

```text
公告/财报/新闻原文
  ↓
文本抽取
  ↓
清洗
  ↓
切片
  ↓
Embedding
  ↓
pgvector 存储
  ↓
语义检索 + 元数据过滤
  ↓
AI 摘要与引用
```

### 16.5 AI 输出约束

AI 输出必须遵守以下格式：

```text
1. 当前结论
2. 数据依据
3. 技术面分析
4. 基本面分析
5. 板块与市场环境
6. 风险因素
7. 买入条件
8. 卖出条件
9. 失效条件
10. 不确定性说明
```

### 16.6 防幻觉机制

| 风险 | 控制方式 |
|---|---|
| AI 编造数据 | 所有数值必须来自工具返回 |
| AI 错读行情 | 使用结构化字段，不让 AI 手算关键数据 |
| AI 直接荐股 | 输出条件和风险，不输出确定性承诺 |
| AI 忽略回测 | 策略信号必须附带历史统计 |
| AI 混用时间 | 所有分析带交易日期和数据更新时间 |
| AI 误读财报 | 财报摘要附来源文档片段 |

---

## 17. 核心业务流程

### 17.1 日线数据同步流程

```text
交易日 15:30 后
  ↓
拉取当日行情
  ↓
写入 raw 数据
  ↓
清洗与标准化
  ↓
更新 stock_daily
  ↓
更新复权因子
  ↓
计算技术指标
  ↓
生成事件标签
  ↓
生成策略信号
  ↓
触发 AI 复盘
```

### 17.2 分钟行情同步流程

```text
盘中行情源
  ↓
订阅/轮询实时行情
  ↓
写入 Redis 快照
  ↓
聚合 1m K线
  ↓
写入 TimescaleDB
  ↓
实时计算预警规则
  ↓
推送 WebSocket / 通知
```

### 17.3 公告财报处理流程

```text
公告列表同步
  ↓
下载 PDF / HTML
  ↓
对象存储落盘
  ↓
文本抽取
  ↓
公告类型识别
  ↓
切片与向量化
  ↓
生成事件标签
  ↓
AI 摘要
```

### 17.4 每日持仓复盘流程

```text
读取持仓列表
  ↓
查询当日行情和技术指标
  ↓
查询策略信号和风险事件
  ↓
查询板块表现
  ↓
查询公告新闻
  ↓
AI 生成复盘
  ↓
保存 ai_stock_analysis
  ↓
前端展示 / 通知
```

### 17.5 相似案例分析流程

```text
输入当前股票状态
  ↓
抽取条件：
  - 技术形态
  - 财报表现
  - 板块状态
  - 涨跌幅
  - 量能
  - 事件标签
  ↓
历史数据库检索
  ↓
统计后续 1/3/5/10/20 日收益
  ↓
输出概率分布与风险提示
```

---

## 18. API 设计

### 18.1 股票基础 API

```http
GET /api/stocks
GET /api/stocks/{ts_code}
GET /api/stocks/{ts_code}/daily
GET /api/stocks/{ts_code}/factors
GET /api/stocks/{ts_code}/financials
GET /api/stocks/{ts_code}/announcements
```

### 18.2 策略信号 API

```http
GET /api/signals
GET /api/signals/{ts_code}
POST /api/strategies
PUT /api/strategies/{strategy_code}
POST /api/strategies/{strategy_code}/run
POST /api/strategies/{strategy_code}/backtest
```

### 18.3 认证与会话 API

```http
POST /api/auth/login
POST /api/auth/logout
GET  /api/auth/me
```

认证规则：

1. V2 继续沿用 `gin-contrib/sessions + Redis`，不切换到 JWT。
2. 账号来源为管理员预置账户，不开放注册。
3. Session 中只保存最小必要的 `user_id` 和登录态元数据。
4. 未登录请求只能访问 `/api/auth/login`、`/api/healthz` 等公开接口。

### 18.4 AI 分析 API

```http
POST /api/ai/analyze-stock
POST /api/ai/explain-signal
POST /api/ai/similar-cases
POST /api/ai/daily-review
POST /api/ai/compare-stocks
POST /api/ai/ask
```

### 18.5 自选股、持仓与预警 API

```http
GET /api/watchlists
POST /api/watchlists
PUT /api/watchlists/{id}
DELETE /api/watchlists/{id}

GET /api/watchlists/{id}/items
POST /api/watchlists/{id}/items
DELETE /api/watchlists/{id}/items/{item_id}

GET /api/positions
POST /api/positions
PUT /api/positions/{id}
DELETE /api/positions/{id}

GET /api/alerts
POST /api/alerts/rules
PUT /api/alerts/rules/{id}
GET /api/alerts/events
```

这里的用户隔离原则是：

1. 前端和调用方不传 `user_id`。
2. `watchlists` / `positions` 全部通过 session 中的当前用户自动过滤。
3. 共享行情接口如 `GET /api/stocks`、`GET /api/signals` 保持全站共享，不做用户分片。

---

## 19. 前端功能设计

### 19.1 功能模块

| 模块 | 功能 |
|---|---|
| 登录与会话 | 预置账号登录、会话保持、登出、当前用户信息 |
| 首页驾驶舱 | 市场概览、板块热度、自选股异动、持仓风险 |
| 个股详情 | K线、指标、财务、公告、AI 分析 |
| 策略中心 | 策略配置、信号列表、回测结果 |
| 我的自选 | 自选分组管理、分组内股票维护、备注 |
| 我的持仓 | 持仓录入、持仓列表、成本价和持仓日期维护 |
| AI 复盘 | 每日持仓复盘、自选股分析、风险提示 |
| 相似案例 | 当前形态的历史案例与统计 |
| 公告财报 | 公告检索、财报摘要、风险事件 |
| 预警中心 | 放量突破、跌破均线、财报风险、板块异动 |
| 数据任务 | 同步状态、失败任务、补数任务 |
| 系统设置 | 数据源、LLM Provider、策略参数、通知方式 |

### 19.2 页面风格

使用 React + shadcn/ui 构建专业、克制、信息密度较高的分析工作台。

设计原则：

1. 深色/浅色主题均支持。
2. 数据卡片清晰展示关键指标。
3. 表格支持筛选、排序、列配置。
4. K 线、成交量、指标图联动。
5. AI 分析区与数据依据分离展示。
6. 所有信号必须可点击查看触发条件。

### 19.3 个股分析页面

页面结构：

```text
顶部：股票名称、代码、当前价格、涨跌幅、成交额、换手率
左侧：K线 + 均线 + 成交量
右侧：AI 分析摘要
中部：技术指标、策略信号、事件标签
下部：财务指标、公告新闻、相似案例
```

### 19.4 策略信号页面

字段：

```text
日期
股票代码
股票名称
策略名称
信号类型
信号强度
买入参考价
止损参考价
失效条件
AI 解释
历史胜率
```

### 19.5 V2 登录与用户隔离约束

V2 前端页面至少包括：

```text
#/login
#/stocks
#/stocks/:id
#/signals
#/watchlists
#/positions
#/jobs
```

约束：

1. `stocks` / `signals` / `jobs` 继续展示共享底座数据。
2. `watchlists` / `positions` 只展示当前登录用户数据。
3. 登录成功后缓存 `me`，登出时清空 `me`、`watchlists`、`positions` 等私有缓存。
4. 路由守卫必须阻止未登录用户访问私有页面。

---

## 20. 部署架构

### 20.1 单机 Docker Compose 架构

```text
┌──────────────────────────────┐
│           Nginx              │
└──────────────┬───────────────┘
               │
┌──────────────▼───────────────┐
│        React Web             │
└──────────────┬───────────────┘
               │
┌──────────────▼───────────────┐
│        Go API Service        │
└───────┬───────────┬──────────┘
        │           │
┌───────▼──────┐ ┌──▼──────────┐
│ PostgreSQL   │ │ Redis       │
│ TimescaleDB  │ │             │
│ pgvector     │ │             │
└───────┬──────┘ └─────────────┘
        │
┌───────▼──────┐
│ MinIO        │
└──────────────┘

┌──────────────────────────────┐
│ Go Worker Service            │
│ - 数据同步                   │
│ - 指标计算                   │
│ - 信号生成                   │
│ - AI 复盘                    │
└──────────────────────────────┘
```

### 20.2 Docker Compose 示例

```yaml
version: "3.9"

services:
  postgres:
    image: timescale/timescaledb:latest-pg16
    container_name: quantsage-postgres
    environment:
      POSTGRES_DB: quantsage
      POSTGRES_USER: quantsage
      POSTGRES_PASSWORD: quantsage_password
    ports:
      - "5432:5432"
    volumes:
      - ./data/postgres:/var/lib/postgresql/data

  redis:
    image: redis:7
    container_name: quantsage-redis
    ports:
      - "6379:6379"

  minio:
    image: minio/minio
    container_name: quantsage-minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minio
      MINIO_ROOT_PASSWORD: minio_password
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - ./data/minio:/data

  server:
    build: ./apps/server
    container_name: quantsage-server
    depends_on:
      - postgres
      - redis
      - minio
    ports:
      - "8080:8080"
    env_file:
      - .env

  worker:
    build: ./apps/server
    container_name: quantsage-worker
    command: ["/app/quantsage-worker"]
    depends_on:
      - postgres
      - redis
      - minio
    env_file:
      - .env

  web:
    build: ./apps/web
    container_name: quantsage-web
    ports:
      - "3000:80"
```

---

## 21. 任务调度设计

### 21.1 任务类型

| 任务 | 触发时间 | 说明 |
|---|---|---|
| sync_stock_basic | 每日 08:00 | 更新股票基础数据 |
| sync_trade_calendar | 每月一次 | 更新交易日历 |
| sync_daily_market | 交易日 15:30 后 | 同步日线行情 |
| sync_adj_factor | 交易日 16:00 后 | 同步复权因子 |
| calc_daily_factor | 日线同步后 | 计算技术因子 |
| generate_event_tags | 因子计算后 | 生成事件标签 |
| generate_strategy_signals | 标签生成后 | 生成策略信号 |
| sync_announcements | 每日 20:00 / 23:00 | 同步公告 |
| parse_documents | 公告下载后 | 解析公告财报文本 |
| embed_documents | 文本解析后 | 生成向量 |
| generate_daily_review | 每日 21:00 后 | 生成 AI 复盘 |
| backfill_missing_data | 手动触发 | 补历史数据 |

### 21.2 任务状态表

```sql
CREATE TABLE job_run_log (
    id            BIGSERIAL PRIMARY KEY,
    job_name      TEXT NOT NULL,
    biz_date      DATE,
    status        TEXT NOT NULL,
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    error_message TEXT,
    retry_count   INT DEFAULT 0,
    meta          JSONB,
    created_at    TIMESTAMPTZ DEFAULT now()
);
```

### 21.3 Go Worker 要求

1. 所有任务必须幂等。
2. 所有任务必须记录 job_run_log。
3. 支持按日期补数。
4. 支持按股票重算指标。
5. 支持按策略版本重算信号。
6. 失败任务可重试。
7. 长任务需记录进度。

---

## 22. 数据质量设计

### 22.1 数据校验规则

| 校验项 | 规则 |
|---|---|
| 交易日校验 | 非交易日不应有普通日线数据 |
| 价格校验 | high >= open/close/low，low <= open/close/high |
| 成交量校验 | 成交量和成交额不得为负 |
| 涨跌幅校验 | pct_chg 与 close/pre_close 差异需在容忍范围内 |
| 复权校验 | 复权因子缺失时不计算复权行情 |
| 重复校验 | ts_code + trade_date 不允许重复 |
| 缺失校验 | 当日全市场数据数量异常需告警 |
| 财报校验 | 同一报告期不得重复覆盖未确认数据 |

### 22.2 异常处理

| 异常 | 处理方式 |
|---|---|
| 数据源不可用 | 重试 + 降级数据源 |
| 部分股票缺失 | 记录缺失列表，进入补数任务 |
| 数据明显异常 | 标记 dirty，不进入 feature 层 |
| 公告解析失败 | 保留原文件，等待人工或异步重试 |
| AI 调用失败 | 记录失败状态，可重跑 |

---

## 23. 安全与合规设计

### 23.1 数据合规

1. 明确数据源授权边界。
2. 区分个人研究使用和商业服务使用。
3. 不对外分发未经授权的实时行情数据。
4. 不向第三方公开原始数据源返回内容。
5. 公告、财报等公开信息应保留来源和抓取时间。

### 23.2 投资合规

系统输出应定位为“辅助分析”，不应表述为确定性投资建议。

AI 输出应避免：

```text
一定上涨
必须买入
保证盈利
无风险
内幕消息
```

推荐输出形式：

```text
当前信号满足哪些条件
买入逻辑成立条件
止损条件
失效条件
历史相似案例统计
主要风险
```

### 23.3 系统安全

| 项目 | 方案 |
|---|---|
| API 鉴权 | V1/V2 优先 Session；JWT 不进入主路径 |
| 敏感配置 | .env / Secret 管理 |
| 数据库权限 | 最小权限原则 |
| AI 工具调用 | 参数校验 + SQL 模板 |
| 日志 | 避免记录 Token、密钥、账户信息 |
| 备份 | PostgreSQL 定期备份 + MinIO 文件备份 |

V2 额外增加 4 条用户隔离要求：

1. 所有用户私有查询必须显式带 `user_id` 条件，禁止仅靠前端隐藏。
2. `watchlist_group`、`watchlist_item`、`user_position` 的写操作必须校验资源归属。
3. Redis session 必须使用 `HttpOnly` Cookie；生产环境开启 `Secure`。
4. 管理员预置账号只通过配置或启动 bootstrap 写入，不开放公网注册入口。

---

## 24. 可观测性与运维

### 24.1 日志

日志分为：

```text
api.log
worker.log
data_sync.log
factor_calc.log
strategy_signal.log
ai_analysis.log
backtest.log
```

### 24.2 监控指标

| 指标 | 说明 |
|---|---|
| 数据同步成功率 | 每日任务成功比例 |
| 数据缺失数量 | 缺失股票/日期数量 |
| API 响应时间 | P50/P95/P99 |
| AI 调用成功率 | LLM 调用成功比例 |
| 回测任务耗时 | 策略回测性能 |
| 数据库容量 | PostgreSQL / MinIO 增长 |
| Worker 队列积压 | 异步任务积压情况 |

### 24.3 备份策略

| 数据 | 策略 |
|---|---|
| PostgreSQL | 每日全量 + WAL 归档 |
| MinIO 文件 | 每周快照或对象同步 |
| Parquet 数据湖 | 可重算数据可低频备份，raw 数据优先备份 |
| 策略配置 | 纳入数据库备份与 Git 版本管理 |
| AI 分析结果 | 可重算，但建议保留版本 |

---

## 25. 性能与容量评估

### 25.1 数据量粗估

| 数据类型 | 粒度 | 估算 |
|---|---|---|
| A 股日线 | 约 5000+ 股票 × 10 年 | 百万级记录 |
| 分钟线 | 5000+ 股票 × 每日 240 根 × 多年 | 亿级记录 |
| Tick / Level-2 | 全市场逐笔 | 十亿级以上 |
| 财务指标 | 股票 × 报告期 | 百万级以内 |
| 公告文本 | 多年公告 | 十万到百万文档级 |
| 技术因子 | 股票 × 交易日 × 指标 | 百万到千万级 |

### 25.2 存储策略

| 阶段 | 策略 |
|---|---|
| V1 | 全市场日线 + 自选股分钟线，PostgreSQL/TimescaleDB 足够 |
| V2 | 增加用户登录、自选股和持仓隔离；行情底座仍共享，TimescaleDB 继续按共享时序数据优化 |
| V3 | 全市场 tick/Level-2，增加 ClickHouse 和对象存储归档 |

### 25.3 查询优化

1. 高频查询建立复合索引。
2. 时序数据按时间分区。
3. 历史归档写 Parquet。
4. 前端常用聚合结果写 mart 表。
5. AI 查询走工具接口，不直接扫大表。
6. 相似案例提前生成事件标签和特征，避免临时复杂计算。

---

## 26. 里程碑规划

### 26.1 第一阶段：工程骨架与数据底座

周期：1–2 周

目标：完成基础工程结构、核心数据同步和查询。

交付：

1. quantsage monorepo 初始化。
2. Go 后端工程骨架。
3. React + shadcn/ui 前端工程骨架。
4. PostgreSQL + TimescaleDB + pgvector 环境。
5. 股票基础信息表。
6. 交易日历表。
7. 日线行情表。
8. 复权因子表。
9. 财务指标表。
10. 数据同步任务。
11. 基础 API。

验收标准：

```text
可以查询任意股票过去 N 年日线和基础财务指标。
每日收盘后可自动同步当日行情。
失败任务可查看、可重试。
前端可展示股票基础信息和日线数据。
```

### 26.2 第二阶段：最简用户版与数据隔离

周期：1–2 周

目标：在共享行情底座上支持管理员预置账户、登录态和用户级自选股/持仓隔离。

交付：

1. `app_user`、`watchlist_group`、`watchlist_item`、`user_position` 表结构。
2. 基于 Redis session 的登录 / 登出 / 当前用户接口。
3. 预置账号 bootstrap 能力。
4. 自选分组与分组内股票 CRUD。
5. 用户持仓 CRUD。
6. 前端登录页、路由守卫、自选股页、持仓页。
7. 用户私有缓存失效和登出清理机制。

验收标准：

```text
两个不同账号登录后，只能看到自己的自选股和持仓。
股票、日线、指标和信号数据仍为全站共享，不重复存储。
未登录用户不能访问用户私有页面和接口。
不新增新的迁移脚本，直接修改开发期 schema 即可完成交付。
```

### 26.3 第三阶段：指标与信号

周期：1–2 周

目标：完成技术指标和短线信号计算。

交付：

1. MA、MACD、RSI、成交量均线。
2. 放量、缩量、突破、破位、长上影、长下影标签。
3. 放量突破策略。
4. 缩量回踩策略。
5. 趋势破位卖点。
6. 信号评分模型。
7. 策略信号页面。

验收标准：

```text
每日可生成全市场策略信号。
每个信号可解释触发原因。
每个信号有信号强度和失效条件。
```

### 26.4 第四阶段：回测系统

周期：2 周

目标：验证策略有效性。

交付：

1. 回测任务配置。
2. 策略回测执行。
3. 回测统计结果。
4. 分市场、分板块、分持有周期统计。
5. 策略效果页面。

验收标准：

```text
任意策略可以指定时间区间和股票池回测。
输出胜率、平均收益、盈亏比、最大回撤等指标。
回测结果可用于 AI 解释。
```

### 26.5 第五阶段：AI 分析

周期：2 周

目标：完成 AI 个股分析和每日复盘。

交付：

1. AI 工具调用框架。
2. 个股分析接口。
3. 信号解释接口。
4. 持仓每日复盘。
5. 相似案例检索。
6. AI 分析结果入库。
7. AI 复盘页面。

验收标准：

```text
AI 分析结论必须附带数据依据。
可以分析持仓股票的趋势、量价、财务、板块、风险。
可以生成买入条件、卖出条件、止损条件、失效条件。
```

### 26.6 第六阶段：公告财报 RAG

周期：2–3 周

目标：完成公告和财报文本检索分析。

交付：

1. 公告同步。
2. PDF/HTML 文本提取。
3. 文本切片。
4. 向量化入库。
5. 财报摘要。
6. 风险事件抽取。
7. 公告财报检索页面。

验收标准：

```text
可以针对某股票检索公告和财报内容。
AI 可以回答财报核心变化、风险点、管理层表述等问题。
```

### 26.7 第七阶段：盘中预警

周期：2–4 周

目标：实现实时辅助盯盘。

交付：

1. 实时行情接入。
2. Redis 行情快照。
3. 盘中预警规则。
4. WebSocket 推送。
5. 自选股盘中监控。
6. 预警中心页面。

验收标准：

```text
可对自选股进行放量、跌破均线、冲高回落、涨停开板等预警。
预警延迟满足个人盯盘使用要求。
```

---

## 27. 风险分析

### 27.1 数据源风险

| 风险 | 影响 | 应对 |
|---|---|---|
| 免费接口不稳定 | 数据缺失、任务失败 | 多数据源冗余、失败重试、补数任务 |
| 数据授权不明确 | 合规风险 | 仅个人研究使用，商业化前采购授权 |
| 实时数据延迟 | 盘中信号不准确 | 明确数据延迟等级，关键盘中场景使用券商源 |
| 财务数据口径差异 | 分析结论偏差 | 保留来源字段和版本 |

### 27.2 策略风险

| 风险 | 影响 | 应对 |
|---|---|---|
| 过拟合 | 回测好，实盘差 | 样本外验证、分阶段回测 |
| 幸存者偏差 | 高估策略收益 | 保留退市和历史成分数据 |
| 未来函数 | 回测失真 | 严格限制信号只使用当时可见数据 |
| 忽略交易成本 | 收益虚高 | 加入佣金、印花税、滑点 |
| 样本不足 | 统计不稳 | 设定最低交易次数阈值 |

### 27.3 AI 风险

| 风险 | 影响 | 应对 |
|---|---|---|
| 幻觉 | 输出不存在的数据或结论 | 工具调用 + 引用数据 + 输出约束 |
| 过度自信 | 误导交易 | 必须输出不确定性和风险条件 |
| 误读财报 | 错误归因 | 文档检索片段辅助，关键指标程序计算 |
| 直接荐股 | 合规风险 | 输出交易条件，不输出确定性建议 |

### 27.4 工程风险

| 风险 | 影响 | 应对 |
|---|---|---|
| 数据量增长过快 | 查询变慢、存储膨胀 | 分区、压缩、Parquet 归档 |
| 任务链路复杂 | 失败难排查 | job_run_log、链路追踪、任务幂等 |
| 指标版本混乱 | 信号不可复现 | 指标和策略都带 version |
| AI 成本不可控 | 调用费用高 | 缓存分析结果、批量复盘、按需调用 |
| 多语言复杂度 | 部署和排障复杂 | 主链路 Go 化，Python 仅作为可选研究工具 |

---

## 28. 评审关注点

### 28.1 架构评审问题

1. 当前 Go 单体 + Worker 架构是否满足 V1/V2 阶段需求？
2. TimescaleDB 是否足以支撑当前时序数据规模？
3. 是否需要提前引入 ClickHouse？
4. 数据湖 Parquet 是否作为长期归档标准？
5. AI 工具调用是否具备足够边界控制？
6. Python 是否需要进入主链路？建议结论：不需要。

### 28.2 数据评审问题

1. 股票代码、交易日、复权口径是否统一？
2. 原始数据是否可追溯？
3. 数据更新失败后是否可补数？
4. 财报和公告是否有来源、时间、版本？
5. 指标和信号是否支持重算？

### 28.3 策略评审问题

1. 买卖点公式是否存在未来函数？
2. 回测是否加入交易成本和滑点？
3. 策略信号是否有明确失效条件？
4. 策略是否经过样本外验证？
5. AI 是否只负责解释，而不直接作为交易信号？

### 28.4 前端评审问题

1. shadcn/ui 是否满足分析工作台的信息密度要求？
2. K 线和指标图是否采用专门图表组件？
3. 策略信号和 AI 分析是否分区展示？
4. 数据依据是否可追溯？
5. 预警和持仓复盘是否具备良好可读性？

### 28.5 合规评审问题

1. 数据源是否允许当前使用方式？
2. 是否存在对外分发行情数据风险？
3. AI 输出是否避免确定性荐股？
4. 是否明确系统只是辅助分析？
5. 是否保留免责声明和风险提示？

---

## 29. 最终推荐落地路径

建议采用“Go 主链路、先数据、再规则、后 AI、最后实时”的路径。

### 第一优先级

```text
quantsage monorepo
Go 后端工程骨架
React + shadcn/ui 前端工程骨架
PostgreSQL + TimescaleDB + pgvector
Tushare / 公开数据源同步
日线行情 + 财务指标
技术指标计算
短线信号生成
```

### 第二优先级

```text
策略 DSL
策略回测
信号评分
相似案例检索
每日持仓复盘
AI 个股分析
```

### 第三优先级

```text
公告财报 RAG
实时行情接入
盘中预警
Web 控制台完善
```

### 第四优先级

```text
Level-2 数据
ClickHouse
多策略组合分析
半自动交易辅助
Python 机器学习研究链路
```

---

## 30. 结论

QuantSage（量策智研）推荐建设为一个以 Golang 为主后端、React + shadcn/ui 为前端、PostgreSQL + TimescaleDB 为核心数据底座、pgvector 为文本检索能力、AI 工具调用为分析入口的股票数据中台与策略分析系统。

Python 不是必须项。它可以作为研究工具、Notebook 工具、机器学习训练工具或数据源原型验证工具存在，但不建议进入系统主链路。

最终系统判断逻辑应坚持：

```text
买卖点由公式和规则计算；
信号有效性由回测验证；
AI 负责解释、归因、过滤和生成交易计划；
最终交易决策由人工完成。
```

该方案具备以下优势：

1. Go 主系统部署简单、稳定性高。
2. React + shadcn/ui 适合构建现代化投研工作台。
3. 数据和信号可追溯。
4. 策略可回测、可验证。
5. AI 分析有数据约束，降低幻觉风险。
6. Python 可选，不增加主链路复杂度。
7. 系统可从个人单机版平滑演进到小团队级分析平台。

QuantSage 的最终目标不是让 AI 替代交易者，而是让交易者拥有一个：

> 有数据、有纪律、有复盘、有统计、有解释能力的决策辅助系统。
