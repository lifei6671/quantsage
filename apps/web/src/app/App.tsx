import { lazy, Suspense } from 'react'
import { NavLink, Navigate, Route, Routes } from 'react-router-dom'

const JobListPage = lazy(async () => {
  const module = await import('../pages/jobs/JobListPage')
  return { default: module.JobListPage }
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

const navigation = [
  { to: '/stocks', label: '股票' },
  { to: '/signals', label: '信号' },
  { to: '/jobs', label: '任务' },
]

export default function App() {
  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark">QS</div>
          <div>
            <h1>QuantSage</h1>
            <p>V1 Trading Workspace</p>
          </div>
        </div>

        <nav className="nav">
          {navigation.map((item) => (
            <NavLink
              key={item.to}
              className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
              to={item.to}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="sidebar-note">
          <span className="sidebar-note-label">当前范围</span>
          <p>股票主数据、日线、信号与手动任务触发。</p>
        </div>
      </aside>

      <main className="main">
        <header className="topbar">
          <div>
            <span className="eyebrow">QuantSage Console</span>
            <h2>数据工作台</h2>
          </div>
          <div className="topbar-meta">
            <span className="status-dot" />
            <span>Local Preview</span>
          </div>
        </header>

        <div className="content">
          <Suspense fallback={<div className="notice info">页面加载中...</div>}>
            <Routes>
              <Route path="/" element={<Navigate replace to="/stocks" />} />
              <Route path="/stocks" element={<StockListPage />} />
              <Route path="/stocks/:id" element={<StockDetailPage />} />
              <Route path="/signals" element={<SignalListPage />} />
              <Route path="/jobs" element={<JobListPage />} />
            </Routes>
          </Suspense>
        </div>
      </main>
    </div>
  )
}
