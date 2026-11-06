import { describe, it, expect } from 'vitest'
import { parseCurl } from './curlImport'

describe('parseCurl', () => {
  it('returns null for non-curl input', () => {
    expect(parseCurl('wget https://example.com')).toBeNull()
    expect(parseCurl('')).toBeNull()
    expect(parseCurl('https://example.com')).toBeNull()
  })

  it('parses a simple GET', () => {
    const r = parseCurl('curl https://example.com')
    expect(r).not.toBeNull()
    expect(r!.method).toBe('GET')
    // URL constructor normalises bare origin to include trailing slash
    expect(r!.url).toBe('https://example.com/')
  })

  it('parses explicit -X method', () => {
    const r = parseCurl('curl -X DELETE https://example.com/item/1')
    expect(r!.method).toBe('DELETE')
  })

  it('parses -X attached to flag', () => {
    const r = parseCurl('curl -XPOST https://example.com')
    expect(r!.method).toBe('POST')
  })

  it('infers POST from -d body', () => {
    const r = parseCurl("curl -d '{\"a\":1}' https://example.com")
    expect(r!.method).toBe('POST')
  })

  it('parses query params from URL', () => {
    const r = parseCurl('curl "https://example.com/search?q=hello&page=2"')
    expect(r!.url).toBe('https://example.com/search')
    expect(r!.queryParams).toHaveLength(2)
    expect(r!.queryParams[0]).toMatchObject({ name: 'q', value: 'hello' })
    expect(r!.queryParams[1]).toMatchObject({ name: 'page', value: '2' })
  })

  it('parses -H headers', () => {
    const r = parseCurl("curl -H 'X-Custom: hello' https://example.com")
    expect(r!.headers).toHaveLength(1)
    expect(r!.headers[0]).toMatchObject({ name: 'X-Custom', value: 'hello' })
  })

  it('extracts Bearer auth from Authorization header', () => {
    const r = parseCurl("curl -H 'Authorization: Bearer mytoken' https://example.com")
    expect(r!.auth).toEqual({ type: 'bearer', token: 'mytoken' })
    expect(r!.headers).toHaveLength(0)
  })

  it('extracts Basic auth from -u flag', () => {
    const r = parseCurl('curl -u admin:secret https://example.com')
    expect(r!.auth).toEqual({ type: 'basic', username: 'admin', password: 'secret' })
  })

  it('parses JSON body and sets body type', () => {
    const r = parseCurl(`curl -X POST -H 'Content-Type: application/json' -d '{"name":"alice"}' https://example.com`)
    expect(r!.body.type).toBe('json')
    expect(r!.body.content).toBe('{"name":"alice"}')
  })

  it('handles line continuations', () => {
    const cmd = 'curl \\\n  -X GET \\\n  https://example.com'
    const r = parseCurl(cmd)
    expect(r!.method).toBe('GET')
    expect(r!.url).toBe('https://example.com/')
  })

  it('handles -G flag: moves body to query params', () => {
    const r = parseCurl('curl -G -d "page=1&size=10" https://example.com')
    expect(r!.method).toBe('GET')
    expect(r!.body.type).toBe('none')
    expect(r!.queryParams).toContainEqual(expect.objectContaining({ name: 'page', value: '1' }))
    expect(r!.queryParams).toContainEqual(expect.objectContaining({ name: 'size', value: '10' }))
  })

  it('parses -b (cookie) as Cookie header', () => {
    const r = parseCurl("curl -b 'session=abc' https://example.com")
    expect(r!.headers).toContainEqual(expect.objectContaining({ name: 'Cookie', value: 'session=abc' }))
  })

  it('parses -A (user-agent)', () => {
    const r = parseCurl("curl -A 'MyAgent/1.0' https://example.com")
    expect(r!.headers).toContainEqual(expect.objectContaining({ name: 'User-Agent', value: 'MyAgent/1.0' }))
  })
})
