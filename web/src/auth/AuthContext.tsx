import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { get, onUnauthorized, post } from '../lib/api'
import type { User } from '../types'

interface AuthValue {
  user: User | null; loading: boolean
  login(username: string, password: string): Promise<void>
  register(username: string, password: string): Promise<void>
  logout(): Promise<void>
}
const AuthContext = createContext<AuthValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  useEffect(() => {
    onUnauthorized(() => setUser(null))
    get<User>('/auth/me').then(setUser).catch(() => setUser(null)).finally(() => setLoading(false))
  }, [])
  const value = useMemo<AuthValue>(() => ({
    user, loading,
    login: async (username, password) => setUser(await post<User>('/auth/login', { username, password })),
    register: async (username, password) => setUser(await post<User>('/auth/register', { username, password })),
    logout: async () => { await post('/auth/logout'); setUser(null) },
  }), [user, loading])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth() {
  const value = useContext(AuthContext)
  if (!value) throw new Error('useAuth must be used inside AuthProvider')
  return value
}
