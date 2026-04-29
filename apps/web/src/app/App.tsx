import { lazy, Suspense, type ReactNode } from 'react'
import { NavLink, Navigate, Outlet, Route, Routes, useLocation, useNavigate } from 'react-router-dom'

import { useAuth } from './authContext'

const DashboardPage = lazy(async () => {
  const module = await import('../pages/dashboard/DashboardPage')
  return { default: module.DashboardPage }
})

const JobListPage = lazy(async () => {
  const module = await import('../pages/jobs/JobListPage')
  return { default: module.JobListPage }
})

const LoginPage = lazy(async () => {
  const module = await import('../pages/login/LoginPage')
  return { default: module.LoginPage }
})

const PositionPage = lazy(async () => {
  const module = await import('../pages/positions/PositionPage')
  return { default: module.PositionPage }
})

const SignalListPage = lazy(async () => {
  const module = await import('../pages/signals/SignalListPage')
  return { default: module.SignalListPage }
})

const StockDetailPage = lazy(async () => {
  const module = await import('../pages/stocks/StockDetailPage')
  return { default: module.StockDetailPage }
})

const StockListPage = lazy(async () => {
  const module = await import('../pages/stocks/StockListPage')
  return { default: module.StockListPage }
})

const WatchlistPage = lazy(async () => {
  const module = await import('../pages/watchlists/WatchlistPage')
  return { default: module.WatchlistPage }
})

const navigation = [
  { to: '/dashboard', label: '总览', icon: 'grid' },
  { to: '/stocks', label: '行情中心', icon: 'trend' },
  { to: '/watchlists', label: '自选股', icon: 'star' },
  { to: '/positions', label: '持仓管理', icon: 'wallet' },
  { to: '/signals', label: 'AI分析', icon: 'brain' },
  { to: '/trade-signals', label: '买卖点信号', icon: 'target' },
  { to: '/strategies', label: '策略中心', icon: 'briefcase' },
  { to: '/backtests', label: '回测中心', icon: 'bars' },
  { to: '/data', label: '数据管理', icon: 'database' },
  { to: '/jobs', label: '任务调度', icon: 'clock' },
  { to: '/risks', label: '风险预警', icon: 'shield' },
  { to: '/settings', label: '系统设置', icon: 'gear' },
]

export default function App() {
  return (
    <Suspense fallback={<div className="notice info app-loading">页面加载中...</div>}>
      <Routes>
        <Route path="/login" element={<PublicLoginRoute />} />
        <Route element={<RequireAuth />}>
          <Route element={<AppShell />}>
            <Route path="/" element={<Navigate replace to="/dashboard" />} />
            <Route path="/dashboard" element={<DashboardPage />} />
            <Route path="/stocks" element={<StockListPage />} />
            <Route path="/stocks/:id" element={<StockDetailPage />} />
            <Route path="/watchlists" element={<WatchlistPage />} />
            <Route path="/positions" element={<PositionPage />} />
            <Route path="/signals" element={<SignalListPage />} />
            <Route path="/jobs" element={<JobListPage />} />
            <Route path="/trade-signals" element={<ComingSoonPage title="买卖点信号" />} />
            <Route path="/strategies" element={<ComingSoonPage title="策略中心" />} />
            <Route path="/backtests" element={<ComingSoonPage title="回测中心" />} />
            <Route path="/data" element={<ComingSoonPage title="数据管理" />} />
            <Route path="/risks" element={<ComingSoonPage title="风险预警" />} />
            <Route path="/settings" element={<ComingSoonPage title="系统设置" />} />
          </Route>
        </Route>
        <Route path="*" element={<Navigate replace to="/" />} />
      </Routes>
    </Suspense>
  )
}

function PublicLoginRoute() {
  const { isAuthenticated, isChecking } = useAuth()
  if (isChecking) {
    return <div className="notice info app-loading">正在恢复登录态...</div>
  }
  if (isAuthenticated) {
    return <Navigate replace to="/dashboard" />
  }
  return <LoginPage />
}

function RequireAuth() {
  const { isAuthenticated, isChecking } = useAuth()
  const location = useLocation()
  if (isChecking) {
    return <div className="notice info app-loading">正在恢复登录态...</div>
  }
  if (!isAuthenticated) {
    const redirectTo = `${location.pathname}${location.search}${location.hash}`
    return <Navigate replace to="/login" state={{ redirectTo }} />
  }
  return <Outlet />
}

function AppShell() {
  const navigate = useNavigate()
  const { currentUser, logout, isLoggingOut } = useAuth()

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark" aria-hidden="true">
            <span />
          </div>
          <div>
            <h1>QuantSage</h1>
          </div>
        </div>

        <nav className="nav">
          {navigation.map((item) => (
            <NavLink
              key={item.to}
              className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
              to={item.to}
            >
              <NavIcon name={item.icon} />
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="sidebar-user-card">
          <span className="sidebar-user-label">当前登录</span>
          <strong>{currentUser?.display_name ?? currentUser?.username ?? '未登录'}</strong>
          <p>{currentUser?.role === 'admin' ? '管理员会话' : '用户会话'}</p>
        </div>
      </aside>

      <main className="main">
        <header className="topbar">
          <h2>控制台</h2>
          <div className="topbar-tools">
            <label className="search-box">
              <TopIcon name="search" />
              <input placeholder="搜索股票 / 策略 / 代码" />
            </label>
            <button className="date-button" type="button">
              <TopIcon name="calendar" />
              <span>2025-05-26</span>
              <span className="down-mark">⌄</span>
            </button>
            <button className="notification-button" type="button" aria-label="通知">
              <TopIcon name="bell" />
              <strong>3</strong>
            </button>
            <div className="user-menu">
              <span className="avatar">{(currentUser?.display_name ?? currentUser?.username ?? '管').slice(0, 1)}</span>
              <div className="user-menu-copy">
                <strong>{currentUser?.display_name ?? currentUser?.username ?? '未登录'}</strong>
                <span>{currentUser?.username ?? ''}</span>
              </div>
              <button
                className="button button-secondary user-menu-action"
                disabled={isLoggingOut}
                onClick={async () => {
                  try {
                    await logout()
                  } catch {
                    // 即使服务端登出请求失败，前端也已经清掉私有缓存，仍然应回到登录页避免旧数据继续停留在界面上。
                  }
                  navigate('/login', { replace: true })
                }}
                type="button"
              >
                {isLoggingOut ? '退出中...' : '退出登录'}
              </button>
            </div>
          </div>
        </header>

        <div className="content">
          <Outlet />
        </div>
      </main>
    </div>
  )
}

function ComingSoonPage({ title }: { title: string }) {
  return (
    <section className="empty-module">
      <h3>{title}</h3>
      <p>该模块将在 V2 用户隔离和私有数据能力中逐步接入。</p>
    </section>
  )
}

function NavIcon({ name }: { name: string }) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <g fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.8">
        {navPath(name)}
      </g>
    </svg>
  )
}

function TopIcon({ name }: { name: string }) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <g fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.8">
        {topPath(name)}
      </g>
    </svg>
  )
}

function navPath(name: string) {
  const paths: Record<string, ReactNode> = {
    grid: <path d="M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z" />,
    trend: <path d="M4 17h16M5 14l5-5 4 3 5-7" />,
    star: <path d="m12 3 2.7 5.5 6.1.9-4.4 4.3 1 6.1-5.4-2.9-5.4 2.9 1-6.1-4.4-4.3 6.1-.9L12 3z" />,
    wallet: <path d="M4 8h14a2 2 0 0 1 2 2v7H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2Zm0 0V6a2 2 0 0 1 2-2h10m4 8h-4v2h4" />,
    brain: <path d="M9 4a3 3 0 0 0-3 3v.5A3.5 3.5 0 0 0 6 14v1a3 3 0 0 0 5 2.2V4H9Zm6 0a3 3 0 0 1 3 3v.5a3.5 3.5 0 0 1 0 6.5v1a3 3 0 0 1-5 2.2V4h2Z" />,
    target: <path d="M12 21a9 9 0 1 0-9-9 9 9 0 0 0 9 9Zm0-5a4 4 0 1 0-4-4 4 4 0 0 0 4 4Zm0-4 7-7" />,
    briefcase: <path d="M9 7V5h6v2m-11 3h16v9H4zM4 10V7h16v3" />,
    bars: <path d="M5 20V9m7 11V4m7 16v-7" />,
    database: <path d="M4 7c0 2 4 4 8 4s8-2 8-4-4-4-8-4-8 2-8 4Zm0 0v10c0 2 4 4 8 4s8-2 8-4V7m-16 5c0 2 4 4 8 4s8-2 8-4" />,
    clock: <path d="M12 21a9 9 0 1 0-9-9 9 9 0 0 0 9 9Zm0-14v5l3 2" />,
    shield: <path d="M12 3 5 6v6c0 4 3 7 7 9 4-2 7-5 7-9V6l-7-3Zm0 6v4m0 4h.01" />,
    gear: <path d="M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Zm0-12v2m0 13v2m8.5-8.5h-2m-13 0h-2m14.4-6.4-1.4 1.4m-9.2 9.2-1.4 1.4m0-12 1.4 1.4m9.2 9.2 1.4 1.4" />,
  }

  return paths[name] ?? paths.grid
}

function topPath(name: string) {
  const paths: Record<string, ReactNode> = {
    search: <path d="m21 21-4.5-4.5M10.5 18a7.5 7.5 0 1 1 0-15 7.5 7.5 0 0 1 0 15Z" />,
    calendar: <path d="M7 3v4m10-4v4M4 9h16M5 5h14v15H5z" />,
    bell: <path d="M18 16v-5a6 6 0 1 0-12 0v5l-2 2h16l-2-2Zm-8 4h4" />,
  }

  return paths[name] ?? paths.search
}
