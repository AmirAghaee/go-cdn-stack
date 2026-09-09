import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { App } from './App'
import { ToastProvider } from './components/ui'
import { AuthProvider } from './lib/AuthProvider'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 20_000, retry: 1, refetchOnWindowFocus: false } },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <QueryClientProvider client={queryClient}>
        <AuthProvider><ToastProvider><App /></ToastProvider></AuthProvider>
      </QueryClientProvider>
    </BrowserRouter>
  </StrictMode>,
)
