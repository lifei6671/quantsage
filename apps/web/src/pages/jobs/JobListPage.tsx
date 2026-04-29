import {useState} from 'react'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'

import {getJobRuns, type JobRunItem, runJob, V1_TASKS} from '../../lib/api'
import {getTodayDateInput} from '../../lib/date'
import {sharedQueryKeys} from '../../lib/query'

export function JobListPage() {
  const [bizDate, setBizDate] = useState(() => getTodayDateInput())
  const [jobFilter, setJobFilter] = useState('')
  const [lastResult, setLastResult] = useState<{ jobName: string; status: string } | null>(null)
  const queryClient = useQueryClient()
  const jobsQuery = useQuery({
    queryKey: sharedQueryKeys.jobs(jobFilter, bizDate),
    queryFn: () => getJobRuns(jobFilter, bizDate, 1, 20),
  })

  const mutation = useMutation({
    mutationFn: ({ jobName, runDate }: { jobName: string; runDate: string }) => runJob(jobName, runDate),
    onSuccess: async (data) => {
      setLastResult({ jobName: data.job_name, status: data.status })
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: sharedQueryKeys.stocksRoot() }),
        queryClient.invalidateQueries({ queryKey: sharedQueryKeys.stockDailyRoot() }),
        queryClient.invalidateQueries({ queryKey: sharedQueryKeys.signalsRoot() }),
        queryClient.invalidateQueries({ queryKey: sharedQueryKeys.jobsRoot() }),
      ])
    },
  })

  const busyJobName = mutation.variables?.jobName ?? ''
  const jobRuns = jobsQuery.data?.items ?? []
  const hasFilters = jobFilter !== '' || bizDate !== ''

  return (
    <section className="page">
      <div className="page-header">
        <div>
          <h3>任务中心</h3>
          <p>手动触发 V1 数据任务，并查看最近的任务执行状态。</p>
        </div>

        <div className="toolbar toolbar-compact">
          <label className="field-inline">
            <span>业务日期</span>
            <input
              className="input"
              type="date"
              value={bizDate}
              onChange={(event) => setBizDate(event.target.value)}
            />
          </label>
        </div>
      </div>

      {mutation.isPending ? (
        <div className="notice info">
          <strong>{busyJobName || '任务'}</strong>
          <span>正在提交中，任务按钮已暂时禁用。</span>
        </div>
      ) : null}

      {mutation.error ? (
        <div className="notice danger">
          <strong>任务触发失败。</strong>
          <span>{String((mutation.error as Error).message)}</span>
        </div>
      ) : null}

      {lastResult ? (
        <div className="notice success">
          <strong>{lastResult.jobName}</strong>
          <span>已提交，状态 {lastResult.status}</span>
        </div>
      ) : (
        <div className="notice info">
          <strong>当前页面同时提供任务触发和状态查询。</strong>
          <span>触发成功后会刷新股票、日线、信号和任务列表缓存。</span>
        </div>
      )}

      <div className="task-grid">
        {V1_TASKS.map((task) => (
          <section className="surface task-card" key={task.name}>
            <div>
              <h4>{task.title}</h4>
              <p>{task.description}</p>
            </div>
            <div className="task-meta">
              <code>{task.name}</code>
              <button
                className="button"
                disabled={mutation.isPending}
                onClick={() => mutation.mutate({ jobName: task.name, runDate: bizDate })}
                type="button"
              >
                {mutation.isPending && busyJobName === task.name ? '提交中...' : '立即执行'}
              </button>
            </div>
          </section>
        ))}
      </div>

      <section className="surface task-panel">
        <div className="task-panel-header">
          <div>
            <h4>任务状态列表</h4>
            <p>查看最近 20 条任务执行记录，可按任务名和业务日期过滤。</p>
          </div>

          <div className="toolbar toolbar-compact">
            <label className="field-inline">
              <span>任务名</span>
              <select
                className="select"
                value={jobFilter}
                onChange={(event) => setJobFilter(event.target.value)}
              >
                <option value="">全部任务</option>
                {V1_TASKS.map((task) => (
                  <option key={task.name} value={task.name}>
                    {task.title}
                  </option>
                ))}
              </select>
            </label>
            <button className="button button-secondary" onClick={() => jobsQuery.refetch()} type="button">
              刷新列表
            </button>
          </div>
        </div>

        <div className="task-panel-meta">
          <span>筛选任务：{jobFilter || '全部任务'}</span>
          <span>业务日期：{bizDate || '全部日期'}</span>
          <span>返回记录：{jobRuns.length} 条</span>
        </div>

        {jobsQuery.error ? (
          <div className="notice warning">
            <strong>任务状态接口暂不可用。</strong>
            <span>{String((jobsQuery.error as Error).message)}</span>
          </div>
        ) : jobsQuery.isPending ? (
          <div className="notice info">
            <strong>正在加载任务状态。</strong>
            <span>稍等一下，执行记录马上刷新。</span>
          </div>
        ) : jobRuns.length > 0 ? (
          <div className="table-surface">
            <table className="data-table">
              <thead>
                <tr>
                  <th>业务日</th>
                  <th>任务名</th>
                  <th>状态</th>
                  <th>开始时间</th>
                  <th>结束时间</th>
                  <th>错误信息</th>
                </tr>
              </thead>
              <tbody>
                {jobRuns.map((item) => (
                  <tr key={item.id}>
                    <td>{item.biz_date}</td>
                    <td>{item.job_name}</td>
                    <td>
                      <span className={`badge ${jobStatusClassName(item.status)}`}>{jobStatusLabel(item.status)}</span>
                    </td>
                    <td>{formatTimestamp(item.started_at)}</td>
                    <td>{formatTimestamp(item.finished_at)}</td>
                    <td className="table-cell-wrap">{item.error_message || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="notice info">
            <strong>{hasFilters ? '当前筛选条件下没有任务记录。' : '暂时还没有任务执行记录。'}</strong>
            <span>可以先触发一个任务，或者调整筛选条件后再试一次。</span>
          </div>
        )}
      </section>
    </section>
  )
}

function formatTimestamp(value: string) {
  if (!value) {
    return '-'
  }

  return value.replace('T', ' ').replace('Z', ' UTC')
}

function jobStatusLabel(status: JobRunItem['status']) {
  switch (status) {
    case 'success':
      return '成功'
    case 'running':
      return '执行中'
    case 'failed':
      return '失败'
    case 'queued':
      return '已排队'
    default:
      return status || '-'
  }
}

function jobStatusClassName(status: JobRunItem['status']) {
  switch (status) {
    case 'success':
      return 'badge-positive'
    case 'running':
    case 'queued':
      return 'badge-neutral'
    case 'failed':
      return 'badge-danger'
    default:
      return 'badge-muted'
  }
}
