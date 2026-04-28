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

## 路由

前端使用 `HashRouter`，避免在没有服务端 SPA fallback 的场景下刷新详情页出现 404。
