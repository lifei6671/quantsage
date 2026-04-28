import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'

import { getSignals } from '../../lib/api'
import { getTodayDateInput } from '../../lib/date'

export function SignalListPage() {
  const [tradeDate, setTradeDate] = useState(() => getTodayDateInput())
  const [strategyCode, setStrategyCode] = useState('')

  const query = useQuery({
    queryKey: ['signals', tradeDate, strategyCode],
    queryFn: () => getSignals(tradeDate, strategyCode, 1, 20),
  })

  const items = query.data?.items ?? []

  return (
    <section className="page">
      <div className="page-header">
        <div>
          <h3>策略信号</h3>
          <p>按交易日筛选固定策略输出的买卖点。</p>
        </div>

        <div className="toolbar toolbar-compact">
          <input
            className="input"
            type="date"
            value={tradeDate}
            onChange={(event) => setTradeDate(event.target.value)}
          />
          <select
            className="select"
            value={strategyCode}
            onChange={(event) => setStrategyCode(event.target.value)}
          >
            <option value="">全部策略</option>
            <option value="volume_breakout_v1">放量突破</option>
            <option value="trend_break_v1">趋势破位</option>
          </select>
        </div>
      </div>

      {query.error ? (
        <div className="notice warning">
          <strong>信号接口暂不可用。</strong>
          <span>{String((query.error as Error).message)}</span>
        </div>
      ) : query.isPending ? (
        <div className="notice info">
          <strong>正在加载策略信号。</strong>
          <span>稍等一下，列表马上刷新。</span>
        </div>
      ) : items.length > 0 ? (
        <div className="surface table-surface">
          <table className="data-table">
            <thead>
              <tr>
                <th>交易日</th>
                <th>策略</th>
                <th>标的</th>
                <th>方向</th>
                <th>强度</th>
                <th>等级</th>
                <th>买入参考</th>
                <th>止损</th>
                <th>止盈</th>
                <th>原因</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={`${item.strategy_code}-${item.ts_code}-${item.trade_date}-${item.signal_type}`}>
                  <td>{item.trade_date}</td>
                  <td>{item.strategy_code}</td>
                  <td>{item.ts_code}</td>
                  <td>{item.signal_type === 'buy_signal' ? '买入' : '卖出'}</td>
                  <td>{item.signal_strength}</td>
                  <td>
                    <span className={`badge ${item.signal_level === 'A' ? 'badge-positive' : 'badge-neutral'}`}>
                      {item.signal_level}
                    </span>
                  </td>
                  <td>{item.buy_price_ref}</td>
                  <td>{item.stop_loss_ref}</td>
                  <td>{item.take_profit_ref}</td>
                  <td>{item.reason}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="notice info">
          <strong>当前筛选条件下没有策略信号。</strong>
          <span>可以切换交易日或策略后再试一次。</span>
        </div>
      )}
    </section>
  )
}
