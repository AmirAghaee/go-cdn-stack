import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, Clock3, Edit3, Globe2, Plus, RefreshCw, Search, Trash2 } from 'lucide-react'
import { useMemo, useState, type FormEvent } from 'react'
import { Button, EmptyState, Field, inputClass, LoadingRows, Modal } from '../components/ui'
import { api } from '../lib/api'
import { useToast } from '../lib/toast-context'
import type { CDN, CDNInput } from '../types'

const emptyForm: CDNInput = { domain: '', origin: '', cache_ttl: 60, is_active: true }

function CDNForm({ cdn, busy, onSubmit, onClose }: { cdn?: CDN, busy: boolean, onSubmit: (input: CDNInput) => void, onClose: () => void }) {
  const [form, setForm] = useState<CDNInput>(cdn ? { domain: cdn.domain, origin: cdn.origin, cache_ttl: cdn.cache_ttl, is_active: cdn.is_active } : emptyForm)
  const [errors, setErrors] = useState<Record<string, string>>({})
  function submit(event: FormEvent) {
    event.preventDefault()
    const nextErrors: Record<string, string> = {}
    const domain = form.domain.trim().toLowerCase()
    const origin = form.origin.trim()
    if (!domain) nextErrors.domain = 'Domain is required.'
    try { const url = new URL(origin); if (!['http:', 'https:'].includes(url.protocol)) nextErrors.origin = 'Origin must use HTTP or HTTPS.' } catch { nextErrors.origin = 'Enter a valid origin URL.' }
    if (!Number.isInteger(form.cache_ttl) || form.cache_ttl < 0) nextErrors.cache_ttl = 'TTL must be a whole number of zero or greater.'
    setErrors(nextErrors)
    if (!Object.keys(nextErrors).length) onSubmit({ ...form, domain, origin })
  }
  return <form onSubmit={submit}><div className="space-y-5 p-6"><Field label="Domain" error={errors.domain}><input className={inputClass} value={form.domain} onChange={(event) => setForm({ ...form, domain: event.target.value })} placeholder="cdn.example.com" autoFocus /></Field><Field label="Origin URL" error={errors.origin}><input className={inputClass} value={form.origin} onChange={(event) => setForm({ ...form, origin: event.target.value })} placeholder="https://origin.example.com" /></Field><Field label="Cache TTL (seconds)" error={errors.cache_ttl}><input className={inputClass} type="number" min="0" step="1" value={form.cache_ttl} onChange={(event) => setForm({ ...form, cache_ttl: Number(event.target.value) })} /></Field><label className="flex items-center justify-between rounded-2xl border border-slate-200 bg-slate-50 p-4"><span><span className="block text-sm font-semibold text-slate-800">Active configuration</span><span className="mt-1 block text-xs text-slate-500">Allow edges to serve this domain.</span></span><input type="checkbox" checked={form.is_active} onChange={(event) => setForm({ ...form, is_active: event.target.checked })} className="size-5 accent-red-600" /></label></div><div className="flex justify-end gap-3 border-t border-slate-100 px-6 py-4"><Button type="button" variant="secondary" onClick={onClose}>Cancel</Button><Button disabled={busy}>{busy ? 'Saving…' : cdn ? 'Save changes' : 'Create CDN'}</Button></div></form>
}

export function CDNsPage() {
  const queryClient = useQueryClient()
  const { notify } = useToast()
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState<'all' | 'active' | 'inactive'>('all')
  const [editing, setEditing] = useState<CDN | 'new' | null>(null)
  const [deleting, setDeleting] = useState<CDN | null>(null)
  const query = useQuery({ queryKey: ['cdns'], queryFn: api.listCDNs })
  const cdns = useMemo(() => query.data ?? [], [query.data])
  const visible = useMemo(() => cdns.filter((cdn) => {
    const matchesSearch = `${cdn.domain} ${cdn.origin}`.toLowerCase().includes(search.toLowerCase())
    const matchesStatus = status === 'all' || (status === 'active' ? cdn.is_active : !cdn.is_active)
    return matchesSearch && matchesStatus
  }), [cdns, search, status])
  const save = useMutation({
    mutationFn: ({ input, cdn }: { input: CDNInput, cdn?: CDN }) => cdn ? api.updateCDN(cdn.id, input) : api.createCDN(input),
    onSuccess: async (_, variables) => { await queryClient.invalidateQueries({ queryKey: ['cdns'] }); setEditing(null); notify(variables.cdn ? 'CDN configuration updated.' : 'CDN configuration created.') },
    onError: (error) => notify(error.message, 'error'),
  })
  const remove = useMutation({
    mutationFn: (cdn: CDN) => api.deleteCDN(cdn.id),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['cdns'] }); setDeleting(null); notify('CDN configuration deleted.') },
    onError: (error) => notify(error.message, 'error'),
  })

  return <div className="space-y-6"><div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between"><div><p className="max-w-2xl text-sm leading-6 text-slate-500">Manage customer domains, origins, cache lifetimes, and availability.</p></div><Button onClick={() => setEditing('new')}><Plus size={17} />Add CDN</Button></div><div className="flex flex-col gap-3 rounded-2xl border border-slate-200 bg-white p-3 shadow-sm sm:flex-row"><div className="relative flex-1"><Search className="absolute left-3.5 top-3 text-slate-400" size={18} /><input className={`${inputClass} border-0 bg-slate-50 pl-10 focus:bg-white`} value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search domain or origin…" /></div><select className={`${inputClass} sm:w-44`} value={status} onChange={(event) => setStatus(event.target.value as typeof status)}><option value="all">All statuses</option><option value="active">Active</option><option value="inactive">Inactive</option></select><Button variant="secondary" onClick={() => query.refetch()} disabled={query.isFetching}><RefreshCw size={16} className={query.isFetching ? 'animate-spin' : ''} />Refresh</Button></div>
    {query.isLoading ? <LoadingRows /> : query.isError ? <EmptyState icon={<AlertTriangle />} title="Could not load configurations" description="Check the control-panel connection and try refreshing this page." /> : visible.length === 0 ? <EmptyState icon={<Globe2 />} title={cdns.length ? 'No matching configurations' : 'No CDN configurations'} description={cdns.length ? 'Try another search or status filter.' : 'Create the first CDN configuration to begin serving a domain.'} /> : <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm"><div className="overflow-x-auto"><table className="w-full min-w-[760px] text-left"><thead className="border-b border-slate-200 bg-slate-50/80 text-[11px] font-bold uppercase tracking-wider text-slate-500"><tr><th className="px-5 py-3.5">Domain</th><th className="px-5 py-3.5">Origin</th><th className="px-5 py-3.5">Cache TTL</th><th className="px-5 py-3.5">Status</th><th className="px-5 py-3.5 text-right">Actions</th></tr></thead><tbody className="divide-y divide-slate-100">{visible.map((cdn) => <tr key={cdn.id} className="group hover:bg-slate-50/70"><td className="px-5 py-4"><div className="flex items-center gap-3"><div className="grid size-9 place-items-center rounded-xl bg-red-50 text-red-600"><Globe2 size={17} /></div><span className="font-semibold text-slate-800">{cdn.domain}</span></div></td><td className="max-w-xs truncate px-5 py-4 text-sm text-slate-500">{cdn.origin}</td><td className="px-5 py-4"><span className="inline-flex items-center gap-1.5 text-sm text-slate-600"><Clock3 size={15} />{cdn.cache_ttl}s</span></td><td className="px-5 py-4"><span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-bold ${cdn.is_active ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-500'}`}><span className={`size-1.5 rounded-full ${cdn.is_active ? 'bg-emerald-500' : 'bg-slate-400'}`} />{cdn.is_active ? 'Active' : 'Inactive'}</span></td><td className="px-5 py-4"><div className="flex justify-end gap-1"><button onClick={() => setEditing(cdn)} className="rounded-lg p-2 text-slate-400 hover:bg-slate-100 hover:text-slate-700" aria-label={`Edit ${cdn.domain}`}><Edit3 size={17} /></button><button onClick={() => setDeleting(cdn)} className="rounded-lg p-2 text-slate-400 hover:bg-red-50 hover:text-red-600" aria-label={`Delete ${cdn.domain}`}><Trash2 size={17} /></button></div></td></tr>)}</tbody></table></div><div className="border-t border-slate-100 px-5 py-3 text-xs text-slate-400">Showing {visible.length} of {cdns.length} configurations</div></div>}
    {editing && <Modal title={editing === 'new' ? 'Create CDN configuration' : `Edit ${editing.domain}`} description="Changes are published to edge nodes after persistence." onClose={() => setEditing(null)}><CDNForm cdn={editing === 'new' ? undefined : editing} busy={save.isPending} onClose={() => setEditing(null)} onSubmit={(input) => save.mutate({ input, cdn: editing === 'new' ? undefined : editing })} /></Modal>}
    {deleting && <Modal title="Delete CDN configuration?" description="This action cannot be undone." onClose={() => setDeleting(null)}><div className="p-6"><div className="rounded-2xl border border-red-100 bg-red-50 p-4 text-sm text-red-800"><strong>{deleting.domain}</strong> will be removed from the control plane and edge nodes will be notified.</div></div><div className="flex justify-end gap-3 border-t border-slate-100 px-6 py-4"><Button variant="secondary" onClick={() => setDeleting(null)}>Cancel</Button><Button variant="danger" disabled={remove.isPending} onClick={() => remove.mutate(deleting)}>{remove.isPending ? 'Deleting…' : 'Delete configuration'}</Button></div></Modal>}
  </div>
}
