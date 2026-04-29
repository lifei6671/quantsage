import {createContext, useContext} from 'react'

import type {CurrentUser, LoginRequest} from '../lib/api'

export type AuthContextValue = {
  currentUser: CurrentUser | null
  isAuthenticated: boolean
  isChecking: boolean
  isLoggingIn: boolean
  isLoggingOut: boolean
  login: (input: LoginRequest) => Promise<CurrentUser>
  logout: () => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | null>(null)

export function useAuth() {
  const value = useContext(AuthContext)
  if (value == null) {
    throw new Error('useAuth must be used within AuthProvider')
  }

  return value
}
