import {useState} from 'react'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'

import {createPosition, deletePosition, getPositions, type PositionItem, updatePosition} from '../../lib/api'
import {getTodayDateInput} from '../../lib/date'
import {privateQueryKeys} from '../../lib/query'

type PositionFormState = {
  tsCode: string
  positionDate: string
  quantity: string
  costPrice: string
  note: string
}

const emptyForm = (): PositionFormState => ({
  tsCode: '',
  positionDate: getTodayDateInput(),
  quantity: '',
  costPrice: '',
  note: '',
})

export function PositionPage() {
  const queryClient = useQueryClient()
  const [editingPositionID, setEditingPositionID] = useState<number | null>(null)
  const [form, setForm] = useState<PositionFormState>(() => emptyForm())

  const positionsQuery = useQuery({
    queryKey: privateQueryKeys.positions(),
    queryFn: getPositions,
  })

  const createMutation = useMutation({
    mutationFn: createPosition,
    onSuccess: async () => {
      setForm(emptyForm())
      await queryClient.invalidateQueries({ queryKey: privateQueryKeys.positions() })
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ positionID, position }: { positionID: number; position: Parameters<typeof updatePosition>[1] }) =>
      updatePosition(positionID, position),
    onSuccess: async () => {
      setEditingPositionID(null)
      setForm(emptyForm())
      await queryClient.invalidateQueries({ queryKey: privateQueryKeys.positions() })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: deletePosition,
    onSuccess: async () => {
      if (editingPositionID != null) {
        setEditingPositionID(null)
        setForm(emptyForm())
      }
      await queryClient.invalidateQueries({ queryKey: privateQueryKeys.positions() })
    },
  })

  const mutationError = createMutation.error || updateMutation.error || deleteMutation.error
  const positions = positionsQuery.data ?? []

  return (
    <section className="page">
      <div className="page-header">
        <div>
          <h3>持仓管理</h3>
          <p>按当前登录用户维护独立持仓，用于后续策略验证和收益分析。</p>
        </div>
      </div>

      {mutationError ? (
        <div className="notice warning">
          <strong>持仓操作失败。</strong>
          <span>{String((mutationError as Error).message)}</span>
        </div>
      ) : null}

      <section className="surface manage-panel">
        <div className="manage-panel-header">
          <div>
            <h4>{editingPositionID == null ? '新增持仓' : '编辑持仓'}</h4>
            <p>支持录入股票代码、建仓日期、数量、成本价和备注。</p>
          </div>
        </div>

        <form
          className="form-grid position-form-grid"
          onSubmit={(event) => {
            event.preventDefault()
            const payload = {
              ts_code: form.tsCode.trim().toUpperCase(),
              position_date: form.positionDate,
              quantity: form.quantity.trim(),
              cost_price: form.costPrice.trim(),
              note: form.note.trim(),
            }
            if (editingPositionID == null) {
              createMutation.mutate(payload)
              return
            }
            updateMutation.mutate({
              positionID: editingPositionID,
              position: payload,
            })
          }}
        >
          <label className="form-stack">
            <span>股票代码</span>
            <input
              className="input"
              onChange={(event) => setForm((current) => ({ ...current, tsCode: event.target.value }))}
              placeholder="例如 600519.SH"
              value={form.tsCode}
            />
          </label>
          <label className="form-stack">
            <span>持仓日期</span>
            <input
              className="input"
              onChange={(event) => setForm((current) => ({ ...current, positionDate: event.target.value }))}
              type="date"
              value={form.positionDate}
            />
          </label>
          <label className="form-stack">
            <span>数量</span>
            <input
              className="input"
              onChange={(event) => setForm((current) => ({ ...current, quantity: event.target.value }))}
              placeholder="例如 200"
              value={form.quantity}
            />
          </label>
          <label className="form-stack">
            <span>成本价</span>
            <input
              className="input"
              onChange={(event) => setForm((current) => ({ ...current, costPrice: event.target.value }))}
              placeholder="例如 12.35"
              value={form.costPrice}
            />
          </label>
          <label className="form-stack form-span-full">
            <span>备注</span>
            <input
              className="input"
              onChange={(event) => setForm((current) => ({ ...current, note: event.target.value }))}
              placeholder="可记录建仓理由或风险提示"
              value={form.note}
            />
          </label>
          <div className="form-actions form-span-full">
            <button
              className="button"
              disabled={
                createMutation.isPending ||
                updateMutation.isPending ||
                form.tsCode.trim() === '' ||
                form.positionDate === '' ||
                form.quantity.trim() === '' ||
                form.costPrice.trim() === ''
              }
              type="submit"
            >
              {editingPositionID == null ? '新增持仓' : '保存修改'}
            </button>
            {editingPositionID != null ? (
              <button
                className="button button-secondary"
                onClick={() => {
                  setEditingPositionID(null)
                  setForm(emptyForm())
                }}
                type="button"
              >
                取消编辑
              </button>
            ) : null}
          </div>
        </form>
      </section>

      {positionsQuery.error ? (
        <div className="notice warning">
          <strong>持仓接口暂不可用。</strong>
          <span>{String((positionsQuery.error as Error).message)}</span>
        </div>
      ) : positionsQuery.isPending ? (
        <div className="notice info">
          <strong>正在加载持仓。</strong>
          <span>稍等一下，当前账号的持仓列表马上刷新。</span>
        </div>
      ) : positions.length > 0 ? (
        <div className="surface table-surface">
          <table className="data-table">
            <thead>
              <tr>
                <th>股票代码</th>
                <th>持仓日期</th>
                <th>数量</th>
                <th>成本价</th>
                <th>备注</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {positions.map((item) => (
                <tr key={item.id}>
                  <td>{item.ts_code}</td>
                  <td>{item.position_date}</td>
                  <td>{item.quantity}</td>
                  <td>{item.cost_price}</td>
                  <td className="table-cell-wrap">{item.note || '-'}</td>
                  <td>
                    <div className="row-actions">
                      <button
                        className="button button-secondary button-inline"
                        onClick={() => {
                          setEditingPositionID(item.id)
                          setForm(buildFormFromPosition(item))
                        }}
                        type="button"
                      >
                        编辑
                      </button>
                      <button
                        className="button button-secondary button-inline"
                        disabled={deleteMutation.isPending}
                        onClick={() => deleteMutation.mutate(item.id)}
                        type="button"
                      >
                        删除
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="notice info">
          <strong>当前账号还没有持仓记录。</strong>
          <span>先录入一条持仓，后续页面就可以按用户维度展示仓位。</span>
        </div>
      )}
    </section>
  )
}

function buildFormFromPosition(item: PositionItem): PositionFormState {
  return {
    tsCode: item.ts_code,
    positionDate: item.position_date,
    quantity: item.quantity,
    costPrice: item.cost_price,
    note: item.note,
  }
}
