import { Activity, Globe2, LayoutDashboard, LogOut, Menu, ShieldCheck, Users, X, Zap } from 'lucide-react'
import { useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../lib/auth-context'

const navigation = [
  { to: '/', label: 'Overview', icon: LayoutDashboard },
  { to: '/cdns', label: 'CDN Management', icon: Globe2 },
  { to: '/health', label: 'Node Health', icon: Activity },
  { to: '/users', label: 'Users', icon: Users },
]

const titles: Record<string, { title: string, eyebrow: string }> = {
  '/': { title: 'Network overview', eyebrow: 'Command center' },
  '/cdns': { title: 'CDN configurations', eyebrow: 'Traffic delivery' },
  '/health': { title: 'Node health', eyebrow: 'Live infrastructure' },
  '/users': { title: 'Team access', eyebrow: 'Administration' },
}

function Sidebar({ onNavigate }: { onNavigate?: () => void }) {
  const { session, logout } = useAuth()
  return (
    <div className="flex h-full flex-col bg-slate-950 px-4 py-5 text-white">
      <div className="flex items-center gap-3 px-2">
        <div className="grid size-10 place-items-center rounded-xl bg-red-600 shadow-lg shadow-red-950/40"><Zap size={21} fill="currentColor" /></div>
        <div><div className="text-lg font-black tracking-tight">CDNeto</div><div className="text-[10px] font-semibold uppercase tracking-[0.2em] text-slate-500">Control plane</div></div>
      </div>
      <nav className="mt-10 flex-1 space-y-1.5">
        {navigation.map(({ to, label, icon: Icon }) => (
          <NavLink key={to} to={to} end={to === '/'} onClick={onNavigate} className={({ isActive }) => `flex items-center gap-3 rounded-xl px-3 py-3 text-sm font-semibold transition ${isActive ? 'bg-red-600 text-white shadow-lg shadow-red-950/30' : 'text-slate-400 hover:bg-white/5 hover:text-white'}`}>
            <Icon size={18} />{label}
          </NavLink>
        ))}
      </nav>
      <div className="rounded-2xl border border-white/10 bg-white/5 p-3">
        <div className="flex items-center gap-3">
          <div className="grid size-9 shrink-0 place-items-center rounded-full bg-red-500/15 text-sm font-bold text-red-400">{session?.user.email.charAt(0).toUpperCase()}</div>
          <div className="min-w-0 flex-1"><div className="truncate text-xs font-semibold text-slate-200">{session?.user.email}</div><div className="mt-0.5 text-[11px] text-slate-500">Administrator</div></div>
          <button onClick={() => logout()} className="rounded-lg p-2 text-slate-500 hover:bg-white/5 hover:text-white" aria-label="Log out"><LogOut size={17} /></button>
        </div>
      </div>
    </div>
  )
}

export function AppLayout() {
  const [menuOpen, setMenuOpen] = useState(false)
  const location = useLocation()
  const heading = titles[location.pathname] ?? titles['/']
  return (
    <div className="min-h-screen bg-[#f7f7f8] lg:grid lg:grid-cols-[268px_1fr]">
      <aside className="fixed inset-y-0 left-0 z-40 hidden w-[268px] lg:block"><Sidebar /></aside>
      {menuOpen && <div className="fixed inset-0 z-50 lg:hidden"><button className="absolute inset-0 bg-slate-950/60" onClick={() => setMenuOpen(false)} aria-label="Close menu" /><aside className="relative h-full w-[286px] shadow-2xl"><button className="absolute right-3 top-3 z-10 rounded-lg p-2 text-slate-400 hover:bg-white/10" onClick={() => setMenuOpen(false)}><X size={19} /></button><Sidebar onNavigate={() => setMenuOpen(false)} /></aside></div>}
      <main className="min-w-0 lg:col-start-2">
        <header className="sticky top-0 z-30 border-b border-slate-200/80 bg-[#f7f7f8]/90 px-4 py-4 backdrop-blur-xl sm:px-7 lg:px-10">
          <div className="mx-auto flex max-w-7xl items-center gap-4">
            <button className="rounded-xl border border-slate-200 bg-white p-2.5 text-slate-600 lg:hidden" onClick={() => setMenuOpen(true)} aria-label="Open menu"><Menu size={20} /></button>
            <div className="flex-1"><p className="text-[10px] font-bold uppercase tracking-[0.22em] text-red-600">{heading.eyebrow}</p><h1 className="mt-0.5 text-xl font-black tracking-tight text-slate-950 sm:text-2xl">{heading.title}</h1></div>
            <div className="hidden items-center gap-2 rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1.5 text-xs font-semibold text-emerald-700 sm:flex"><ShieldCheck size={14} />Authenticated session</div>
          </div>
        </header>
        <div className="mx-auto max-w-7xl p-4 sm:p-7 lg:p-10"><Outlet /></div>
      </main>
    </div>
  )
}
