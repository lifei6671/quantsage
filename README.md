# quantsage

QuantSage 是一个面向本地研究与验证场景的 AI 量化工作台，当前 `V1` 提供：

- `apps/server`：Gin API、样例数据运行时、手动任务与本地 worker 调度
- `apps/web`：带登录态的股票、日线、策略信号、任务、自选股和持仓工作台
- `deployments/docker-compose`：本地 PostgreSQL / Redis / MinIO 依赖

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

前端默认运行在 `http://127.0.0.1:4173/#/stocks`，并通过 Vite proxy 请求同源 `/api/*`。

## 本地默认账号

当前 `configs/config.example.yaml` 会在服务启动时同步一个预置管理员账号：

- 用户名：`admin`
- 密码：`admin123`

这个默认密码只用于本地开发和冒烟验证；若在共享环境使用，请先替换 `bootstrap_users.password_hash`。

如果前后端分开部署、需要跨域使用登录态：

- 后端必须显式配置 `auth.allowed_origins`
- 若浏览器需要跨站点携带 session cookie，还必须同时配置 `auth.session_same_site: none` 和 `auth.session_secure: true`
