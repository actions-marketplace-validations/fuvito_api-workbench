import type { KVPair, BodyState, AuthState } from '@/types'
import { HTTP_METHODS } from './httpUtils'
import type { HttpMethod } from './httpUtils'

export interface CurlImportResult {
  method: HttpMethod
  url: string
  queryParams: KVPair[]
  headers: KVPair[]
  body: BodyState
  auth: AuthState
}

// Split a curl command into tokens, respecting single/double quotes and line continuations.
function tokenize(cmd: string): string[] {
  const src = cmd.replace(/\\\n\s*/g, ' ').trim()
  const tokens: string[] = []
  let i = 0

  while (i < src.length) {
    while (i < src.length && /\s/.test(src[i])) i++
    if (i >= src.length) break

    if (src[i] === "'") {
      i++
      let s = ''
      while (i < src.length && src[i] !== "'") s += src[i++]
      i++ // closing '
      tokens.push(s)
    } else if (src[i] === '"') {
      i++
      let s = ''
      while (i < src.length && src[i] !== '"') {
        if (src[i] === '\\' && i + 1 < src.length) { i++; s += src[i++] }
        else s += src[i++]
      }
      i++ // closing "
      tokens.push(s)
    } else {
      let s = ''
      while (i < src.length && !/\s/.test(src[i])) s += src[i++]
      tokens.push(s)
    }
  }
  return tokens
}

function kv(name: string, value: string): KVPair {
  return { id: crypto.randomUUID(), name, value, enabled: true }
}

export function parseCurl(cmd: string): CurlImportResult | null {
  const trimmed = cmd.trim()
  if (!/^curl\b/i.test(trimmed)) return null

  const tokens = tokenize(trimmed)
  if (!tokens.length) return null

  let method = ''
  let rawUrl = ''
  const rawHeaders: [string, string][] = []
  let bodyContent = ''
  let basicCreds = ''
  let useGet = false           // -G: send --data as query string
  const urlencodeData: string[] = []

  let i = 1 // skip 'curl'
  while (i < tokens.length) {
    const t = tokens[i]

    if ((t === '-X' || t === '--request') && i + 1 < tokens.length) {
      method = tokens[++i].toUpperCase()
    } else if (/^-X.+/.test(t)) {
      method = t.slice(2).toUpperCase()
    } else if ((t === '-H' || t === '--header') && i + 1 < tokens.length) {
      const h = tokens[++i]
      const c = h.indexOf(':')
      if (c !== -1) rawHeaders.push([h.slice(0, c).trim(), h.slice(c + 1).trim()])
    } else if ((t === '-d' || t === '--data' || t === '--data-raw' || t === '--data-binary' || t === '--data-ascii') && i + 1 < tokens.length) {
      bodyContent = tokens[++i]
    } else if (t === '--data-urlencode' && i + 1 < tokens.length) {
      urlencodeData.push(tokens[++i])
    } else if ((t === '-u' || t === '--user') && i + 1 < tokens.length) {
      basicCreds = tokens[++i]
    } else if ((t === '-b' || t === '--cookie') && i + 1 < tokens.length) {
      rawHeaders.push(['Cookie', tokens[++i]])
    } else if ((t === '-A' || t === '--user-agent') && i + 1 < tokens.length) {
      rawHeaders.push(['User-Agent', tokens[++i]])
    } else if ((t === '-e' || t === '--referer') && i + 1 < tokens.length) {
      rawHeaders.push(['Referer', tokens[++i]])
    } else if (t === '-G' || t === '--get') {
      useGet = true
    } else if (/^-[a-zA-Z]$/.test(t) && i + 1 < tokens.length && !tokens[i + 1].startsWith('-')) {
      // unknown single-char flag that takes a value — skip both
      i++
    } else if (!t.startsWith('-')) {
      // bare argument = URL (take the first one)
      if (!rawUrl) rawUrl = t
    }
    // ignore boolean flags: -L, -k, -s, -v, -i, -S, --compressed, etc.

    i++
  }

  if (!rawUrl) return null

  // ---- URL: split out query params ----
  const queryParams: KVPair[] = []
  let baseUrl = rawUrl
  try {
    const u = new URL(rawUrl)
    u.searchParams.forEach((v, k) => queryParams.push(kv(k, v)))
    u.search = ''
    baseUrl = u.toString()
  } catch {
    const qIdx = rawUrl.indexOf('?')
    if (qIdx !== -1) {
      baseUrl = rawUrl.slice(0, qIdx)
      new URLSearchParams(rawUrl.slice(qIdx + 1)).forEach((v, k) => queryParams.push(kv(k, v)))
    }
  }

  // ---- -G: move body data to query params ----
  if (useGet) {
    if (bodyContent) {
      new URLSearchParams(bodyContent).forEach((v, k) => queryParams.push(kv(k, v)))
      bodyContent = ''
    }
    for (const seg of urlencodeData) {
      const eq = seg.indexOf('=')
      queryParams.push(eq !== -1 ? kv(seg.slice(0, eq), seg.slice(eq + 1)) : kv(seg, ''))
    }
    if (!method) method = 'GET'
  }

  // ---- Auth: extract from headers or -u ----
  let auth: AuthState = { type: 'none' }
  const headers: KVPair[] = []

  for (const [name, value] of rawHeaders) {
    if (name.toLowerCase() === 'authorization') {
      if (value.startsWith('Bearer ')) {
        auth = { type: 'bearer', token: value.slice(7) }
      } else if (value.startsWith('Basic ')) {
        try {
          const decoded = atob(value.slice(6))
          const c = decoded.indexOf(':')
          auth = { type: 'basic', username: c !== -1 ? decoded.slice(0, c) : decoded, password: c !== -1 ? decoded.slice(c + 1) : '' }
        } catch { headers.push(kv(name, value)) }
      } else {
        headers.push(kv(name, value))
      }
    } else {
      headers.push(kv(name, value))
    }
  }

  // -u flag overrides Authorization header
  if (basicCreds) {
    const c = basicCreds.indexOf(':')
    auth = { type: 'basic', username: c !== -1 ? basicCreds.slice(0, c) : basicCreds, password: c !== -1 ? basicCreds.slice(c + 1) : '' }
  }

  // ---- Body type ----
  let body: BodyState = { type: 'none', content: '' }
  if (bodyContent) {
    const t = bodyContent.trim()
    const type = (t.startsWith('{') || t.startsWith('[')) ? 'json'
      : rawHeaders.some(([n]) => n.toLowerCase() === 'content-type' && rawHeaders.find(([n2]) => n2.toLowerCase() === 'content-type')?.[1].includes('x-www-form-urlencoded')) ? 'form-urlencoded'
      : 'raw'
    body = { type, content: bodyContent }
  }

  // ---- Infer method ----
  if (!method) method = bodyContent ? 'POST' : 'GET'
  const finalMethod: HttpMethod = (HTTP_METHODS as readonly string[]).includes(method)
    ? method as HttpMethod
    : 'GET'

  return { method: finalMethod, url: baseUrl, queryParams, headers, body, auth }
}
