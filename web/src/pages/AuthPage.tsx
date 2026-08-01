import { useState, type FormEvent } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { APIError } from '../lib/api'

export function AuthPage({ mode }: { mode: 'login' | 'register' }) {
  const { user, login, register } = useAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
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
        <label>密码<input type="password" autoComplete={registering ? 'new-password' : 'current-password'} minLength={10} required value={password} onChange={e => setPassword(e.target.value)} /></label>
        {registering && <p className="hint">用户名为 3–32 位中英文、数字、下划线或短横线；密码至少 10 位，并包含字母和数字。</p>}
        <button className="button primary wide" disabled={busy}>{busy ? '请稍候…' : registering ? '注册并开始' : '登录'}</button>
      </form>
      <p>{registering ? '已有账户？' : '还没有账户？'} <Link to={registering ? '/login' : '/register'}>{registering ? '去登录' : '立即注册'}</Link></p>
    </section>
  </main>
}
