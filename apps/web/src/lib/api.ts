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

export type JobRunItem = {
  id: number
  job_name: string
  biz_date: string
  status: string
  started_at: string
  finished_at: string
  error_code: number
  error_message: string
}

export type CurrentUser = {
  id: number
  username: string
  display_name: string
  status: string
  role: string
  last_login_at?: string
}

export type LoginRequest = {
  username: string
  password: string
}

export type WatchlistGroupItem = {
  id: number
  name: string
  sort_order: number
  created_at: string
  updated_at: string
}

export type CreateWatchlistGroupRequest = {
  name: string
  sort_order: number
}

export type UpdateWatchlistGroupRequest = CreateWatchlistGroupRequest

export type WatchlistItem = {
  id: number
  group_id: number
  ts_code: string
  note: string
  created_at: string
}

export type CreateWatchlistItemRequest = {
  ts_code: string
  note: string
}

export type PositionItem = {
  id: number
  ts_code: string
  position_date: string
  quantity: string
  cost_price: string
  note: string
  created_at: string
  updated_at: string
}

export type CreatePositionRequest = {
  ts_code: string
  position_date: string
  quantity: string
  cost_price: string
  note: string
}

export type UpdatePositionRequest = CreatePositionRequest

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
    // 登录态依赖 HttpOnly session cookie，这里统一允许浏览器在同源和显式跨域基地址下都带上凭据。
    credentials: 'include',
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

export async function login(input: LoginRequest) {
  return request<CurrentUser>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function logout() {
  return request<{ status: string }>('/api/auth/logout', {
    method: 'POST',
  })
}

export async function getMe() {
  try {
    return await request<CurrentUser>('/api/auth/me')
  } catch (error) {
    const apiError = error as Partial<ApiError>
    if (apiError.code === 401) {
      return null
    }
    throw error
  }
}

export async function runJob(jobName: string, bizDate: string) {
  return request<RunJobResponse>(`/api/jobs/${jobName}/run`, {
    method: 'POST',
    body: JSON.stringify({ biz_date: bizDate }),
  })
}

export async function getJobRuns(jobName = '', bizDate = '', page = 1, pageSize = 20) {
  const search = new URLSearchParams({
    page: String(page),
    page_size: String(pageSize),
  })
  if (jobName) {
    search.set('job_name', jobName)
  }
  if (bizDate) {
    search.set('biz_date', bizDate)
  }
  return request<PageResponse<JobRunItem>>(`/api/jobs?${search.toString()}`)
}

export async function getWatchlistGroups() {
  return request<WatchlistGroupItem[]>('/api/watchlists')
}

export async function createWatchlistGroup(input: CreateWatchlistGroupRequest) {
  return request<WatchlistGroupItem>('/api/watchlists', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function updateWatchlistGroup(groupID: number, input: UpdateWatchlistGroupRequest) {
  return request<WatchlistGroupItem>(`/api/watchlists/${groupID}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export async function deleteWatchlistGroup(groupID: number) {
  return request<{ deleted: boolean }>(`/api/watchlists/${groupID}`, {
    method: 'DELETE',
  })
}

export async function getWatchlistItems(groupID: number) {
  return request<WatchlistItem[]>(`/api/watchlists/${groupID}/items`)
}

export async function createWatchlistItem(groupID: number, input: CreateWatchlistItemRequest) {
  return request<WatchlistItem>(`/api/watchlists/${groupID}/items`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function deleteWatchlistItem(groupID: number, itemID: number) {
  return request<{ deleted: boolean }>(`/api/watchlists/${groupID}/items/${itemID}`, {
    method: 'DELETE',
  })
}

export async function getPositions() {
  return request<PositionItem[]>('/api/positions')
}

export async function createPosition(input: CreatePositionRequest) {
  return request<PositionItem>('/api/positions', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function updatePosition(positionID: number, input: UpdatePositionRequest) {
  return request<PositionItem>(`/api/positions/${positionID}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export async function deletePosition(positionID: number) {
  return request<{ deleted: boolean }>(`/api/positions/${positionID}`, {
    method: 'DELETE',
  })
}
