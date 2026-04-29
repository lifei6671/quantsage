import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import {
  createWatchlistGroup,
  createWatchlistItem,
  deleteWatchlistGroup,
  deleteWatchlistItem,
  getWatchlistGroups,
  getWatchlistItems,
  updateWatchlistGroup,
} from '../../lib/api'
import { privateQueryKeys } from '../../lib/query'

export function WatchlistPage() {
  const queryClient = useQueryClient()
  const [selectedGroupID, setSelectedGroupID] = useState<number | null>(null)
  const [createGroupName, setCreateGroupName] = useState('')
  const [itemForm, setItemForm] = useState({ tsCode: '', note: '' })

  const groupsQuery = useQuery({
    queryKey: privateQueryKeys.watchlists(),
    queryFn: getWatchlistGroups,
  })

  const groups = groupsQuery.data ?? []
  const selectedGroup = groups.find((item) => item.id === selectedGroupID) ?? groups[0] ?? null
  const effectiveSelectedGroupID = selectedGroup?.id ?? null

  const itemsQuery = useQuery({
    queryKey: privateQueryKeys.watchlistItems(effectiveSelectedGroupID),
    queryFn: () => getWatchlistItems(effectiveSelectedGroupID as number),
    enabled: effectiveSelectedGroupID != null,
  })

  const createGroupMutation = useMutation({
    mutationFn: createWatchlistGroup,
    onSuccess: async (group) => {
      setCreateGroupName('')
      setSelectedGroupID(group.id)
      await queryClient.invalidateQueries({ queryKey: privateQueryKeys.watchlists() })
    },
  })

  const updateGroupMutation = useMutation({
    mutationFn: ({ groupID, group }: { groupID: number; group: { name: string; sort_order: number } }) =>
      updateWatchlistGroup(groupID, group),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: privateQueryKeys.watchlists() })
    },
  })

  const deleteGroupMutation = useMutation({
    mutationFn: deleteWatchlistGroup,
    onSuccess: async (_, groupID) => {
      if (selectedGroupID === groupID) {
        setSelectedGroupID(null)
      }
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: privateQueryKeys.watchlists() }),
        queryClient.removeQueries({ queryKey: privateQueryKeys.watchlistItems(groupID) }),
      ])
    },
  })

  const createItemMutation = useMutation({
    mutationFn: ({ groupID, item }: { groupID: number; item: { ts_code: string; note: string } }) =>
      createWatchlistItem(groupID, item),
    onSuccess: async (_, variables) => {
      setItemForm({ tsCode: '', note: '' })
      await queryClient.invalidateQueries({ queryKey: privateQueryKeys.watchlistItems(variables.groupID) })
    },
  })

  const deleteItemMutation = useMutation({
    mutationFn: ({ groupID, itemID }: { groupID: number; itemID: number }) => deleteWatchlistItem(groupID, itemID),
    onSuccess: async (_, variables) => {
      await queryClient.invalidateQueries({ queryKey: privateQueryKeys.watchlistItems(variables.groupID) })
    },
  })

  const items = itemsQuery.data ?? []
  const hasGroup = selectedGroup != null
  const mutationError =
    createGroupMutation.error ||
    updateGroupMutation.error ||
    deleteGroupMutation.error ||
    createItemMutation.error ||
    deleteItemMutation.error

  return (
    <section className="page">
      <div className="page-header">
        <div>
          <h3>自选股</h3>
          <p>每个账号维护独立分组和股票条目，默认不会与其他用户共享。</p>
        </div>
      </div>

      {mutationError ? (
        <div className="notice warning">
          <strong>自选股操作失败。</strong>
          <span>{String((mutationError as Error).message)}</span>
        </div>
      ) : null}

      <div className="management-grid">
        <section className="surface manage-panel">
          <div className="manage-panel-header">
            <div>
              <h4>分组列表</h4>
              <p>先创建分组，再向分组里维护股票。</p>
            </div>
          </div>

          <form
            className="inline-form"
            onSubmit={(event) => {
              event.preventDefault()
              createGroupMutation.mutate({
                name: createGroupName.trim(),
                sort_order: groups.length,
              })
            }}
          >
            <input
              className="input"
              onChange={(event) => setCreateGroupName(event.target.value)}
              placeholder="新增分组名称"
              value={createGroupName}
            />
            <button className="button" disabled={createGroupMutation.isPending || createGroupName.trim() === ''} type="submit">
              新增分组
            </button>
          </form>

          {groupsQuery.error ? (
            <div className="notice warning">
              <strong>分组接口暂不可用。</strong>
              <span>{String((groupsQuery.error as Error).message)}</span>
            </div>
          ) : groupsQuery.isPending ? (
            <div className="notice info">
              <strong>正在加载分组。</strong>
              <span>稍等一下，马上显示你的自选分类。</span>
            </div>
          ) : groups.length > 0 ? (
            <div className="group-list">
              {groups.map((group) => (
                <button
                  className={`group-card${group.id === effectiveSelectedGroupID ? ' active' : ''}`}
                  key={group.id}
                  onClick={() => setSelectedGroupID(group.id)}
                  type="button"
                >
                  <strong>{group.name}</strong>
                  <span>排序 {group.sort_order}</span>
                </button>
              ))}
            </div>
          ) : (
            <div className="notice info">
              <strong>还没有任何自选分组。</strong>
              <span>先创建一个分组，例如“趋势观察”或“高股息”。</span>
            </div>
          )}
        </section>

        <section className="surface manage-panel">
          <div className="manage-panel-header">
            <div>
              <h4>{selectedGroup?.name ?? '分组详情'}</h4>
              <p>支持重命名、调序和删除当前选中分组。</p>
            </div>
          </div>

          {hasGroup ? (
            <form
              className="form-grid"
              onSubmit={(event) => {
                event.preventDefault()
                if (effectiveSelectedGroupID == null) {
                  return
                }
                updateGroupMutation.mutate({
                  groupID: effectiveSelectedGroupID,
                  group: {
                    name: getFormValue(event.currentTarget, 'name').trim(),
                    sort_order: Number(getFormValue(event.currentTarget, 'sortOrder') || '0'),
                  },
                })
              }}
              key={effectiveSelectedGroupID}
            >
              <label className="form-stack">
                <span>分组名称</span>
                <input
                  className="input"
                  defaultValue={selectedGroup.name}
                  name="name"
                />
              </label>
              <label className="form-stack">
                <span>排序值</span>
                <input
                  className="input"
                  defaultValue={String(selectedGroup.sort_order)}
                  name="sortOrder"
                  type="number"
                />
              </label>
              <div className="form-actions">
                <button className="button" disabled={updateGroupMutation.isPending} type="submit">
                  保存分组
                </button>
                <button
                  className="button button-secondary"
                  disabled={deleteGroupMutation.isPending || effectiveSelectedGroupID == null}
                  onClick={() => {
                    if (effectiveSelectedGroupID != null) {
                      deleteGroupMutation.mutate(effectiveSelectedGroupID)
                    }
                  }}
                  type="button"
                >
                  删除分组
                </button>
              </div>
            </form>
          ) : (
            <div className="notice info">
              <strong>请选择一个分组。</strong>
              <span>右侧股票列表会跟随当前分组自动刷新。</span>
            </div>
          )}
        </section>
      </div>

      <section className="surface manage-panel">
        <div className="manage-panel-header">
          <div>
            <h4>{selectedGroup?.name ? `${selectedGroup.name} 的股票` : '分组股票列表'}</h4>
            <p>可以为当前分组新增股票和备注，也可以直接删除条目。</p>
          </div>
        </div>

        {hasGroup ? (
          <form
            className="inline-form"
            onSubmit={(event) => {
              event.preventDefault()
              if (effectiveSelectedGroupID == null) {
                return
              }
              createItemMutation.mutate({
                groupID: effectiveSelectedGroupID,
                item: {
                  ts_code: itemForm.tsCode.trim().toUpperCase(),
                  note: itemForm.note.trim(),
                },
              })
            }}
          >
            <input
              className="input"
              onChange={(event) => setItemForm((current) => ({ ...current, tsCode: event.target.value }))}
              placeholder="股票代码，例如 000001.SZ"
              value={itemForm.tsCode}
            />
            <input
              className="input"
              onChange={(event) => setItemForm((current) => ({ ...current, note: event.target.value }))}
              placeholder="备注"
              value={itemForm.note}
            />
            <button className="button" disabled={createItemMutation.isPending || itemForm.tsCode.trim() === ''} type="submit">
              新增股票
            </button>
          </form>
        ) : null}

        {itemsQuery.error ? (
          <div className="notice warning">
            <strong>分组股票接口暂不可用。</strong>
            <span>{String((itemsQuery.error as Error).message)}</span>
          </div>
        ) : itemsQuery.isPending && hasGroup ? (
          <div className="notice info">
            <strong>正在加载分组股票。</strong>
            <span>稍等一下，股票列表马上刷新。</span>
          </div>
        ) : items.length > 0 ? (
          <div className="table-surface">
            <table className="data-table">
              <thead>
                <tr>
                  <th>股票代码</th>
                  <th>备注</th>
                  <th>创建时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.id}>
                    <td>{item.ts_code}</td>
                    <td>{item.note || '-'}</td>
                    <td>{formatTimestamp(item.created_at)}</td>
                    <td>
                      <button
                        className="button button-secondary button-inline"
                        disabled={deleteItemMutation.isPending || effectiveSelectedGroupID == null}
                        onClick={() => {
                          if (effectiveSelectedGroupID != null) {
                            deleteItemMutation.mutate({ groupID: effectiveSelectedGroupID, itemID: item.id })
                          }
                        }}
                        type="button"
                      >
                        删除
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : hasGroup ? (
          <div className="notice info">
            <strong>当前分组里还没有股票。</strong>
            <span>可以先添加几只关注标的，后续再接个股详情联动。</span>
          </div>
        ) : (
          <div className="notice info">
            <strong>先选一个分组。</strong>
            <span>未选分组时不会加载股票条目。</span>
          </div>
        )}
      </section>
    </section>
  )
}

function formatTimestamp(value: string) {
  return value ? value.replace('T', ' ').replace('Z', ' UTC') : '-'
}

function getFormValue(form: HTMLFormElement, fieldName: string) {
  const value = new FormData(form).get(fieldName)
  return typeof value === 'string' ? value : ''
}
