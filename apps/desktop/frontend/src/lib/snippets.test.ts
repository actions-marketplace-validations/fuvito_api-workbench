import { describe, it, expect } from 'vitest'
import type { TabState } from './tabState'
import { buildCurl, buildFetch, buildPython } from './snippets'

function makeTab(overrides: Partial<TabState> = {}): TabState {
  return {
    id: 'test',
    method: 'GET',
    url: 'https://example.com/api',
    queryParams: [],
    headers: [],
    body: { type: 'none', content: '' },
    auth: { type: 'none' },
    timeoutMs: 0,
    name: null,
    filePath: null,
    targetDir: null,
    savedKey: '',
    response: null,
    previousResponse: null,
    responseCachedAt: null,
    loading: false,
    saveError: null,
    editingName: false,
    ...overrides,
  }
}

// ---- buildCurl ----

describe('buildCurl', () => {
  it('generates a minimal GET', () => {
    const out = buildCurl(makeTab())
    expect(out).toContain('curl -X GET')
    expect(out).toContain('"https://example.com/api"')
  })

  it('appends enabled query params to URL', () => {
    const out = buildCurl(makeTab({
      queryParams: [
        { id: '1', name: 'page', value: '2', enabled: true },
        { id: '2', name: 'skip', value: 'this', enabled: false },
      ],
    }))
    expect(out).toContain('page=2')
    expect(out).not.toContain('skip')
  })

  it('adds bearer token header', () => {
    const out = buildCurl(makeTab({ auth: { type: 'bearer', token: 'tok123' } }))
    expect(out).toContain('-H "Authorization: Bearer tok123"')
  })

  it('adds basic auth flag', () => {
    const out = buildCurl(makeTab({ auth: { type: 'basic', username: 'usr', password: 'pwd' } }))
    expect(out).toContain('--user "usr:pwd"')
  })

  it('adds apikey as header', () => {
    const out = buildCurl(makeTab({ auth: { type: 'apikey', key: 'X-Key', value: 'secret', location: 'header' } }))
    expect(out).toContain('-H "X-Key: secret"')
  })

  it('adds apikey as query param', () => {
    const out = buildCurl(makeTab({ auth: { type: 'apikey', key: 'api_key', value: 'abc', location: 'query' } }))
    expect(out).toContain('api_key=abc')
  })

  it('adds custom headers', () => {
    const out = buildCurl(makeTab({
      headers: [{ id: '1', name: 'X-Custom', value: 'val', enabled: true }],
    }))
    expect(out).toContain('-H "X-Custom: val"')
  })

  it('skips disabled headers', () => {
    const out = buildCurl(makeTab({
      headers: [{ id: '1', name: 'X-Skip', value: 'nope', enabled: false }],
    }))
    expect(out).not.toContain('X-Skip')
  })

  it('adds JSON body with Content-Type', () => {
    const out = buildCurl(makeTab({
      method: 'POST',
      body: { type: 'json', content: '{"x":1}' },
    }))
    expect(out).toContain('-H "Content-Type: application/json"')
    expect(out).toContain("--data '{\"x\":1}'")
  })

  it('adds form-urlencoded body with correct Content-Type', () => {
    const out = buildCurl(makeTab({
      method: 'POST',
      body: { type: 'form-urlencoded', content: 'a=1&b=2' },
    }))
    expect(out).toContain('application/x-www-form-urlencoded')
  })

  it('skips body when type is none', () => {
    const out = buildCurl(makeTab({ body: { type: 'none', content: 'ignored' } }))
    expect(out).not.toContain('--data')
  })
})

// ---- buildFetch ----

describe('buildFetch', () => {
  it('generates await fetch with method', () => {
    const out = buildFetch(makeTab())
    expect(out).toContain("await fetch('https://example.com/api'")
    expect(out).toContain("method: 'GET'")
  })

  it('adds bearer Authorization header', () => {
    const out = buildFetch(makeTab({ auth: { type: 'bearer', token: 'tok' } }))
    expect(out).toContain("'Authorization': 'Bearer tok'")
  })

  it('adds basic Authorization header as Base64', () => {
    const out = buildFetch(makeTab({ auth: { type: 'basic', username: 'u', password: 'p' } }))
    expect(out).toContain("'Authorization': 'Basic ")
  })

  it('adds apikey header', () => {
    const out = buildFetch(makeTab({ auth: { type: 'apikey', key: 'X-API', value: 'v', location: 'header' } }))
    expect(out).toContain("'X-API': 'v'")
  })

  it('adds apikey as query param in URL', () => {
    const out = buildFetch(makeTab({ auth: { type: 'apikey', key: 'token', value: 'secret', location: 'query' } }))
    expect(out).toContain('token=secret')
  })

  it('adds Content-Type for JSON body', () => {
    const out = buildFetch(makeTab({ body: { type: 'json', content: '{}' } }))
    expect(out).toContain("'Content-Type': 'application/json'")
  })

  it('adds body string for JSON', () => {
    const out = buildFetch(makeTab({ method: 'POST', body: { type: 'json', content: '{"a":1}' } }))
    expect(out).toContain('body:')
  })

  it('appends enabled query params to URL', () => {
    const out = buildFetch(makeTab({
      queryParams: [{ id: '1', name: 'q', value: 'test', enabled: true }],
    }))
    expect(out).toContain('q=test')
  })
})

// ---- buildPython ----

describe('buildPython', () => {
  it('generates import requests and correct method', () => {
    const out = buildPython(makeTab())
    expect(out).toContain('import requests')
    expect(out).toContain('requests.get(')
  })

  it('uses lowercase method', () => {
    const out = buildPython(makeTab({ method: 'POST' }))
    expect(out).toContain('requests.post(')
  })

  it('adds bearer Authorization header', () => {
    const out = buildPython(makeTab({ auth: { type: 'bearer', token: 'mytoken' } }))
    expect(out).toContain('"Authorization": "Bearer mytoken"')
  })

  it('adds basic auth tuple', () => {
    const out = buildPython(makeTab({ auth: { type: 'basic', username: 'usr', password: 'pw' } }))
    expect(out).toContain("auth=('usr', 'pw')")
  })

  it('adds apikey header', () => {
    const out = buildPython(makeTab({ auth: { type: 'apikey', key: 'X-K', value: 'v', location: 'header' } }))
    expect(out).toContain('"X-K": "v"')
  })

  it('adds apikey as query param', () => {
    const out = buildPython(makeTab({ auth: { type: 'apikey', key: 'k', value: 'val', location: 'query' } }))
    expect(out).toContain('"k": "val"')
    expect(out).toContain('params=')
  })

  it('includes enabled query params', () => {
    const out = buildPython(makeTab({
      queryParams: [{ id: '1', name: 'page', value: '1', enabled: true }],
    }))
    expect(out).toContain('"page": "1"')
  })

  it('adds json body', () => {
    const out = buildPython(makeTab({ method: 'POST', body: { type: 'json', content: '{"x":1}' } }))
    expect(out).toContain('json={"x":1}')
  })

  it('adds raw body as data string', () => {
    const out = buildPython(makeTab({ method: 'POST', body: { type: 'raw', content: 'raw data' } }))
    expect(out).toContain('data=')
  })

  it('ends with raise_for_status and response.json()', () => {
    const out = buildPython(makeTab())
    expect(out).toContain('response.raise_for_status()')
    expect(out).toContain('data = response.json()')
  })
})
