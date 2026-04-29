import {QueryClient} from '@tanstack/react-query'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 30_000,
    },
    mutations: {
      retry: 0,
    },
  },
})

// sharedQueryKeys 统一管理不依赖登录态的公共缓存键，避免后续清理私有缓存时误删共享数据。
export const sharedQueryKeys = {
  root: () => ['shared'] as const,
  stocksRoot: () => ['shared', 'stocks'] as const,
  stockDailyRoot: () => ['shared', 'stock-daily'] as const,
  signalsRoot: () => ['shared', 'signals'] as const,
  jobsRoot: () => ['shared', 'jobs'] as const,
  stocks: (keyword = '') => ['shared', 'stocks', keyword] as const,
  stockDaily: (tsCode: string, startDate: string, endDate: string) => ['shared', 'stock-daily', tsCode, startDate, endDate] as const,
  signals: (tradeDate: string, strategyCode = '') => ['shared', 'signals', tradeDate, strategyCode] as const,
  jobs: (jobName = '', bizDate = '') => ['shared', 'jobs', jobName, bizDate] as const,
}

// privateQueryKeys 统一管理登录用户私有缓存键，登录/登出时只需要按这个前缀批量刷新或清理。
export const privateQueryKeys = {
  root: () => ['private'] as const,
  auth: () => ['private', 'auth'] as const,
  me: () => ['private', 'auth', 'me'] as const,
  watchlists: () => ['private', 'watchlists'] as const,
  watchlistItems: (groupID: number | null) => ['private', 'watchlists', 'items', groupID ?? 'none'] as const,
  positions: () => ['private', 'positions'] as const,
}
