export type ApiEnvelope<T> = {
  code: number
  errmsg: string
  toast: string
  data: T
}

export type ApiError = Error & {
  code: number
  errmsg: string
  toast: string
}

export type StockItem = {
  ts_code: string
  symbol: string
  name: string
  industry: string
  exchange: string
  is_active: boolean
}

export type DailyBarItem = {
  ts_code: string
  trade_date: string
  open: string
  high: string
  low: string
  close: string
  pct_chg: string
  vol: string
  amount: string
}

export type SignalItem = {
  strategy_code: string
  strategy_version: string
  ts_code: string
  trade_date: string
  signal_type: string
  signal_strength: string
  signal_level: string
  buy_price_ref: string
  stop_loss_ref: string
  take_profit_ref: string
  invalidation_condition: string
  reason: string
}

export type PageResponse<T> = {
  items: T[]
  page: number
  page_size: number
}

export type RunJobResponse = {
  job_name: string
  status: string
}

export type JobTask = {
  name: string
  title: string
  description: string
}

export const V1_TASKS: JobTask[] = [
  {
    name: 'sync_stock_basic',
    title: '同步股票主数据',
    description: '刷新股票代码、简称、行业和上市状态。',
  },
  {
    name: 'sync_trade_calendar',
    title: '同步交易日历',
    description: '更新交易所开市状态和前一交易日映射。',
  },
  {
    name: 'sync_daily_market',
    title: '同步日线行情',
    description: '导入指定交易日的日线、成交量和涨跌幅。',
  },
  {
    name: 'calc_daily_factor',
    title: '计算日频因子',
    description: '重算均线、MACD、RSI 和量价衍生因子。',
  },
  {
    name: 'generate_strategy_signals',
    title: '生成策略信号',
    description: '评估固定策略并产出确定性买卖点。',
  },
]

const apiBaseURL = resolveAPIBaseURL()

async function request<T>(input: string, init?: RequestInit): Promise<T> {
  const response = await fetch(buildAPIURL(input), {
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
    ...init,
  })

  if (!response.ok) {
    const networkError = new Error(`HTTP ${response.status}`) as ApiError
    networkError.code = response.status
    networkError.errmsg = response.statusText
    networkError.toast = '网络请求失败，请稍后重试'
    throw networkError
  }

  const body = (await response.json()) as ApiEnvelope<T>
  if (body.code !== 0) {
    const apiError = new Error(body.errmsg || body.toast || '请求失败') as ApiError
    apiError.code = body.code
    apiError.errmsg = body.errmsg
    apiError.toast = body.toast
    throw apiError
  }

  return body.data
}

function resolveAPIBaseURL() {
  const configuredBaseURL = import.meta.env.VITE_API_BASE_URL?.trim()
  if (configuredBaseURL) {
    return configuredBaseURL.replace(/\/+$/, '')
  }

  return ''
}

function buildAPIURL(path: string) {
  if (!apiBaseURL) {
    return path
  }

  return `${apiBaseURL}${path}`
}

export async function getStocks(keyword = '', page = 1, pageSize = 20) {
  const search = new URLSearchParams({
    keyword,
    page: String(page),
    page_size: String(pageSize),
  })
  return request<PageResponse<StockItem>>(`/api/stocks?${search.toString()}`)
}

export async function getStockDaily(tsCode: string, startDate: string, endDate: string) {
  const search = new URLSearchParams({
    start_date: startDate,
    end_date: endDate,
  })
  return request<DailyBarItem[]>(`/api/stocks/${tsCode}/daily?${search.toString()}`)
}

export async function getSignals(tradeDate: string, strategyCode = '', page = 1, pageSize = 20) {
  const search = new URLSearchParams({
    trade_date: tradeDate,
    page: String(page),
    page_size: String(pageSize),
  })
  if (strategyCode) {
    search.set('strategy_code', strategyCode)
  }
  return request<PageResponse<SignalItem>>(`/api/signals?${search.toString()}`)
}

export async function runJob(jobName: string, bizDate: string) {
  return request<RunJobResponse>(`/api/jobs/${jobName}/run`, {
    method: 'POST',
    body: JSON.stringify({ biz_date: bizDate }),
  })
}
