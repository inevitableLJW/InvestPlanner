import { useState, type FormEvent } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { APIError } from '../lib/api'

export function AuthPage({ mode }: { mode: 'login' | 'register' }) {
  const { user, login, register } = useAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate(); const location = useLocation()
  if (user) return <Navigate to="/plans" replace />
  const submit = async (event: FormEvent) => {
    event.preventDefault(); setBusy(true); setError('')
    try {
      if (mode === 'register') await register(username, password); else await login(username, password)
      const target = (location.state as { from?: string } | null)?.from ?? '/plans'
      navigate(target, { replace: true })
    } catch (reason) {
      setError(reason instanceof APIError ? reason.message : '暂时无法连接服务，请稍后重试')
    } finally { setBusy(false) }
  }
  const registering = mode === 'register'
  return <main className="auth-page">
    <section className="auth-card">
      <p className="eyebrow">INVEST PLANNER</p>
      <h1>{registering ? '创建账户' : '欢迎回来'}</h1>
      <p className="muted">登录后按计划记录每月收支与实际投入。</p>
      {error && <div className="alert" role="alert">{error}</div>}
      <form onSubmit={submit} aria-label={registering ? '注册' : '登录'}>
        <label>用户名<input type="text" autoComplete="username" minLength={registering ? 3 : undefined} maxLength={registering ? 32 : undefined} required value={username} onChange={e => setUsername(e.target.value)} /></label>
        <div className="auth-field">
          <label htmlFor="auth-password">密码</label>
          <div className="password-field">
            <input id="auth-password" type={showPassword ? 'text' : 'password'} autoComplete={registering ? 'new-password' : 'current-password'} minLength={10} required value={password} onChange={e => setPassword(e.target.value)} />
            <button
              type="button"
              className="password-toggle"
              aria-label={showPassword ? '隐藏密码' : '显示密码'}
              aria-pressed={showPassword}
              onClick={() => setShowPassword(value => !value)}
            >
              {showPassword ? (
                <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
                  <path d="M3 3l18 18" />
                  <path d="M10.6 10.7a2 2 0 0 0 2.7 2.7" />
                  <path d="M9.9 4.2A10.7 10.7 0 0 1 12 4c5.5 0 9 5.5 9 8a8.8 8.8 0 0 1-2.2 3.5M6.6 6.6C4.3 8.1 3 10.5 3 12c0 2.5 3.5 8 9 8a10.5 10.5 0 0 0 4.1-.8" />
                </svg>
              ) : (
                <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
                  <path d="M3 12c0-2.5 3.5-8 9-8s9 5.5 9 8-3.5 8-9 8-9-5.5-9-8Z" />
                  <circle cx="12" cy="12" r="2.5" />
                </svg>
              )}
            </button>
          </div>
        </div>
        {registering && <p className="hint">用户名为 3–32 位中英文、数字、下划线或短横线；密码至少 10 位，并包含字母和数字。</p>}
        <button className="button primary wide" disabled={busy}>{busy ? '请稍候…' : registering ? '注册并开始' : '登录'}</button>
      </form>
      <p>{registering ? '已有账户？' : '还没有账户？'} <Link to={registering ? '/login' : '/register'}>{registering ? '去登录' : '立即注册'}</Link></p>
    </section>
  </main>
}
