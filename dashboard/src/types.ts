export interface User {
  id: string
  email: string
  created_at: string
}

export interface CDN {
  id: string
  origin: string
  domain: string
  is_active: boolean
  cache_ttl: number
}

export interface CDNInput {
  origin: string
  domain: string
  is_active: boolean
  cache_ttl: number
}

export interface LoginResponse {
  token: string
  user: User
}

export interface MessageResponse {
  message: string
}

export interface NodeHealth {
  id: string
  service: string
  instance: string
  status: string
  timestamp: string
  version: string
}
