import { Navigate, Outlet, Route, Routes } from 'react-router-dom'
import { AppLayout } from './components/AppLayout'
import { useAuth } from './lib/auth-context'
import { CDNsPage } from './pages/CDNsPage'
import { DashboardPage } from './pages/DashboardPage'
import { LoginPage } from './pages/LoginPage'
import { UsersPage } from './pages/UsersPage'

function RequireAuth() {
  const { session } = useAuth()
  return session ? <Outlet /> : <Navigate to="/login" replace />
}

function LoginRoute() {
  const { session } = useAuth()
  return session ? <Navigate to="/" replace /> : <LoginPage />
}

export function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginRoute />} />
      <Route element={<RequireAuth />}>
        <Route element={<AppLayout />}>
          <Route index element={<DashboardPage />} />
          <Route path="cdns" element={<CDNsPage />} />
          <Route path="users" element={<UsersPage />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
