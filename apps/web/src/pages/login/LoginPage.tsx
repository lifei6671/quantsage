import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

import { useAuth } from '../../app/authContext'

type LoginRouteState = {
  redirectTo?: string
}

export function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const { login, isLoggingIn } = useAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [rememberMe, setRememberMe] = useState(false)
  const [errorText, setErrorText] = useState('')
  const routeState = (location.state as LoginRouteState | null) ?? null

  return (
    <main className="login-layout">
      <div className="login-top-links" aria-label="辅助链接">
        <a href="/help" onClick={(event) => event.preventDefault()}>
          <LoginIcon name="help" />
          帮助中心
        </a>
        <span aria-hidden="true" />
        <a href="/" onClick={(event) => event.preventDefault()}>
          <LoginIcon name="globe" />
          返回官网
        </a>
      </div>

      <section className="login-hero" aria-label="QuantSage 登录">
        <div className="login-brand-panel">
          <div className="login-brand-lockup">
            <QuantSageLogo />
            <strong>QuantSage</strong>
          </div>

          <div className="login-copy">
            <h1>
              <span>更聪明的</span>量化投资决策平台
            </h1>
            <p>聚合行情、自选股、AI 分析、买卖点信号与策略管理，帮助你高效发现机会、识别风险。</p>
          </div>

          <div className="login-feature-list" aria-label="核心能力">
            <FeatureCard icon="brain" title="AI 分析摘要" description="自动提炼热点、趋势与风险" tone="blue" />
            <FeatureCard icon="signal" title="买卖点信号" description="量价结合，辅助择时" tone="green" />
            <FeatureCard icon="layers" title="策略与数据管理" description="统一管理任务、策略与数据" tone="purple" />
          </div>

          <div className="login-market-scene" aria-hidden="true">
            <div className="login-chart-grid" />
            <div className="login-candle-row">
              {['short', 'mid', 'tall', 'short', 'mid', 'tall', 'short', 'mid', 'short', 'tall', 'mid', 'short'].map(
                (size, index) => (
                  <span className={`login-candle ${size}`} key={`${size}-${index}`} />
                ),
              )}
            </div>
            <svg className="login-trend-line" viewBox="0 0 520 260" preserveAspectRatio="none">
              <path d="M4 238 C72 208 88 196 126 202 C170 208 168 160 214 152 C260 144 260 98 308 110 C356 122 374 76 424 84 C464 92 482 44 516 28" />
              <circle cx="126" cy="202" r="5" />
              <circle cx="214" cy="152" r="5" />
              <circle cx="308" cy="110" r="5" />
              <circle cx="424" cy="84" r="5" />
            </svg>
            <div className="login-wave login-wave-a" />
            <div className="login-wave login-wave-b" />
          </div>
        </div>

        <div className="login-card-wrap">
          <form
            className="login-card"
            onSubmit={async (event) => {
              event.preventDefault()
              setErrorText('')
              try {
                await login({
                  username: username.trim(),
                  password,
                })
                navigate(routeState?.redirectTo || '/dashboard', { replace: true })
              } catch (error) {
                setErrorText(String((error as Error).message))
              }
            }}
          >
            <QuantSageLogo compact />
            <div className="login-card-heading">
              <h2>欢迎登录</h2>
              <p>登录 QuantSage，开始今日投资分析</p>
            </div>

            <div className="login-tabs" role="tablist" aria-label="登录方式">
              <button className="active" type="button" role="tab" aria-selected="true">
                账号登录
              </button>
              <button type="button" role="tab" aria-selected="false">
                验证码登录
              </button>
            </div>

            <label className="login-field">
              <span>用户名 / 邮箱</span>
              <div className="login-input-shell">
                <LoginIcon name="user" />
                <input
                  autoComplete="username"
                  onChange={(event) => setUsername(event.target.value)}
                  placeholder="请输入用户名或邮箱"
                  value={username}
                />
              </div>
            </label>

            <label className="login-field">
              <span>密码</span>
              <div className="login-input-shell">
                <LoginIcon name="lock" />
                <input
                  autoComplete="current-password"
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="请输入密码"
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                />
                <button
                  className="login-icon-button"
                  onClick={() => setShowPassword((visible) => !visible)}
                  type="button"
                  aria-label={showPassword ? '隐藏密码' : '显示密码'}
                >
                  <LoginIcon name={showPassword ? 'eyeOff' : 'eye'} />
                </button>
              </div>
            </label>

            <div className="login-form-row">
              <label className="login-checkbox">
                <input
                  checked={rememberMe}
                  onChange={(event) => setRememberMe(event.target.checked)}
                  type="checkbox"
                />
                <span>记住我</span>
              </label>
              <button className="login-text-button" type="button">
                忘记密码?
              </button>
            </div>

            {errorText ? (
              <div className="login-error" role="alert">
                {errorText}
              </div>
            ) : null}

            <button className="login-primary-button" disabled={isLoggingIn} type="submit">
              {isLoggingIn ? '登录中...' : '登录'}
            </button>

            <button className="login-ghost-button" type="button">
              游客体验
            </button>

            <div className="login-divider">
              <span>其他方式</span>
            </div>

            <div className="login-oauth-row" aria-label="第三方登录">
              <button type="button">
                <LoginIcon name="wechat" />
                微信
              </button>
              <button type="button">
                <LoginIcon name="github" />
                GitHub
              </button>
            </div>

            <p className="login-register">
              还没有账号?
              <button type="button">立即注册</button>
            </p>
          </form>
        </div>
      </section>

      <footer className="login-footer">
        <span>© 2025 QuantSage. All rights reserved.</span>
        <span aria-hidden="true" />
        <a href="/privacy" onClick={(event) => event.preventDefault()}>
          隐私政策
        </a>
        <span aria-hidden="true" />
        <a href="/terms" onClick={(event) => event.preventDefault()}>
          服务条款
        </a>
      </footer>
    </main>
  )
}

function FeatureCard({
  icon,
  title,
  description,
  tone,
}: {
  icon: 'brain' | 'signal' | 'layers'
  title: string
  description: string
  tone: 'blue' | 'green' | 'purple'
}) {
  return (
    <article className="login-feature-card">
      <span className={`login-feature-icon ${tone}`}>
        <LoginIcon name={icon} />
      </span>
      <span>
        <strong>{title}</strong>
        <small>{description}</small>
      </span>
    </article>
  )
}

function QuantSageLogo({ compact = false }: { compact?: boolean }) {
  return (
    <span className={`login-logo${compact ? ' compact' : ''}`} aria-hidden="true">
      <span />
    </span>
  )
}

function LoginIcon({
  name,
}: {
  name:
    | 'brain'
    | 'signal'
    | 'layers'
    | 'help'
    | 'globe'
    | 'user'
    | 'lock'
    | 'eye'
    | 'eyeOff'
    | 'wechat'
    | 'github'
}) {
  if (name === 'wechat') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path
          fill="#22c55e"
          d="M9.4 4.5c-4 0-7.2 2.6-7.2 5.9 0 1.9 1.1 3.6 2.8 4.7l-.7 2.2 2.5-1.3c.8.2 1.6.3 2.6.3.4 0 .8 0 1.2-.1-.2-.6-.3-1.2-.3-1.8 0-3.1 2.9-5.6 6.4-5.6h.5c-.8-2.5-3.9-4.3-7.8-4.3Zm-2.6 4.9a.9.9 0 1 1 0-1.8.9.9 0 0 1 0 1.8Zm5.1 0a.9.9 0 1 1 0-1.8.9.9 0 0 1 0 1.8Z"
        />
        <path
          fill="#22c55e"
          d="M16.7 10c-3 0-5.4 2-5.4 4.5s2.4 4.5 5.4 4.5c.7 0 1.3-.1 1.9-.3l2 1-.5-1.7c1.2-.9 2-2.1 2-3.5 0-2.5-2.4-4.5-5.4-4.5Zm-1.8 3.7a.7.7 0 1 1 0-1.4.7.7 0 0 1 0 1.4Zm3.8 0a.7.7 0 1 1 0-1.4.7.7 0 0 1 0 1.4Z"
        />
      </svg>
    )
  }

  if (name === 'github') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path
          fill="currentColor"
          d="M12 2.4a9.6 9.6 0 0 0-3 18.7c.5.1.7-.2.7-.5v-1.8c-2.8.6-3.4-1.2-3.4-1.2-.5-1.1-1.1-1.4-1.1-1.4-.9-.6.1-.6.1-.6 1 0 1.6 1.1 1.6 1.1.9 1.5 2.3 1.1 2.9.8.1-.6.4-1.1.6-1.3-2.2-.3-4.6-1.1-4.6-4.8 0-1.1.4-2 1.1-2.7-.1-.3-.5-1.3.1-2.7 0 0 .9-.3 2.8 1a9.4 9.4 0 0 1 5.1 0c1.9-1.3 2.8-1 2.8-1 .6 1.4.2 2.4.1 2.7.7.7 1.1 1.6 1.1 2.7 0 3.7-2.4 4.5-4.6 4.8.4.3.7.9.7 1.9v2.8c0 .3.2.6.7.5A9.6 9.6 0 0 0 12 2.4Z"
        />
      </svg>
    )
  }

  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <g fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.8">
        {loginIconPath(name)}
      </g>
    </svg>
  )
}

function loginIconPath(name: Exclude<Parameters<typeof LoginIcon>[0]['name'], 'wechat' | 'github'>) {
  switch (name) {
    case 'brain':
      return (
        <>
          <path d="M9 4.8a3 3 0 0 0-4.1 3.8 3.1 3.1 0 0 0 .2 5.7A3 3 0 0 0 9 19.2V4.8Z" />
          <path d="M15 4.8a3 3 0 0 1 4.1 3.8 3.1 3.1 0 0 1-.2 5.7 3 3 0 0 1-3.9 4.9V4.8Z" />
          <path d="M9 9h3m-3 4h6m0-4h-1.5M12 13v3" />
        </>
      )
    case 'signal':
      return (
        <>
          <path d="M4 18V9m5 9V5m5 13v-7m5 7V7" />
          <path d="m4 14 5-4 5 2 5-6" />
          <path d="M18 6h1v4" />
        </>
      )
    case 'layers':
      return (
        <>
          <path d="m12 3 8 4-8 4-8-4 8-4Z" />
          <path d="m4 12 8 4 8-4" />
          <path d="m4 17 8 4 8-4" />
        </>
      )
    case 'help':
      return (
        <>
          <circle cx="12" cy="12" r="9" />
          <path d="M9.6 9.2a2.6 2.6 0 1 1 4.6 1.7c-.8.8-1.4 1.2-1.7 2.4" />
          <path d="M12 16.8h.01" />
        </>
      )
    case 'globe':
      return (
        <>
          <circle cx="12" cy="12" r="9" />
          <path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18" />
        </>
      )
    case 'user':
      return (
        <>
          <circle cx="12" cy="8" r="3.2" />
          <path d="M5.4 20a6.8 6.8 0 0 1 13.2 0" />
        </>
      )
    case 'lock':
      return (
        <>
          <rect width="12" height="9" x="6" y="11" rx="2" />
          <path d="M8.5 11V8a3.5 3.5 0 0 1 7 0v3" />
        </>
      )
    case 'eye':
      return (
        <>
          <path d="M2.8 12s3.4-5.6 9.2-5.6 9.2 5.6 9.2 5.6-3.4 5.6-9.2 5.6S2.8 12 2.8 12Z" />
          <circle cx="12" cy="12" r="2.6" />
        </>
      )
    case 'eyeOff':
      return (
        <>
          <path d="m4 4 16 16" />
          <path d="M9.5 5.9a9.6 9.6 0 0 1 2.5-.3c5.8 0 9.2 6.4 9.2 6.4a15 15 0 0 1-2.8 3.5" />
          <path d="M6.4 7.8A15.4 15.4 0 0 0 2.8 12s3.4 6.4 9.2 6.4c1.3 0 2.5-.3 3.5-.8" />
        </>
      )
  }
}
