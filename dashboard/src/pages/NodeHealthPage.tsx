import { useQuery } from '@tanstack/react-query'
import { Activity, AlertTriangle, Box, CheckCircle2, Clock3, Radio, RefreshCw, Search, Server, ServerOff, Wifi } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Button, EmptyState, inputClass, LoadingRows } from '../components/ui'
import { api } from '../lib/api'
import type { NodeHealth } from '../types'

const ONLINE_THRESHOLD_MS = 30_000

type NodeState = 'online' | 'degraded' | 'offline'

function getNodeState(node: NodeHealth): NodeState {
  const heartbeatAge = Date.now() - new Date(node.timestamp).getTime()
  if (!Number.isFinite(heartbeatAge) || heartbeatAge > ONLINE_THRESHOLD_MS) return 'offline'
  return ['ok', 'healthy', 'online'].includes(node.status.toLowerCase()) ? 'online' : 'degraded'
}

function lastSeen(timestamp: string) {
  const seconds = Math.max(0, Math.floor((Date.now() - new Date(timestamp).getTime()) / 1000))
  if (!Number.isFinite(seconds)) return 'Unknown'
  if (seconds < 10) return 'Just now'
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

const stateStyles: Record<NodeState, { label: string, badge: string, dot: string, panel: string }> = {
  online: { label: 'Online', badge: 'bg-emerald-50 text-emerald-700 ring-emerald-200', dot: 'bg-emerald-500', panel: 'from-emerald-500/10' },
  degraded: { label: 'Degraded', badge: 'bg-amber-50 text-amber-700 ring-amber-200', dot: 'bg-amber-500', panel: 'from-amber-500/10' },
  offline: { label: 'Offline', badge: 'bg-red-50 text-red-700 ring-red-200', dot: 'bg-red-500', panel: 'from-red-500/10' },
}

export function NodeHealthPage() {
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<'all' | NodeState>('all')
  const query = useQuery({
    queryKey: ['node-health'],
    queryFn: api.listNodeHealth,
    refetchInterval: 10_000,
  })
  const nodes = useMemo(() => query.data ?? [], [query.data])
  const counts = useMemo(() => nodes.reduce((result, node) => {
    result[getNodeState(node)] += 1
    return result
  }, { online: 0, degraded: 0, offline: 0 }), [nodes])
  const visible = useMemo(() => nodes.filter((node) => {
    const text = `${node.service} ${node.instance} ${node.version}`.toLowerCase()
    return text.includes(search.toLowerCase()) && (filter === 'all' || getNodeState(node) === filter)
  }), [filter, nodes, search])
  const updatedAt = query.dataUpdatedAt
    ? new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(query.dataUpdatedAt)
    : 'Waiting for data'

  const stats = [
    { label: 'Total nodes', value: nodes.length, detail: 'Reporting edge instances', icon: Server, style: 'bg-slate-100 text-slate-600' },
    { label: 'Online', value: counts.online, detail: 'Heartbeat within 30 seconds', icon: CheckCircle2, style: 'bg-emerald-50 text-emerald-600' },
    { label: 'Degraded', value: counts.degraded, detail: 'Reporting a warning state', icon: AlertTriangle, style: 'bg-amber-50 text-amber-600' },
    { label: 'Offline', value: counts.offline, detail: 'Heartbeat is stale or missing', icon: ServerOff, style: 'bg-red-50 text-red-600' },
  ]

  return (
    <div className="space-y-6">
      <section className="relative overflow-hidden rounded-3xl bg-slate-950 px-6 py-7 text-white shadow-xl shadow-slate-200 sm:px-8">
        <div className="absolute -right-16 -top-20 size-64 rounded-full bg-red-600/25 blur-3xl" />
        <div className="relative flex flex-col gap-5 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <div className="mb-3 inline-flex items-center gap-2 rounded-full border border-emerald-400/20 bg-emerald-400/10 px-3 py-1.5 text-xs font-semibold text-emerald-300"><span className="relative flex size-2"><span className="absolute inline-flex size-full animate-ping rounded-full bg-emerald-400 opacity-60" /><span className="relative inline-flex size-2 rounded-full bg-emerald-400" /></span>Live heartbeat feed</div>
            <h2 className="text-2xl font-black tracking-tight sm:text-3xl">Edge network pulse</h2>
            <p className="mt-2 max-w-xl text-sm leading-6 text-slate-400">Node status refreshes every 10 seconds from the control panel's latest recorded heartbeat.</p>
          </div>
          <div className="flex items-center gap-3 rounded-2xl border border-white/10 bg-white/5 px-4 py-3">
            <Radio size={18} className="text-red-500" />
            <div><div className="text-[10px] font-bold uppercase tracking-wider text-slate-500">Last refresh</div><div className="mt-0.5 text-sm font-semibold text-slate-200">{updatedAt}</div></div>
          </div>
        </div>
      </section>

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {stats.map(({ label, value, detail, icon: Icon, style }) => (
          <article key={label} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-100">
            <div className="flex items-start justify-between"><div><p className="text-sm font-semibold text-slate-500">{label}</p><p className="mt-3 text-3xl font-black tracking-tight text-slate-950">{value}</p></div><div className={`rounded-xl p-2.5 ${style}`}><Icon size={20} /></div></div>
            <p className="mt-3 text-xs text-slate-400">{detail}</p>
          </article>
        ))}
      </section>

      <div className="flex flex-col gap-3 rounded-2xl border border-slate-200 bg-white p-3 shadow-sm sm:flex-row">
        <div className="relative flex-1"><Search className="absolute left-3.5 top-3 text-slate-400" size={18} /><input className={`${inputClass} border-0 bg-slate-50 pl-10 focus:bg-white`} value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search service, instance, or version…" /></div>
        <select className={`${inputClass} sm:w-44`} value={filter} onChange={(event) => setFilter(event.target.value as typeof filter)}><option value="all">All states</option><option value="online">Online</option><option value="degraded">Degraded</option><option value="offline">Offline</option></select>
        <Button variant="secondary" onClick={() => query.refetch()} disabled={query.isFetching}><RefreshCw size={16} className={query.isFetching ? 'animate-spin' : ''} />Refresh</Button>
      </div>

      {query.isLoading ? <LoadingRows /> : query.isError ? (
        <EmptyState icon={<AlertTriangle />} title="Could not load node health" description="The control panel could not read the latest heartbeat records. Try refreshing this page." />
      ) : visible.length === 0 ? (
        <EmptyState icon={<Server />} title={nodes.length ? 'No matching nodes' : 'No nodes reporting yet'} description={nodes.length ? 'Try another search or health-state filter.' : 'Edge nodes will appear here after their first health message reaches the control panel.'} />
      ) : (
        <section className="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
          {visible.map((node) => {
            const state = getNodeState(node)
            const styles = stateStyles[state]
            return (
              <article key={node.id || `${node.service}-${node.instance}`} className={`relative overflow-hidden rounded-2xl border border-slate-200 bg-gradient-to-br ${styles.panel} via-white to-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md`}>
                <div className="flex items-start justify-between gap-4">
                  <div className="flex min-w-0 items-center gap-3"><div className="grid size-11 shrink-0 place-items-center rounded-2xl bg-slate-950 text-white"><Server size={20} /></div><div className="min-w-0"><h3 className="truncate font-bold text-slate-950">{node.instance}</h3><p className="mt-0.5 truncate text-xs font-medium text-slate-500">{node.service}</p></div></div>
                  <span className={`inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-[10px] font-bold uppercase ring-1 ring-inset ${styles.badge}`}><span className={`size-1.5 rounded-full ${styles.dot}`} />{styles.label}</span>
                </div>
                <div className="mt-6 grid grid-cols-2 gap-3">
                  <div className="rounded-xl border border-slate-100 bg-white/80 p-3"><div className="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-slate-400"><Clock3 size={12} />Last seen</div><div className="mt-2 text-sm font-bold text-slate-700">{lastSeen(node.timestamp)}</div></div>
                  <div className="rounded-xl border border-slate-100 bg-white/80 p-3"><div className="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-slate-400"><Box size={12} />Version</div><div className="mt-2 truncate text-sm font-bold text-slate-700">{node.version || 'Unknown'}</div></div>
                </div>
                <div className="mt-4 flex items-center justify-between border-t border-slate-100 pt-4"><span className="inline-flex items-center gap-1.5 text-xs font-medium text-slate-400"><Wifi size={13} />Raw status</span><code className="rounded-md bg-slate-100 px-2 py-1 text-[10px] font-bold text-slate-600">{node.status || 'unknown'}</code></div>
              </article>
            )
          })}
        </section>
      )}
    </div>
  )
}
