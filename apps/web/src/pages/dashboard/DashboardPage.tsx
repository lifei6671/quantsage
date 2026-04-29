import {type ReactNode, useMemo} from 'react'
import ReactECharts from 'echarts-for-react'

const watchlistRows = [
  ['中际旭创', '300308', '128.56', '+3.85%', '1.42', '突破', '买入观察'],
  ['英维克', '002837', '38.92', '+2.17%', '1.28', '突破', '持有'],
  ['沪电股份', '002463', '31.48', '-0.76%', '0.92', '回踩', '买入观察'],
  ['思源电气', '002028', '73.21', '-1.25%', '0.81', '谨慎', '谨慎'],
  ['汇川技术', '300124', '66.35', '+1.35%', '1.15', '回踩', '持有'],
  ['宁德时代', '300750', '214.18', '-0.42%', '0.88', '观望', '观望'],
]

const signalRows = [
  ['领益智造', '002600', '放量突破', '86%', '待确认'],
  ['应流股份', '603308', '回踩支撑', '78%', '观察中'],
  ['新易盛', '300502', '趋势延续', '74%', '观察中'],
  ['科华电源', '300153', '放量突破', '72%', '待确认'],
  ['中科曙光', '603019', '回踩支撑', '68%', '观察中'],
]

const activityRows = [
  ['success', '数据同步成功：A股日线数据更新完成', '10:32:21'],
  ['success', '策略任务执行完成：趋势动量策略（日线）', '09:41:08'],
  ['info', '自选股已更新：新增 3 只，移除 1 只', '09:15:47'],
  ['warning', '风险预警触发：高位量价背离预警（2 只）', '08:55:12'],
  ['info', '系统通知：版本更新 v1.2.3 已上线', '08:20:05'],
]

export function DashboardPage() {
  const chartOption = useMemo(() => buildMarketOption(), [])

  return (
    <section className="dashboard-page">
      <div className="metric-grid">
        <MetricCard
          title="A股概览"
          footer={
            <>
              <span>成交额 4,215.81 亿</span>
              <span className="danger-text">上涨 2136</span>
              <span className="success-text">下跌 871</span>
            </>
          }
        >
          <div className="market-value-row">
            <span>上证指数</span>
            <strong className="danger-text">3,128.42</strong>
            <span className="danger-text">+0.86%</span>
          </div>
          <MiniSparkline />
        </MetricCard>

        <MetricCard
          title="今日信号"
          footer={
            <>
              <span>强信号 10 条</span>
              <span>观察中 18 条</span>
            </>
          }
        >
          <MetricSpot tone="blue" icon="pulse" value="28" unit="条" sub="较昨日 +6" />
        </MetricCard>

        <MetricCard
          title="风险预警"
          footer={
            <>
              <span>高风险 2 条</span>
              <span>中风险 4 条</span>
            </>
          }
        >
          <MetricSpot tone="orange" icon="shield" value="6" unit="条" sub="较昨日 +2" />
        </MetricCard>

        <MetricCard
          title="任务状态"
          footer={
            <>
              <span>数据同步</span>
              <span className="success-text">正常</span>
            </>
          }
        >
          <MetricSpot tone="green" icon="check" value="7 / 7" unit="正常" sub="全部任务正常运行" />
        </MetricCard>
      </div>

      <div className="dashboard-main-grid">
        <section className="dash-panel market-panel">
          <PanelHeader title="市场趋势">
            <div className="segmented-control">
              <button className="active" type="button">1D</button>
              <button type="button">1W</button>
              <button type="button">1M</button>
            </div>
            <select className="compact-select" defaultValue="sse">
              <option value="sse">上证指数</option>
              <option value="szse">深证成指</option>
            </select>
          </PanelHeader>
          <ReactECharts option={chartOption} style={{ height: 240, width: '100%' }} />
        </section>

        <section className="dash-panel analysis-panel">
          <PanelHeader title="AI分析摘要" action="查看全部分析" />
          <div className="analysis-stack">
            <InsightRow icon="trend" tone="blue" title="信号强度较高：液冷、电源板块热度上升" tag="短线">
              资金流入明显，板块动量增强，关注龙头个股机会
            </InsightRow>
            <InsightRow icon="warn" tone="orange" title="需关注：部分高位个股量价背离" tag="风险">
              部分个股出现量能萎缩而价格创新高，警惕回调风险
            </InsightRow>
            <InsightRow icon="target" tone="green" title="建议：优先观察放量突破且回踩确认标的" tag="趋势">
              结合均线支撑与成交量变化，筛选胜率更高的标的
            </InsightRow>
          </div>
        </section>
      </div>

      <div className="dashboard-lower-grid">
        <section className="dash-panel">
          <PanelHeader title="自选股" action="查看全部" />
          <table className="console-table">
            <thead>
              <tr>
                <th>股票</th>
                <th>现价</th>
                <th>涨跌幅</th>
                <th>量比</th>
                <th>信号</th>
                <th>操作建议</th>
              </tr>
            </thead>
            <tbody>
              {watchlistRows.map((row) => (
                <tr key={row[1]}>
                  <td>
                    <strong>{row[0]}</strong>
                    <span>{row[1]}</span>
                  </td>
                  <td>{row[2]}</td>
                  <td className={row[3].startsWith('+') ? 'danger-text' : 'success-text'}>{row[3]}</td>
                  <td>{row[4]}</td>
                  <td><StatusPill value={row[5]} /></td>
                  <td><StatusPill value={row[6]} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>

        <section className="dash-panel">
          <PanelHeader title="买卖点信号" action="查看全部" />
          <table className="console-table signal-table">
            <thead>
              <tr>
                <th>#</th>
                <th>股票</th>
                <th>策略</th>
                <th>置信度</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              {signalRows.map((row, index) => (
                <tr key={row[1]}>
                  <td>{index + 1}</td>
                  <td>
                    <strong>{row[0]}</strong>
                    <span>{row[1]}</span>
                  </td>
                  <td>{row[2]}</td>
                  <td className={index === 0 ? 'danger-text' : 'warning-text'}>{row[3]}</td>
                  <td><StatusPill value={row[4]} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>

        <aside className="side-stack">
          <section className="dash-panel">
            <PanelHeader title="一期扩展点" />
            <div className="extension-grid">
              <ExtensionItem icon="chart" title="回测中心" note="预留" />
              <ExtensionItem icon="agent" title="AI Agent 协作" note="预留" />
              <ExtensionItem icon="globe" title="多市场接入" note="预留" />
            </div>
          </section>

          <section className="dash-panel activity-panel">
            <PanelHeader title="最新动态" action="查看全部" />
            <div className="activity-list">
              {activityRows.map((item) => (
                <div className="activity-item" key={item[1]}>
                  <span className={`activity-dot ${item[0]}`} />
                  <p>{item[1]}</p>
                  <time>{item[2]}</time>
                </div>
              ))}
            </div>
          </section>
        </aside>
      </div>
    </section>
  )
}

function MetricCard({ title, children, footer }: { title: string; children: ReactNode; footer: ReactNode }) {
  return (
    <section className="dash-panel metric-card">
      <PanelHeader title={title} />
      <div className="metric-body">{children}</div>
      <div className="metric-footer">{footer}</div>
    </section>
  )
}

function PanelHeader({ title, action, children }: { title: string; action?: string; children?: ReactNode }) {
  return (
    <header className="panel-header">
      <h3>{title}<span className="info-mark">i</span></h3>
      <div className="panel-actions">
        {children}
        {action ? <button type="button">{action}</button> : <button className="chevron-button" type="button">›</button>}
      </div>
    </header>
  )
}

function MetricSpot({ tone, icon, value, unit, sub }: { tone: string; icon: IconName; value: string; unit: string; sub: string }) {
  return (
    <div className="metric-spot">
      <span className={`round-icon ${tone}`}><Icon name={icon} /></span>
      <div>
        <strong>{value}<small>{unit}</small></strong>
        <p>{sub}</p>
      </div>
    </div>
  )
}

function MiniSparkline() {
  return (
    <svg className="mini-sparkline" viewBox="0 0 260 48" role="img" aria-label="上证指数日内走势">
      <polyline
        points="0,36 12,28 22,30 34,20 45,24 56,21 68,28 82,18 94,23 108,21 121,34 134,29 146,36 160,31 174,34 188,24 202,26 216,18 230,24 244,21 260,15"
      />
    </svg>
  )
}

function InsightRow({ icon, tone, title, tag, children }: { icon: IconName; tone: string; title: string; tag: string; children: ReactNode }) {
  return (
    <article className="insight-row">
      <span className={`square-icon ${tone}`}><Icon name={icon} /></span>
      <div>
        <strong>{title}</strong>
        <p>{children}</p>
      </div>
      <span className={`tag ${tone}`}>{tag}</span>
    </article>
  )
}

function StatusPill({ value }: { value: string }) {
  const tone = value.includes('买入') || value === '突破'
    ? 'red'
    : value.includes('持有') || value.includes('观察') || value === '回踩'
      ? 'blue'
      : value.includes('谨慎') || value.includes('待确认')
        ? 'orange'
        : 'gray'

  return <span className={`status-pill ${tone}`}>{value}</span>
}

function ExtensionItem({ icon, title, note }: { icon: IconName; title: string; note: string }) {
  return (
    <button className="extension-item" type="button">
      <Icon name={icon} />
      <strong>{title}</strong>
      <span>({note})</span>
    </button>
  )
}

type IconName = 'pulse' | 'shield' | 'check' | 'trend' | 'warn' | 'target' | 'chart' | 'agent' | 'globe'

function Icon({ name }: { name: IconName }) {
  const paths: Record<IconName, ReactNode> = {
    pulse: <path d="M3 12h4l2-6 4 12 2-6h6" />,
    shield: <path d="M12 3 5 6v6c0 4 3 7 7 9 4-2 7-5 7-9V6l-7-3Zm0 5v5m0 4h.01" />,
    check: <path d="M5 12l4 4L19 6" />,
    trend: <path d="M4 17V7m5 10V4m5 13v-7m5 7V6M4 17h16M5 12l4-4 4 3 6-7" />,
    warn: <path d="M12 4 3 20h18L12 4Zm0 5v5m0 3h.01" />,
    target: <path d="M12 21a9 9 0 1 0-9-9 9 9 0 0 0 9 9Zm0-5a4 4 0 1 0-4-4 4 4 0 0 0 4 4Zm0-4 7-7" />,
    chart: <path d="M4 18h16M6 15l4-4 3 3 5-7M6 7v8h12" />,
    agent: <path d="M7 9V7a5 5 0 0 1 10 0v2M5 10h14v8a3 3 0 0 1-3 3H8a3 3 0 0 1-3-3v-8Zm4 4h.01M15 14h.01" />,
    globe: <path d="M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Zm-9-9h18M12 3c3 3 3 15 0 18M12 3c-3 3-3 15 0 18" />,
  }

  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <g fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
        {paths[name]}
      </g>
    </svg>
  )
}

function buildMarketOption() {
  const times = ['09:30', '10:00', '10:30', '11:00', '11:30', '13:30', '14:00', '14:30', '15:00']
  const candleData = [
    [3090, 3108, 3085, 3116],
    [3108, 3088, 3078, 3112],
    [3088, 3132, 3080, 3148],
    [3132, 3120, 3104, 3150],
    [3120, 3156, 3118, 3168],
    [3156, 3136, 3128, 3172],
    [3136, 3094, 3088, 3140],
    [3094, 3126, 3086, 3138],
    [3126, 3128, 3112, 3145],
  ]

  return {
    animation: false,
    grid: [
      { left: 42, right: 44, top: 8, height: 132 },
      { left: 42, right: 44, top: 160, height: 46 },
    ],
    tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
    xAxis: [
      {
        type: 'category',
        data: times,
        boundaryGap: true,
        axisTick: { show: false },
        axisLine: { lineStyle: { color: '#d8e0ea' } },
        axisLabel: { color: '#68758a' },
      },
      {
        type: 'category',
        gridIndex: 1,
        data: times,
        boundaryGap: true,
        axisTick: { show: false },
        axisLine: { lineStyle: { color: '#d8e0ea' } },
        axisLabel: { color: '#68758a' },
      },
    ],
    yAxis: [
      {
        scale: true,
        position: 'right',
        splitLine: { lineStyle: { color: '#eef2f7' } },
        axisLabel: { color: '#68758a' },
      },
      {
        scale: true,
        gridIndex: 1,
        position: 'right',
        splitLine: { show: false },
        axisLabel: { color: '#68758a' },
      },
    ],
    series: [
      {
        type: 'candlestick',
        data: candleData,
        itemStyle: {
          color: '#16b36a',
          color0: '#ff4057',
          borderColor: '#16b36a',
          borderColor0: '#ff4057',
        },
      },
      {
        type: 'bar',
        xAxisIndex: 1,
        yAxisIndex: 1,
        data: [680, 520, 610, 420, 380, 450, 360, 410, 500],
        itemStyle: {
          color: (params: { dataIndex: number }) => (params.dataIndex % 2 === 0 ? '#ff7582' : '#54d192'),
        },
      },
    ],
  }
}
