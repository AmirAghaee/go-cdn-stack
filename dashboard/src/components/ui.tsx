import { AlertCircle, CheckCircle2, X } from 'lucide-react'
import { useCallback, useMemo, useState, type ButtonHTMLAttributes, type ReactNode } from 'react'
import { ToastContext } from '../lib/toast-context'

export function Button({ className = '', variant = 'primary', ...props }: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'secondary' | 'danger' | 'ghost' }) {
  const variants = {
    primary: 'bg-red-600 text-white shadow-sm hover:bg-red-700 focus:ring-red-200',
    secondary: 'border border-slate-200 bg-white text-slate-700 hover:border-slate-300 hover:bg-slate-50 focus:ring-slate-200',
    danger: 'bg-red-50 text-red-700 hover:bg-red-100 focus:ring-red-200',
    ghost: 'text-slate-500 hover:bg-slate-100 hover:text-slate-800 focus:ring-slate-200',
  }
  return <button className={`inline-flex h-10 items-center justify-center gap-2 rounded-xl px-4 text-sm font-semibold transition focus:outline-none focus:ring-4 disabled:cursor-not-allowed disabled:opacity-50 ${variants[variant]} ${className}`} {...props} />
}

export function Field({ label, error, children }: { label: string, error?: string, children: ReactNode }) {
  return (
    <label className="block text-sm font-medium text-slate-700">
      <span>{label}</span>
      <span className="mt-2 block">{children}</span>
      {error && <span className="mt-1.5 block text-xs text-red-600">{error}</span>}
    </label>
  )
}

export const inputClass = 'h-11 w-full rounded-xl border border-slate-200 bg-white px-3.5 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-red-400 focus:ring-4 focus:ring-red-100'

export function EmptyState({ icon, title, description }: { icon: ReactNode, title: string, description: string }) {
  return (
    <div className="flex min-h-64 flex-col items-center justify-center rounded-2xl border border-dashed border-slate-300 bg-white p-8 text-center">
      <div className="mb-4 rounded-2xl bg-red-50 p-3 text-red-600">{icon}</div>
      <h3 className="font-semibold text-slate-900">{title}</h3>
      <p className="mt-1 max-w-md text-sm text-slate-500">{description}</p>
    </div>
  )
}

export function LoadingRows() {
  return <div className="space-y-3 rounded-2xl border border-slate-200 bg-white p-5">{[1, 2, 3, 4].map((item) => <div key={item} className="h-14 animate-pulse rounded-xl bg-slate-100" />)}</div>
}

export function Modal({ title, description, children, onClose }: { title: string, description?: string, children: ReactNode, onClose: () => void }) {
  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-slate-950/55 p-0 backdrop-blur-sm sm:items-center sm:p-6" role="dialog" aria-modal="true" aria-labelledby="modal-title">
      <div className="max-h-[92vh] w-full overflow-y-auto rounded-t-3xl bg-white shadow-2xl sm:max-w-lg sm:rounded-3xl">
        <div className="flex items-start justify-between border-b border-slate-100 px-6 py-5">
          <div>
            <h2 id="modal-title" className="text-lg font-bold text-slate-950">{title}</h2>
            {description && <p className="mt-1 text-sm text-slate-500">{description}</p>}
          </div>
          <button onClick={onClose} className="rounded-lg p-2 text-slate-400 hover:bg-slate-100 hover:text-slate-700" aria-label="Close dialog"><X size={19} /></button>
        </div>
        {children}
      </div>
    </div>
  )
}

type Toast = { id: number, message: string, tone: 'success' | 'error' }

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])
  const notify = useCallback((message: string, tone: Toast['tone'] = 'success') => {
    const id = Date.now()
    setToasts((current) => [...current, { id, message, tone }])
    window.setTimeout(() => setToasts((current) => current.filter((toast) => toast.id !== id)), 4200)
  }, [])
  const value = useMemo(() => ({ notify }), [notify])
  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="fixed bottom-5 right-5 z-[60] flex w-[calc(100%-2.5rem)] max-w-sm flex-col gap-2" aria-live="polite">
        {toasts.map((toast) => (
          <div key={toast.id} className={`flex items-center gap-3 rounded-2xl border px-4 py-3 text-sm font-medium shadow-lg ${toast.tone === 'success' ? 'border-emerald-200 bg-emerald-50 text-emerald-800' : 'border-red-200 bg-red-50 text-red-800'}`}>
            {toast.tone === 'success' ? <CheckCircle2 size={18} /> : <AlertCircle size={18} />}
            {toast.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}
