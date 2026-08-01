import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { useAuth } from './auth/AuthContext'
import { AppShell } from './components/AppShell'
import { AuthPage } from './pages/AuthPage'
import { PlansPage } from './pages/PlansPage'
import { PlanPage } from './pages/PlanPage'

function Protected() {
  const { user, loading } = useAuth(); const location = useLocation()
  if (loading) return <div className="center" role="status">正在恢复登录状态…</div>
  return user ? <AppShell /> : <Navigate to="/login" replace state={{ from: location.pathname }} />
}

export default function App() {
  return <Routes>
    <Route path="/login" element={<AuthPage mode="login" />} />
    <Route path="/register" element={<AuthPage mode="register" />} />
    <Route element={<Protected />}>
      <Route path="/plans" element={<PlansPage />} />
      <Route path="/plans/:planId/*" element={<PlanPage />} />
    </Route>
    <Route path="*" element={<Navigate to="/plans" replace />} />
  </Routes>
}
