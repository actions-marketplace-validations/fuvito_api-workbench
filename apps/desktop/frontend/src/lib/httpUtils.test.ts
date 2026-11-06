import { describe, it, expect } from 'vitest'
import { statusColor, formatSize, prettyBody, authStateToPayload, requestFileIOToState } from './httpUtils'
import type { RequestFileIO } from './bridge'

describe('statusColor', () => {
  it('2xx → emerald', () => {
    expect(statusColor(200)).toBe('text-emerald-400')
    expect(statusColor(201)).toBe('text-emerald-400')
    expect(statusColor(299)).toBe('text-emerald-400')
  })

  it('3xx → blue', () => {
    expect(statusColor(301)).toBe('text-blue-400')
    expect(statusColor(304)).toBe('text-blue-400')
  })

  it('4xx → amber', () => {
    expect(statusColor(400)).toBe('text-amber-400')
    expect(statusColor(404)).toBe('text-amber-400')
  })

  it('5xx → red', () => {
    expect(statusColor(500)).toBe('text-red-400')
    expect(statusColor(503)).toBe('text-red-400')
  })

  it('0 → red (network error)', () => {
    expect(statusColor(0)).toBe('text-red-400')
  })
})

describe('formatSize', () => {
  it('bytes under 1 KB', () => {
    expect(formatSize(0)).toBe('0 B')
    expect(formatSize(512)).toBe('512 B')
    expect(formatSize(1023)).toBe('1023 B')
  })

  it('kilobytes', () => {
    expect(formatSize(1024)).toBe('1.0 KB')
    expect(formatSize(1536)).toBe('1.5 KB')
  })

  it('megabytes', () => {
    expect(formatSize(1024 * 1024)).toBe('1.0 MB')
    expect(formatSize(1024 * 1024 * 2.5)).toBe('2.5 MB')
  })
})

describe('prettyBody', () => {
  it('pretty-prints valid JSON', () => {
    expect(prettyBody('{"a":1}')).toBe('{\n  "a": 1\n}')
  })

  it('returns input unchanged for non-JSON', () => {
    expect(prettyBody('hello world')).toBe('hello world')
  })

  it('returns input unchanged for invalid JSON', () => {
    expect(prettyBody('{bad}')).toBe('{bad}')
  })
})

describe('authStateToPayload', () => {
  it('returns undefined for none', () => {
    expect(authStateToPayload({ type: 'none' })).toBeUndefined()
  })

  it('maps bearer', () => {
    const p = authStateToPayload({ type: 'bearer', token: 'tok' })
    expect(p?.type).toBe('bearer')
    expect(p?.token).toBe('tok')
  })

  it('maps basic', () => {
    const p = authStateToPayload({ type: 'basic', username: 'u', password: 'p' })
    expect(p?.type).toBe('basic')
    expect(p?.username).toBe('u')
    expect(p?.password).toBe('p')
  })

  it('maps apikey', () => {
    const p = authStateToPayload({ type: 'apikey', key: 'X-Key', value: 'abc', location: 'header' })
    expect(p?.type).toBe('apikey')
    expect(p?.key).toBe('X-Key')
    expect(p?.value).toBe('abc')
    expect(p?.location).toBe('header')
  })
})

describe('requestFileIOToState', () => {
  const base: RequestFileIO = {
    method: 'GET',
    url: 'https://example.com',
    headers: [],
    queryParams: [],
    title: '',
    schemaVersion: '1',
  }

  it('maps method and url', () => {
    const s = requestFileIOToState(base)
    expect(s.method).toBe('GET')
    expect(s.url).toBe('https://example.com')
  })

  it('normalises method to uppercase', () => {
    const s = requestFileIOToState({ ...base, method: 'post' })
    expect(s.method).toBe('POST')
  })

  it('defaults invalid method to GET', () => {
    const s = requestFileIOToState({ ...base, method: 'INVALID' })
    expect(s.method).toBe('GET')
  })

  it('maps bearer auth', () => {
    const s = requestFileIOToState({ ...base, auth: { type: 'bearer', token: 'mytoken', username: '', password: '', key: '', value: '', location: '' } })
    expect(s.auth).toEqual({ type: 'bearer', token: 'mytoken' })
  })

  it('maps basic auth', () => {
    const s = requestFileIOToState({ ...base, auth: { type: 'basic', username: 'admin', password: 'secret', token: '', key: '', value: '', location: '' } })
    expect(s.auth).toEqual({ type: 'basic', username: 'admin', password: 'secret' })
  })

  it('maps body', () => {
    const s = requestFileIOToState({ ...base, body: { type: 'json', content: '{"x":1}' } })
    expect(s.body).toEqual({ type: 'json', content: '{"x":1}' })
  })

  it('defaults to no-body when body is null', () => {
    const s = requestFileIOToState(base)
    expect(s.body).toEqual({ type: 'none', content: '' })
  })

  it('converts header pairs to KVPair', () => {
    const s = requestFileIOToState({ ...base, headers: [{ name: 'X-Custom', value: 'hello' }] })
    expect(s.headers).toHaveLength(1)
    expect(s.headers[0].name).toBe('X-Custom')
    expect(s.headers[0].enabled).toBe(true)
  })
})
