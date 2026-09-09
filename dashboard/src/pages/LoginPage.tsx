import { AlertCircle, ArrowRight, Eye, EyeOff, Globe2, LockKeyhole, Mail, ShieldCheck, Zap } from 'lucide-react'
import { useEffect, useState, type FormEvent } from 'react'
import { useAuth } from '../lib/auth-context'
import { Button, inputClass } from '../components/ui'

export function LoginPage() {
  const { login } = useAuth()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (sessionStorage.getItem('cdneto.auth.notice') === 'expired') {
      setNotice('Your session expired. Sign in again to continue.')
      sessionStorage.removeItem('cdneto.auth.notice')
    }
  }, [])

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      await login(email.trim(), password)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Unable to sign in.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <main className="grid min-h-screen bg-slate-950 lg:grid-cols-[1.05fr_0.95fr]">
      <section className="relative hidden overflow-hidden p-14 lg:flex lg:flex-col lg:justify-between">
        <div className="absolute -left-36 top-1/3 size-96 rounded-full bg-red-600/20 blur-3xl" />
        <div className="absolute right-0 top-0 h-full w-px bg-gradient-to-b from-transparent via-red-500/40 to-transparent" />
        <div className="relative flex items-center gap-3 text-white">
          <div className="grid size-11 place-items-center rounded-2xl bg-red-600 shadow-xl shadow-red-950"><Zap fill="currentColor" /></div>
          <div><div className="text-xl font-black tracking-tight">CDNeto</div><div className="text-[10px] font-semibold uppercase tracking-[0.24em] text-slate-500">Control plane</div></div>
        </div>
        <div className="relative max-w-xl">
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-red-500/20 bg-red-500/10 px-3 py-1.5 text-xs font-semibold text-red-300"><ShieldCheck size={14} />Secure administration</div>
          <h1 className="text-5xl font-black leading-[1.08] tracking-[-0.04em] text-white xl:text-6xl">Control your edge.<br /><span className="text-red-500">Shape the internet.</span></h1>
          <p className="mt-6 max-w-lg text-base leading-7 text-slate-400">Manage domains, origins, cache behavior, and team access from one focused command center.</p>
          <div className="mt-10 grid grid-cols-3 gap-3">
            {['Fast updates', 'Secure access', 'Clear status'].map((item, index) => <div key={item} className="rounded-2xl border border-white/10 bg-white/[0.03] p-4"><div className="mb-3 text-xl font-black text-red-500">0{index + 1}</div><div className="text-xs font-semibold text-slate-300">{item}</div></div>)}
          </div>
        </div>
        <p className="relative text-xs text-slate-600">CDNeto infrastructure console</p>
      </section>

      <section className="flex items-center justify-center bg-[#f7f7f8] px-5 py-12 sm:px-10">
        <div className="w-full max-w-md">
          <div className="mb-10 flex items-center gap-3 lg:hidden">
            <div className="grid size-10 place-items-center rounded-xl bg-red-600 text-white"><Zap size={20} fill="currentColor" /></div><span className="text-xl font-black">CDNeto</span>
          </div>
          <div className="mb-8"><div className="mb-4 grid size-12 place-items-center rounded-2xl bg-red-50 text-red-600"><Globe2 /></div><h2 className="text-3xl font-black tracking-tight text-slate-950">Welcome back</h2><p className="mt-2 text-sm leading-6 text-slate-500">Use your control-panel administrator credentials.</p></div>
          {notice && <div className="mb-5 flex items-start gap-3 rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800"><AlertCircle className="mt-0.5 shrink-0" size={17} />{notice}</div>}
          {error && <div className="mb-5 flex items-start gap-3 rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-700" role="alert"><AlertCircle className="mt-0.5 shrink-0" size={17} />{error}</div>}
          <form className="space-y-5" onSubmit={handleSubmit}>
            <label className="block text-sm font-semibold text-slate-700">Email address<div className="relative mt-2"><Mail className="absolute left-3.5 top-3.5 text-slate-400" size={17} /><input className={`${inputClass} pl-10`} type="email" autoComplete="email" required value={email} onChange={(event) => setEmail(event.target.value)} placeholder="admin@example.com" /></div></label>
            <label className="block text-sm font-semibold text-slate-700">Password<div className="relative mt-2"><LockKeyhole className="absolute left-3.5 top-3.5 text-slate-400" size={17} /><input className={`${inputClass} px-10`} type={showPassword ? 'text' : 'password'} autoComplete="current-password" required value={password} onChange={(event) => setPassword(event.target.value)} placeholder="Enter your password" /><button type="button" onClick={() => setShowPassword((value) => !value)} className="absolute right-3 top-3 rounded p-1 text-slate-400 hover:text-slate-700" aria-label={showPassword ? 'Hide password' : 'Show password'}>{showPassword ? <EyeOff size={17} /> : <Eye size={17} />}</button></div></label>
            <Button className="mt-2 w-full" disabled={submitting}>{submitting ? 'Signing in…' : <>Sign in <ArrowRight size={17} /></>}</Button>
          </form>
          <p className="mt-8 text-center text-xs leading-5 text-slate-400">Access is limited to authorized control-plane administrators.</p>
        </div>
      </section>
    </main>
  )
}
