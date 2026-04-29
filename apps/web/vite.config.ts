import {defineConfig, loadEnv, type ProxyOptions} from 'vite'
import react from '@vitejs/plugin-react'

const defaultAPIProxyTarget = 'http://127.0.0.1:8080'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const apiProxyTarget = normalizeProxyTarget(env.VITE_DEV_API_PROXY_TARGET || defaultAPIProxyTarget)

  return {
    plugins: [react()],
    preview: {
      host: '127.0.0.1',
      port: 4173,
      proxy: buildAPIProxy(apiProxyTarget),
    },
    server: {
      host: '127.0.0.1',
      port: 4173,
      proxy: buildAPIProxy(apiProxyTarget),
    },
  }
})

function buildAPIProxy(target: string): Record<string, ProxyOptions> {
  return {
    '/api': {
      target,
      changeOrigin: true,
      secure: false,
      configure(proxy) {
        proxy.on('proxyReq', (proxyReq) => {
          // 代理到后端时保留同源 Cookie 登录体验，同时显式标记为本地开发代理请求。
          proxyReq.setHeader('X-Forwarded-Host', '127.0.0.1:4173')
          proxyReq.setHeader('X-Forwarded-Proto', 'http')
        })
      },
    },
  }
}

function normalizeProxyTarget(value: string) {
  const trimmed = value.trim().replace(/\/+$/, '')
  return trimmed || defaultAPIProxyTarget
}
