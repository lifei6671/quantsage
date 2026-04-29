# QuantSage Web

## 本地开发

```bash
cd apps/web
npm install
npm run dev
```

开发服务器监听 `127.0.0.1:4173`，并通过 Vite proxy 将 `/api/*` 转发到 `http://127.0.0.1:8080`。

## 生产构建与预览

```bash
cd apps/web
npm run build
npm run preview
```

`preview` 同样会把 `/api/*` 转发到 `http://127.0.0.1:8080`，方便本地验证构建产物。

构建产物默认继续请求同源 `/api/*`。如果前后端分开部署、需要跨域访问，再通过 `VITE_API_BASE_URL` 显式指定 API 基地址。

跨域登录态要想真正可用，还需要服务端同步满足以下条件：

- `auth.allowed_origins` 必须显式包含当前前端 Origin
- 若前后端属于跨站点部署，需额外配置 `auth.session_same_site: none`
- `auth.session_same_site: none` 时必须同时开启 `auth.session_secure: true`

## 路由

前端使用 `HashRouter`，避免在没有服务端 SPA fallback 的场景下刷新详情页出现 404。

当前主要页面包括：

- `#/login`：登录页，成功后会回跳到原目标页面
- `#/watchlists`：当前用户的自选分组和股票条目管理
- `#/positions`：当前用户的持仓录入、编辑和删除
