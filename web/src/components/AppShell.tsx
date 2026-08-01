import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'

export function AppShell() {
  const { user, logout } = useAuth()
  return <div className="app-shell">
    <header className="topbar">
      <NavLink className="brand" to="/plans">定投计划</NavLink>
      <nav aria-label="主导航"><NavLink to="/plans">我的计划</NavLink></nav>
      <div className="account"><span>{user?.username}</span><button className="button ghost" onClick={() => void logout()}>退出</button></div>
    </header>
    <main className="page"><Outlet /></main>
  </div>
}
