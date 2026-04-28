import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'

import { getStocks } from '../../lib/api'

export function StockListPage() {
  const [keyword, setKeyword] = useState('')
  const [inputValue, setInputValue] = useState('')

  const query = useQuery({
    queryKey: ['stocks', keyword],
    queryFn: () => getStocks(keyword, 1, 20),
  })

  const items = query.data?.items ?? []

  return (
    <section className="page">
      <div className="page-header">
        <div>
          <h3>股票列表</h3>
          <p>筛选主数据并进入个股日线详情。</p>
        </div>

        <form
          className="toolbar"
          onSubmit={(event) => {
            event.preventDefault()
            setKeyword(inputValue.trim())
          }}
        >
          <input
            className="input"
            value={inputValue}
            onChange={(event) => setInputValue(event.target.value)}
            placeholder="按代码、简称或名称搜索"
          />
          <button className="button" type="submit">
            查询
          </button>
        </form>
      </div>

      {query.error ? (
        <div className="notice warning">
          <strong>股票接口暂不可用。</strong>
          <span>{String((query.error as Error).message)}</span>
        </div>
      ) : query.isPending ? (
        <div className="notice info">
          <strong>正在加载股票列表。</strong>
          <span>稍等一下，主数据马上刷新。</span>
        </div>
      ) : items.length > 0 ? (
        <div className="surface table-surface">
          <table className="data-table">
            <thead>
              <tr>
                <th>代码</th>
                <th>名称</th>
                <th>行业</th>
                <th>交易所</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.ts_code}>
                  <td>
                    <Link className="text-link" to={`/stocks/${encodeURIComponent(item.ts_code)}`}>
                      {item.ts_code}
                    </Link>
                  </td>
                  <td>{item.name}</td>
                  <td>{item.industry || '-'}</td>
                  <td>{item.exchange}</td>
                  <td>
                    <span className={`badge ${item.is_active ? 'badge-positive' : 'badge-muted'}`}>
                      {item.is_active ? '交易中' : '已停牌'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="notice info">
          <strong>当前条件下没有股票数据。</strong>
          <span>可以调整关键词后再试一次。</span>
        </div>
      )}
    </section>
  )
}
