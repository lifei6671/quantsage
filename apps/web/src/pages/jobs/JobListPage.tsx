import { useMemo, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'

import { runJob, V1_TASKS } from '../../lib/api'
import { getTodayDateInput } from '../../lib/date'

export function JobListPage() {
  const [bizDate, setBizDate] = useState(() => getTodayDateInput())
  const [lastResult, setLastResult] = useState<{ jobName: string; status: string } | null>(null)
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: ({ jobName, runDate }: { jobName: string; runDate: string }) => runJob(jobName, runDate),
    onSuccess: async (data) => {
      setLastResult({ jobName: data.job_name, status: data.status })
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['stocks'] }),
        queryClient.invalidateQueries({ queryKey: ['stock-daily'] }),
        queryClient.invalidateQueries({ queryKey: ['signals'] }),
      ])
    },
  })

  const busyJobName = useMemo(() => mutation.variables?.jobName ?? '', [mutation.variables])

  return (
    <section className="page">
      <div className="page-header">
        <div>
          <h3>任务中心</h3>
          <p>手动触发 V1 数据同步、因子计算和策略信号生成。</p>
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
          <strong>当前页面直接调用 `POST /api/jobs/:job_name/run`。</strong>
          <span>任务日志列表接口尚未进入 V1 已完成范围，所以这里先聚焦手动触发。</span>
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
    </section>
  )
}
