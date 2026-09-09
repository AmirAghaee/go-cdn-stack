import { createContext, useContext } from 'react'
import type { User } from '../types'

export interface AuthState {
  token: string
  user: User
}

export interface AuthContextValue {
  session: AuthState | null
  login: (email: string, password: string) => Promise<void>
  logout: (reason?: 'expired') => void
}

export const AuthContext = createContext<AuthContextValue | null>(null)

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) throw new Error('useAuth must be used within AuthProvider')
  return context
}
