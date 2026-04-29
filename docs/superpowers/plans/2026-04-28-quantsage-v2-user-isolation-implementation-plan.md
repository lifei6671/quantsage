# QuantSage V2 最简用户版实施计划

> **面向 Agent 工作者：** 必需子技能：使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans`，按任务逐项实施本计划。步骤使用复选框（`- [ ]`）语法追踪进度。

**目标：** 在保留 QuantSage V1 共享股票/行情/指标/信号底座的前提下，新增管理员预置账号、基于 Redis Session 的登录态，以及用户级自选股和持仓隔离。

**架构：** V2 继续沿用 `apps/server` 单模块 Gin + Worker 架构。共享数据仍存放在现有股票与行情事实表中；用户私有数据新增到 `app_user`、`watchlist_group`、`watchlist_item`、`user_position`。认证方式保持 `gin-contrib/sessions + Redis`，不引入 JWT、组织或团队。

**技术栈：** Go 1.22+、Gin、pgx/v5、sqlc、goose、gin-contrib/sessions、Redis、React + Vite + TanStack Query。

---

## 1. 已锁定决策

- 账号模式：管理员预置账号，不开放注册。
- 鉴权模式：Redis Session，不引入 JWT。
- 用户边界：只隔离自选股、持仓和后续用户私有业务数据；底层股票、行情、因子、信号全站共享。
- Schema 策略：当前项目仍在开发阶段，**直接修改现有 `migrations/postgres/000002_core_schema.sql`**，不新增新的迁移脚本。
- 旧结构策略：不保留旧 `watchlist` / `position` 的兼容双写逻辑，直接替换为新模型。
- 服务约束：所有用户私有查询一律通过当前 session 自动注入 `user_id`，前端不允许传 `user_id`。

## 2. 目标文件范围

预计涉及：

```text
migrations/postgres/000002_core_schema.sql
apps/server/sql/queries/*.sql
apps/server/internal/config/*
apps/server/internal/app/*
apps/server/internal/domain/user/*
apps/server/internal/domain/watchlist/*
apps/server/internal/domain/position/*
apps/server/internal/interfaces/http/middleware/*
apps/server/internal/interfaces/http/handler/*
apps/server/internal/interfaces/http/router.go
apps/server/cmd/quantsage-server/main.go
apps/web/src/app/App.tsx
apps/web/src/lib/api.ts
apps/web/src/lib/query.ts
apps/web/src/pages/login/*
apps/web/src/pages/watchlists/*
apps/web/src/pages/positions/*
docs/superpowers/specs/ai_stock_analysis_database_technical_proposal.md
README.md
docs/architecture/v1-local-runbook.md
```

## 3. 实施任务

### 任务 1：重构开发期 schema 为用户隔离模型

**文件：**

- 修改：`migrations/postgres/000002_core_schema.sql`
- 修改：`docs/superpowers/specs/ai_stock_analysis_database_technical_proposal.md`

- [ ] **步骤 1：新增用户私有表**

在 `000002_core_schema.sql` 中新增：

- `app_user`
- `watchlist_group`
- `watchlist_item`
- `user_position`

要求：

1. `app_user.username` 唯一。
2. `watchlist_group` 对 `(user_id, name)` 唯一。
3. `watchlist_item` 对 `(group_id, ts_code)` 唯一。
4. `user_position` 通过 `user_id + ts_code + position_date` 支持用户独立持仓录入。

- [ ] **步骤 2：删除旧的扁平业务表定义**

从 schema 中移除或替换旧的：

- `watchlist`
- `position`

说明：当前仍在开发阶段，不保留兼容结构。

- [ ] **步骤 3：补充必要索引与注释**

至少补：

- `watchlist_group(user_id)`
- `watchlist_item(group_id)`
- `user_position(user_id, position_date)`

- [ ] **步骤 4：更新技术方案文档中的表结构与边界说明**

确保技术方案同步反映：

- 共享数据与用户私有数据边界
- 不新增迁移脚本
- V2 最简用户版不含组织/团队

### 任务 2：增加配置与预置账号 bootstrap

**文件：**

- 修改：`configs/config.example.yaml`
- 修改：`apps/server/internal/config/config.go`
- 修改：`apps/server/internal/app/*`
- 新建或修改：`apps/server/internal/domain/user/*`

- [ ] **步骤 1：扩展配置模型**

增加 `auth` 配置段，至少支持：

- session secret
- bootstrap users 列表

- [ ] **步骤 2：定义用户领域模型与密码校验契约**

至少包括：

- 用户实体
- 按用户名查询
- 创建或同步 bootstrap 用户
- 密码哈希校验

- [ ] **步骤 3：实现服务启动时的 bootstrap**

要求：

1. server 启动时读取配置中的预置账号。
2. 若账号不存在则创建。
3. 若账号已存在，按用户名做有限字段更新。
4. 不记录明文密码；配置中只接受 hash。

- [ ] **步骤 4：补充单元测试**

覆盖：

- 配置解析
- bootstrap 幂等
- 重复用户名处理

### 任务 3：实现 Session 认证与当前用户上下文

**文件：**

- 修改：`apps/server/internal/interfaces/http/router.go`
- 新建或修改：`apps/server/internal/interfaces/http/middleware/auth.go`
- 新建：`apps/server/internal/interfaces/http/handler/auth_handler.go`
- 修改：`apps/server/cmd/quantsage-server/main.go`

- [ ] **步骤 1：接入 session store**

要求：

1. 继续使用 `gin-contrib/sessions + Redis`。
2. Cookie 至少开启 `HttpOnly`。
3. 生产环境预留 `Secure` 开关。

- [ ] **步骤 2：实现认证接口**

新增：

- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/me`

- [ ] **步骤 3：实现 auth middleware**

要求：

1. 从 session 中恢复 `user_id`。
2. 将当前用户写入 request context。
3. 未登录访问私有接口时返回统一未授权错误。

- [ ] **步骤 4：划分公开路由与私有路由**

公开路由至少保留：

- `/api/healthz`
- `/api/auth/login`

其余浏览器可访问的业务接口必须走鉴权中间件；local worker 只能通过 loopback-only 的内部任务接口触发 server 进程里的共享任务 runtime，不能依赖浏览器 session。

- [ ] **步骤 5：补充 HTTP 路由测试**

覆盖：

- 登录成功
- 密码错误
- 未登录访问私有接口
- 登出后 session 失效

### 任务 4：实现用户隔离的自选股与持仓服务

**文件：**

- 新建或修改：`apps/server/sql/queries/watchlists.sql`
- 新建或修改：`apps/server/sql/queries/positions.sql`
- 重新生成：`apps/server/internal/infra/db/dbgen/*`
- 新建：`apps/server/internal/domain/watchlist/*`
- 新建：`apps/server/internal/domain/position/*`
- 新建：`apps/server/internal/interfaces/http/handler/watchlist_handler.go`
- 新建：`apps/server/internal/interfaces/http/handler/position_handler.go`
- 修改：`apps/server/internal/interfaces/http/router.go`

- [ ] **步骤 1：定义用户隔离 SQL 契约**

所有查询必须带用户边界，例如：

- 按 `user_id` 查分组
- 按 `group_id + 当前用户归属` 查条目
- 按 `user_id` 查持仓

- [ ] **步骤 2：实现自选分组 CRUD**

接口：

- `GET /api/watchlists`
- `POST /api/watchlists`
- `PUT /api/watchlists/{id}`
- `DELETE /api/watchlists/{id}`

- [ ] **步骤 3：实现分组内股票 CRUD**

接口：

- `GET /api/watchlists/{id}/items`
- `POST /api/watchlists/{id}/items`
- `DELETE /api/watchlists/{id}/items/{item_id}`

- [ ] **步骤 4：实现持仓 CRUD**

接口：

- `GET /api/positions`
- `POST /api/positions`
- `PUT /api/positions/{id}`
- `DELETE /api/positions/{id}`

- [ ] **步骤 5：补充越权与隔离测试**

至少验证：

1. 用户 A 看不到用户 B 的分组。
2. 用户 A 不能删除用户 B 的分组条目。
3. 用户 A 看不到用户 B 的持仓。

### 任务 5：实现前端登录态与私有页面

**文件：**

- 修改：`apps/web/src/app/App.tsx`
- 修改：`apps/web/src/lib/api.ts`
- 修改：`apps/web/src/lib/query.ts`
- 新建：`apps/web/src/pages/login/LoginPage.tsx`
- 新建：`apps/web/src/pages/watchlists/*`
- 新建：`apps/web/src/pages/positions/*`
- 修改：`apps/web/src/index.css`

- [x] **步骤 1：新增认证 API client**

至少包括：

- `login`
- `logout`
- `getMe`

- [x] **步骤 2：增加登录页与路由守卫**

要求：

1. 未登录进入私有页面时跳转 `#/login`。
2. 登录成功后跳回默认页面。
3. 共享页面是否开放由产品决定；V2 默认工作台整体要求登录后访问。
4. 登录页按 QuantSage 品牌视觉完成左右分屏改版：左侧展示平台定位、行情图形背景和三项核心能力，右侧保留账号密码登录主流程。

- [x] **步骤 3：新增自选股页面**

展示：

- 分组列表
- 当前分组股票列表
- 新增 / 删除股票
- 分组重命名或删除

- [x] **步骤 4：新增持仓页面**

展示：

- 持仓列表
- 新增持仓
- 编辑数量 / 成本价 / 日期
- 删除持仓

- [x] **步骤 5：处理缓存失效与登出清理**

要求：

1. 登录后刷新 `me`、`watchlists`、`positions`。
2. 登出后清空所有私有缓存。
3. 共享数据缓存与私有数据缓存分开管理。

### 任务 6：文档、运行说明与最终验证

**文件：**

- 修改：`README.md`
- 修改：`docs/architecture/v1-local-runbook.md`
- 修改：`docs/superpowers/specs/ai_stock_analysis_database_technical_proposal.md`
- 新建：`docs/superpowers/plans/2026-04-28-quantsage-v2-user-isolation-implementation-plan.md`

- [ ] **步骤 1：更新 README 与 runbook**

增加：

- 预置账号说明
- 登录方式
- 用户隔离验证步骤

- [ ] **步骤 2：同步技术方案与计划**

确保文档中不再保留与 V2 冲突的旧 `watchlist` / `position` 叙述。

- [ ] **步骤 3：执行最终验证**

运行：

```bash
make fmt
make tidy
make build
make test
make race
make lint
cd apps/web && npm run build
```

预期：

- 后端构建、单测、竞态检测、静态检查全部通过
- 前端构建通过
- 用户隔离相关新增测试通过

## 4. V2 范围外

- 用户注册
- 密码找回
- 第三方 OAuth 登录
- 团队 / 组织 / 多租户
- 复杂 RBAC
- 用户私有行情副本
- 用户级独立策略运行沙箱
