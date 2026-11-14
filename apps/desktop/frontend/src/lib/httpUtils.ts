import type { KVPair, BodyState, AuthState } from '@/types'
import type { AuthPayload, RequestFileIO } from '@/lib/bridge'

export const HTTP_METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'] as const

export const HTTP_REQUEST_HEADERS: string[] = [
  // Content negotiation
  'Accept',
  'Accept-Charset',
  'Accept-Encoding',
  'Accept-Language',
  // Auth
  'Authorization',
  'Proxy-Authorization',
  // Caching
  'Cache-Control',
  'If-Match',
  'If-Modified-Since',
  'If-None-Match',
  'If-Range',
  'If-Unmodified-Since',
  'Pragma',
  // Connection
  'Connection',
  'Keep-Alive',
  'TE',
  'Trailer',
  'Transfer-Encoding',
  'Upgrade',
  // Request context
  'From',
  'Host',
  'Origin',
  'Referer',
  'User-Agent',
  // Body
  'Content-Encoding',
  'Content-Language',
  'Content-Length',
  'Content-Location',
  'Content-MD5',
  'Content-Range',
  'Content-Type',
  // Misc standard
  'Date',
  'Expect',
  'Forwarded',
  'Max-Forwards',
  'Range',
  'Via',
  // CORS
  'Access-Control-Request-Headers',
  'Access-Control-Request-Method',
  // Common API / custom
  'API-Key',
  'Idempotency-Key',
  'X-API-Key',
  'X-Auth-Token',
  'X-Content-Type-Options',
  'X-Correlation-ID',
  'X-CSRF-Token',
  'X-Forwarded-For',
  'X-Forwarded-Host',
  'X-Forwarded-Proto',
  'X-RateLimit-Limit',
  'X-RateLimit-Remaining',
  'X-Real-IP',
  'X-Request-ID',
  'X-Request-Start',
  'X-Response-Time',
]
export type HttpMethod = typeof HTTP_METHODS[number]

export const METHOD_COLOR: Record<HttpMethod, string> = {
  GET:     'text-emerald-400',
  POST:    'text-blue-400',
  PUT:     'text-amber-400',
  PATCH:   'text-orange-400',
  DELETE:  'text-red-400',
  HEAD:    'text-purple-400',
  OPTIONS: 'text-cyan-400',
}

export const METHOD_BADGE: Record<string, string> = { ...METHOD_COLOR as Record<string, string> }

export function statusColor(code: number): string {
  if (code >= 200 && code < 300) return 'text-emerald-400'
  if (code >= 300 && code < 400) return 'text-blue-400'
  if (code >= 400 && code < 500) return 'text-amber-400'
  return 'text-red-400'
}

export function prettyBody(body: string): string {
  try { return JSON.stringify(JSON.parse(body), null, 2) } catch { return body }
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function authStateToPayload(auth: AuthState): AuthPayload | undefined {
  if (auth.type === 'bearer') {
    return { type: 'bearer', token: auth.token, username: '', password: '', key: '', value: '', location: '' }
  }
  if (auth.type === 'basic') {
    return { type: 'basic', token: '', username: auth.username, password: auth.password, key: '', value: '', location: '' }
  }
  if (auth.type === 'apikey') {
    return { type: 'apikey', token: '', username: '', password: '', key: auth.key, value: auth.value, location: auth.location }
  }
  return undefined
}

export function requestFileIOToState(r: RequestFileIO): {
  method: HttpMethod
  url: string
  headers: KVPair[]
  queryParams: KVPair[]
  body: BodyState
  auth: AuthState
  description: string | null
} {
  const toKV = (pairs: Array<{ name: string; value: string }> = []): KVPair[] =>
    pairs.map(p => ({ id: crypto.randomUUID(), name: p.name, value: p.value, enabled: true }))

  let auth: AuthState = { type: 'none' }
  if (r.auth && r.auth.type !== 'none' && r.auth.type !== '') {
    if (r.auth.type === 'bearer') auth = { type: 'bearer', token: r.auth.token }
    else if (r.auth.type === 'basic') auth = { type: 'basic', username: r.auth.username, password: r.auth.password }
    else if (r.auth.type === 'apikey') {
      auth = { type: 'apikey', key: r.auth.key, value: r.auth.value, location: (r.auth.location as 'header' | 'query') || 'header' }
    }
  }

  const method = (HTTP_METHODS as readonly string[]).includes(r.method?.toUpperCase())
    ? r.method.toUpperCase() as HttpMethod
    : 'GET'

  return {
    method,
    url: r.url ?? '',
    headers: toKV(r.headers),
    queryParams: toKV(r.queryParams),
    body: r.body ? { type: r.body.type as BodyState['type'], content: r.body.content } : { type: 'none', content: '' },
    auth,
    description: r.description ?? null,
  }
}
