import { type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { getMe, login as loginAPI, logout as logoutAPI } from '../lib/api'
import { privateQueryKeys } from '../lib/query'
import { AuthContext, type AuthContextValue } from './authContext'

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const meQuery = useQuery({
    queryKey: privateQueryKeys.me(),
    queryFn: getMe,
    retry: false,
  })

  const loginMutation = useMutation({
    mutationFn: loginAPI,
    onSuccess: async (currentUser) => {
      // 登录成功后先把当前用户写入缓存，再统一刷新私有查询，避免页面先闪未登录态。
      queryClient.setQueryData(privateQueryKeys.me(), currentUser)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: privateQueryKeys.auth() }),
        queryClient.invalidateQueries({ queryKey: privateQueryKeys.watchlists() }),
        queryClient.invalidateQueries({ queryKey: privateQueryKeys.positions() }),
      ])
    },
  })

  const logoutMutation = useMutation({
    mutationFn: logoutAPI,
    onSettled: async () => {
      // 无论服务端是否已经清掉 session，前端都必须立即移除私有缓存，避免旧用户数据残留在内存里。
      await Promise.all([
        queryClient.cancelQueries({ queryKey: privateQueryKeys.root() }),
        queryClient.removeQueries({ queryKey: privateQueryKeys.root() }),
      ])
      queryClient.setQueryData(privateQueryKeys.me(), null)
    },
  })

  const value: AuthContextValue = {
    currentUser: meQuery.data ?? null,
    isAuthenticated: !!meQuery.data,
    isChecking: meQuery.isPending,
    isLoggingIn: loginMutation.isPending,
    isLoggingOut: logoutMutation.isPending,
    login: async (input) => loginMutation.mutateAsync(input),
    logout: async () => {
      await logoutMutation.mutateAsync()
    },
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
