import type { CDN, CDNInput, LoginResponse, MessageResponse, NodeHealth, User } from '../types'

const API_BASE = '/control-api'

type UnauthorizedHandler = () => void

let getToken: () => string | null = () => null
let onUnauthorized: UnauthorizedHandler = () => undefined

export class APIError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'APIError'
    this.status = status
  }
}

export function configureAPI(tokenReader: () => string | null, unauthorizedHandler: UnauthorizedHandler) {
  getToken = tokenReader
  onUnauthorized = unauthorizedHandler
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  const token = getToken()

  headers.set('Accept', 'application/json')
  if (options.body) headers.set('Content-Type', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)

  let response: Response
  try {
    response = await fetch(`${API_BASE}${path}`, { ...options, headers })
  } catch {
    throw new APIError('Unable to reach the control panel.', 0)
  }

  if (response.status === 401 && token) onUnauthorized()

  if (!response.ok) {
    let serverMessage = ''
    try {
      const body = (await response.json()) as { error?: string }
      serverMessage = body.error ?? ''
    } catch {
      // The API may intentionally return an empty response body.
    }

    const message = response.status >= 500
      ? 'The control panel could not complete this request.'
      : serverMessage || `Request failed with status ${response.status}.`
    throw new APIError(message, response.status)
  }

  if (response.status === 204 || response.headers.get('content-length') === '0') {
    return undefined as T
  }

  const text = await response.text()
  return text ? JSON.parse(text) as T : undefined as T
}

export const api = {
  login: (email: string, password: string) => request<LoginResponse>('/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  }),
  listUsers: () => request<User[]>('/api/users'),
  createUser: (email: string, password: string) => request<MessageResponse>('/api/register', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  }),
  changeUserPassword: (id: string, password: string) => request<MessageResponse>(`/api/users/${id}/password`, {
    method: 'PUT',
    body: JSON.stringify({ password }),
  }),
  deleteUser: (id: string) => request<void>(`/api/users/${id}`, { method: 'DELETE' }),
  listCDNs: () => request<CDN[]>('/api/cdns'),
  getCDN: (id: string) => request<CDN>(`/api/cdns/${id}`),
  createCDN: (input: CDNInput) => request<void>('/api/cdns', {
    method: 'POST',
    body: JSON.stringify(input),
  }),
  updateCDN: (id: string, input: CDNInput) => request<void>(`/api/cdns/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  }),
  deleteCDN: (id: string) => request<void>(`/api/cdns/${id}`, { method: 'DELETE' }),
  refreshSnapshot: () => request<MessageResponse>('/api/snapshot', { method: 'POST' }),
  listNodeHealth: () => request<NodeHealth[]>('/api/health/nodes'),
}
