import { useMutation, useQuery } from '@tanstack/react-query'
import { Activity, ArrowRight, Globe2, Radio, RefreshCw, ServerCog, Users } from 'lucide-react'
import { Link } from 'react-router-dom'
import { Button, LoadingRows } from '../components/ui'
import { api } from '../lib/api'
import { useToast } from '../lib/toast-context'

export function DashboardPage() {
  const cdnsQuery = useQuery({ queryKey: ['cdns'], queryFn: api.listCDNs })
  const usersQuery = useQuery({ queryKey: ['users'], queryFn: api.listUsers })
  const { notify } = useToast()
  const snapshot = useMutation({
    mutationFn: api.refreshSnapshot,
    onSuccess: (result) => notify(result.message || 'Snapshot refresh triggered.'),
    onError: (error) => notify(error.message, 'error'),
  })
  const cdns = cdnsQuery.data ?? []
  const users = usersQuery.data ?? []
  const active = cdns.filter((cdn) => cdn.is_active).length
  const stats = [
    { label: 'Total CDNs', value: cdns.length, detail: 'Configured domains', icon: Globe2, color: 'text-red-600 bg-red-50' },
    { label: 'Active', value: active, detail: 'Serving configurations', icon: Activity, color: 'text-emerald-600 bg-emerald-50' },
    { label: 'Inactive', value: cdns.length - active, detail: 'Paused configurations', icon: Radio, color: 'text-amber-600 bg-amber-50' },
    { label: 'Administrators', value: users.length, detail: 'Control-panel users', icon: Users, color: 'text-sky-600 bg-sky-50' },
  ]

  if (cdnsQuery.isLoading || usersQuery.isLoading) return <LoadingRows />

  return (
    <div className="space-y-7">
      {(cdnsQuery.isError || usersQuery.isError) && <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-sm font-medium text-red-700">Some overview data could not be loaded. Refresh the page to try again.</div>}
      <section className="relative overflow-hidden rounded-3xl bg-slate-950 px-6 py-8 text-white shadow-xl shadow-slate-200 sm:px-8">
        <div className="absolute -right-16 -top-20 size-72 rounded-full bg-red-600/20 blur-3xl" />
        <div className="relative flex flex-col gap-7 sm:flex-row sm:items-center sm:justify-between">
          <div><div className="mb-3 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-1.5 text-xs font-semibold text-slate-300"><ServerCog size={14} className="text-red-500" />Configuration network</div><h2 className="text-2xl font-black tracking-tight sm:text-3xl">Keep every edge in sync.</h2><p className="mt-2 max-w-xl text-sm leading-6 text-slate-400">Publish the latest control-plane configuration when an edge needs an immediate refresh.</p></div>
          <Button className="shrink-0" disabled={snapshot.isPending} onClick={() => snapshot.mutate()}><RefreshCw size={17} className={snapshot.isPending ? 'animate-spin' : ''} />{snapshot.isPending ? 'Triggering…' : 'Refresh snapshot'}</Button>
        </div>
      </section>
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {stats.map(({ label, value, detail, icon: Icon, color }) => <article key={label} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-100"><div className="flex items-start justify-between"><div><p className="text-sm font-semibold text-slate-500">{label}</p><p className="mt-3 text-3xl font-black tracking-tight text-slate-950">{value}</p></div><div className={`rounded-xl p-2.5 ${color}`}><Icon size={20} /></div></div><p className="mt-3 text-xs text-slate-400">{detail}</p></article>)}
      </section>
      <section className="grid gap-5 xl:grid-cols-[1.45fr_0.55fr]">
        <article className="rounded-2xl border border-slate-200 bg-white shadow-sm"><div className="flex items-center justify-between border-b border-slate-100 px-5 py-4"><div><h3 className="font-bold text-slate-950">Recent configurations</h3><p className="mt-0.5 text-xs text-slate-500">Latest CDN entries in the control plane</p></div><Link to="/cdns" className="flex items-center gap-1 text-xs font-bold text-red-600 hover:text-red-700">View all <ArrowRight size={14} /></Link></div><div className="divide-y divide-slate-100">{cdns.slice(0, 5).map((cdn) => <div key={cdn.id} className="flex items-center gap-3 px-5 py-4"><div className="grid size-9 place-items-center rounded-xl bg-slate-100 text-slate-500"><Globe2 size={17} /></div><div className="min-w-0 flex-1"><div className="truncate text-sm font-semibold text-slate-800">{cdn.domain}</div><div className="truncate text-xs text-slate-400">{cdn.origin}</div></div><span className={`rounded-full px-2 py-1 text-[10px] font-bold uppercase ${cdn.is_active ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-500'}`}>{cdn.is_active ? 'Active' : 'Inactive'}</span></div>)}{cdns.length === 0 && <p className="p-8 text-center text-sm text-slate-400">No CDN configurations yet.</p>}</div></article>
        <article className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"><h3 className="font-bold text-slate-950">Configuration health</h3><p className="mt-1 text-xs text-slate-500">Based on control-plane records</p><div className="mt-6 flex items-center gap-5"><div className="grid size-20 place-items-center rounded-full border-[7px] border-red-100 text-xl font-black text-red-600">{cdns.length ? Math.round((active / cdns.length) * 100) : 0}%</div><div><p className="text-sm font-semibold text-slate-700">Active ratio</p><p className="mt-1 text-xs leading-5 text-slate-400">{active} of {cdns.length} configurations are enabled.</p></div></div></article>
      </section>
    </div>
  )
}
