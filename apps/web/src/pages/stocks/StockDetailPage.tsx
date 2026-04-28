import { useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import ReactECharts from 'echarts-for-react'

import { getStockDaily } from '../../lib/api'

export function StockDetailPage() {
  const { id } = useParams()
  const tsCode = decodeURIComponent(id ?? '000001.SZ')
  const [endDate, setEndDate] = useState(() => formatDateInput(new Date()))
  const [startDate, setStartDate] = useState(() => {
    const value = new Date()
    value.setDate(value.getDate() - 29)
    return formatDateInput(value)
  })

  const query = useQuery({
    queryKey: ['stock-daily', tsCode, startDate, endDate],
    queryFn: () => getStockDaily(tsCode, startDate, endDate),
  })

  const bars = useMemo(() => query.data ?? [], [query.data])
  const latest = bars[bars.length - 1]

  const chartOption = useMemo(() => {
    return {
      tooltip: { trigger: 'axis' },
      grid: [
        { left: 48, right: 24, top: 18, height: 200 },
        { left: 48, right: 24, top: 248, height: 72 },
      ],
      xAxis: [
        {
          type: 'category',
          data: bars.map((item) => item.trade_date),
          boundaryGap: true,
          axisLine: { lineStyle: { color: '#cfd6de' } },
        },
        {
          type: 'category',
          gridIndex: 1,
          data: bars.map((item) => item.trade_date),
          boundaryGap: true,
          axisLabel: { show: false },
          axisLine: { lineStyle: { color: '#cfd6de' } },
        },
      ],
      yAxis: [
        {
          scale: true,
          splitLine: { lineStyle: { color: '#e7eaef' } },
        },
        {
          scale: true,
          gridIndex: 1,
          splitLine: { show: false },
        },
      ],
      series: [
        {
          type: 'candlestick',
          data: bars.map((item) => [item.open, item.close, item.low, item.high]),
          itemStyle: {
            color: '#0f9d58',
            color0: '#c23b22',
            borderColor: '#0f9d58',
            borderColor0: '#c23b22',
          },
        },
        {
          type: 'bar',
          xAxisIndex: 1,
          yAxisIndex: 1,
          data: bars.map((item) => item.vol),
          itemStyle: { color: '#7d8a99' },
        },
      ],
    }
  }, [bars])

  return (
    <section className="page">
      <div className="page-header">
        <div>
          <Link className="back-link" to="/stocks">
            返回股票列表
          </Link>
          <h3>{tsCode}</h3>
          <p>查看最近交易日的 K 线走势与成交量变化。</p>
        </div>

        <div className="toolbar toolbar-compact">
          <label className="field-inline">
            <span>开始</span>
            <input
              className="input"
              type="date"
              value={startDate}
              onChange={(event) => setStartDate(event.target.value)}
            />
          </label>
          <label className="field-inline">
            <span>结束</span>
            <input
              className="input"
              type="date"
              value={endDate}
              onChange={(event) => setEndDate(event.target.value)}
            />
          </label>
        </div>
      </div>

      {query.error ? (
        <div className="notice warning">
          <strong>后端日线接口暂不可用。</strong>
          <span>{String((query.error as Error).message)}</span>
        </div>
      ) : null}

      <div className="stats-grid">
        <div className="stat-panel">
          <span>最新收盘</span>
          <strong>{latest?.close ?? '--'}</strong>
        </div>
        <div className="stat-panel">
          <span>涨跌幅</span>
          <strong>{latest?.pct_chg ?? '--'}%</strong>
        </div>
        <div className="stat-panel">
          <span>成交量</span>
          <strong>{latest?.vol ?? '--'}</strong>
        </div>
        <div className="stat-panel">
          <span>成交额</span>
          <strong>{latest?.amount ?? '--'}</strong>
        </div>
      </div>

      {query.error ? null : query.isPending ? (
        <div className="notice info">
          <strong>正在加载日线数据。</strong>
          <span>稍等一下，图表和明细马上出来。</span>
        </div>
      ) : bars.length > 0 ? (
        <>
          <div className="surface chart-surface">
            <ReactECharts option={chartOption} style={{ height: 340, width: '100%' }} />
          </div>

          <div className="surface table-surface">
            <table className="data-table">
              <thead>
                <tr>
                  <th>日期</th>
                  <th>开盘</th>
                  <th>最高</th>
                  <th>最低</th>
                  <th>收盘</th>
                  <th>涨跌幅</th>
                </tr>
              </thead>
              <tbody>
                {bars.map((item) => (
                  <tr key={item.trade_date}>
                    <td>{item.trade_date}</td>
                    <td>{item.open}</td>
                    <td>{item.high}</td>
                    <td>{item.low}</td>
                    <td>{item.close}</td>
                    <td>{item.pct_chg}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      ) : (
        <div className="notice info">
          <strong>当前时间窗口没有可展示的日线数据。</strong>
          <span>调整日期范围后再试一次。</span>
        </div>
      )}
    </section>
  )
}

function formatDateInput(value: Date) {
  const year = value.getFullYear()
  const month = String(value.getMonth() + 1).padStart(2, '0')
  const day = String(value.getDate()).padStart(2, '0')

  return `${year}-${month}-${day}`
}
