import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api, configureAPI } from './api'
import { AuthContext, type AuthState } from './auth-context'

const SESSION_KEY = 'cdneto.admin.session'

function readSession(): AuthState | null {
  try {
    const value = sessionStorage.getItem(SESSION_KEY)
    if (!value) return null
    const parsed = JSON.parse(value) as AuthState
    return parsed.token && parsed.user?.email ? parsed : null
  } catch {
    sessionStorage.removeItem(SESSION_KEY)
    return null
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<AuthState | null>(readSession)

  const logout = useCallback((reason?: 'expired') => {
    sessionStorage.removeItem(SESSION_KEY)
    if (reason) sessionStorage.setItem('cdneto.auth.notice', reason)
    setSession(null)
  }, [])

  useEffect(() => {
    configureAPI(() => session?.token ?? null, () => logout('expired'))
  }, [logout, session?.token])

  const login = useCallback(async (email: string, password: string) => {
    const result = await api.login(email, password)
    const nextSession = { token: result.token, user: result.user }
    sessionStorage.setItem(SESSION_KEY, JSON.stringify(nextSession))
    setSession(nextSession)
  }, [])

  const value = useMemo(() => ({ session, login, logout }), [login, logout, session])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
