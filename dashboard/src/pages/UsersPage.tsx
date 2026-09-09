import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, CalendarDays, Mail, Plus, Search, ShieldCheck, UserRound, Users } from 'lucide-react'
import { useMemo, useState, type FormEvent } from 'react'
import { Button, EmptyState, Field, inputClass, LoadingRows, Modal } from '../components/ui'
import { api } from '../lib/api'
import { useToast } from '../lib/toast-context'

export function UsersPage() {
  const queryClient = useQueryClient()
  const { notify } = useToast()
  const [search, setSearch] = useState('')
  const [creating, setCreating] = useState(false)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmation, setConfirmation] = useState('')
  const [error, setError] = useState('')
  const query = useQuery({ queryKey: ['users'], queryFn: api.listUsers })
  const users = useMemo(() => query.data ?? [], [query.data])
  const visible = useMemo(() => users.filter((user) => user.email.toLowerCase().includes(search.toLowerCase())), [search, users])
  const create = useMutation({
    mutationFn: () => api.createUser(email.trim(), password),
    onSuccess: async (result) => { await queryClient.invalidateQueries({ queryKey: ['users'] }); closeForm(); notify(result.message || 'Administrator created.') },
    onError: (reason) => setError(reason.message),
  })
  function closeForm() { setCreating(false); setEmail(''); setPassword(''); setConfirmation(''); setError('') }
  function submit(event: FormEvent) { event.preventDefault(); setError(''); if (password !== confirmation) { setError('Passwords do not match.'); return } create.mutate() }
  const date = (value: string) => value ? new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(value)) : 'Unknown'

  return <div className="space-y-6"><div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between"><p className="max-w-2xl text-sm leading-6 text-slate-500">View administrators and grant access to the control plane.</p><Button onClick={() => setCreating(true)}><Plus size={17} />Add administrator</Button></div><div className="relative rounded-2xl border border-slate-200 bg-white p-3 shadow-sm"><Search className="absolute left-6 top-6 text-slate-400" size={18} /><input className={`${inputClass} border-0 bg-slate-50 pl-10 focus:bg-white`} value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search by email address…" /></div>
    {query.isLoading ? <LoadingRows /> : query.isError ? <EmptyState icon={<AlertTriangle />} title="Could not load administrators" description="Check the control-panel connection and refresh this page." /> : visible.length === 0 ? <EmptyState icon={<Users />} title={users.length ? 'No matching administrators' : 'No administrators found'} description={users.length ? 'Try a different email address.' : 'Create an administrator to grant control-panel access.'} /> : <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">{visible.map((user) => <article key={user.id} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md"><div className="flex items-start justify-between"><div className="grid size-11 place-items-center rounded-2xl bg-red-50 text-red-600"><UserRound size={21} /></div><span className="inline-flex items-center gap-1.5 rounded-full bg-slate-100 px-2.5 py-1 text-[10px] font-bold uppercase tracking-wide text-slate-600"><ShieldCheck size={12} />Admin</span></div><div className="mt-5 flex items-center gap-2 text-sm font-semibold text-slate-800"><Mail size={15} className="text-slate-400" /><span className="truncate">{user.email}</span></div><div className="mt-3 flex items-center gap-2 text-xs text-slate-400"><CalendarDays size={14} />Added {date(user.created_at)}</div><div className="mt-5 truncate border-t border-slate-100 pt-4 font-mono text-[10px] text-slate-400">ID: {user.id}</div></article>)}</div>}
    {creating && <Modal title="Add administrator" description="Create credentials for a new control-panel user." onClose={closeForm}><form onSubmit={submit}><div className="space-y-5 p-6">{error && <div className="rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-700" role="alert">{error}</div>}<Field label="Email address"><input className={inputClass} type="email" required autoFocus autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="operator@example.com" /></Field><Field label="Password"><input className={inputClass} type="password" required autoComplete="new-password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="Enter a secure password" /></Field><Field label="Confirm password"><input className={inputClass} type="password" required autoComplete="new-password" value={confirmation} onChange={(event) => setConfirmation(event.target.value)} placeholder="Repeat the password" /></Field></div><div className="flex justify-end gap-3 border-t border-slate-100 px-6 py-4"><Button type="button" variant="secondary" onClick={closeForm}>Cancel</Button><Button disabled={create.isPending}>{create.isPending ? 'Creating…' : 'Create administrator'}</Button></div></form></Modal>}
  </div>
}
