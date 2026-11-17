import { describe, it, expect } from 'vitest'
import { parseCurl } from './curlImport'

describe('parseCurl — additional cases', () => {
  it('returns null when no URL is present', () => {
    expect(parseCurl('curl -X POST')).toBeNull()
  })

  it('decodes a Basic Authorization header', () => {
    const r = parseCurl(`curl https://x.test -H "Authorization: Basic ${btoa('alice:s3cret')}"`)!
    expect(r.auth).toEqual({ type: 'basic', username: 'alice', password: 's3cret' })
    expect(r.headers).toHaveLength(0)
  })

  it('handles a Basic header without a colon', () => {
    const r = parseCurl(`curl https://x.test -H "Authorization: Basic ${btoa('onlyuser')}"`)!
    expect(r.auth).toEqual({ type: 'basic', username: 'onlyuser', password: '' })
  })

  it('keeps an undecodable Basic header as a plain header', () => {
    const r = parseCurl('curl https://x.test -H "Authorization: Basic !!!notbase64"')!
    expect(r.auth).toEqual({ type: 'none' })
    expect(r.headers.map(h => h.name)).toEqual(['Authorization'])
  })

  it('keeps unknown Authorization schemes as headers', () => {
    const r = parseCurl('curl https://x.test -H "Authorization: Digest abc"')!
    expect(r.auth).toEqual({ type: 'none' })
    expect(r.headers[0].value).toBe('Digest abc')
  })

  it('lets -u override an Authorization header and handles a username-only value', () => {
    const r = parseCurl('curl -u bob https://x.test -H "Authorization: Bearer tok"')!
    expect(r.auth).toEqual({ type: 'basic', username: 'bob', password: '' })
  })

  it('treats form-encoded content types as form-urlencoded bodies', () => {
    const r = parseCurl(`curl https://x.test -H "Content-Type: application/x-www-form-urlencoded" -d 'a=1&b=2'`)!
    expect(r.body).toEqual({ type: 'form-urlencoded', content: 'a=1&b=2' })
    expect(r.method).toBe('POST')
  })

  it('treats other bodies as raw', () => {
    const r = parseCurl(`curl https://x.test --data-raw 'plain text'`)!
    expect(r.body).toEqual({ type: 'raw', content: 'plain text' })
  })

  it('detects JSON array bodies', () => {
    expect(parseCurl(`curl https://x.test -d '[1,2]'`)!.body.type).toBe('json')
  })

  it.each(['--data', '--data-binary', '--data-ascii'])('accepts %s', flag => {
    expect(parseCurl(`curl https://x.test ${flag} '{"a":1}'`)!.body.content).toBe('{"a":1}')
  })

  it('moves --data-urlencode values into the query string with -G', () => {
    const r = parseCurl('curl -G https://x.test --data-urlencode "q=hello world" --data-urlencode flag')!
    expect(r.method).toBe('GET')
    expect(r.queryParams.map(p => [p.name, p.value])).toEqual([['q', 'hello world'], ['flag', '']])
    expect(r.body.type).toBe('none')
  })

  it('moves -d data into the query string with --get and keeps an explicit method', () => {
    const r = parseCurl('curl --get -X DELETE https://x.test -d "a=1&b=2"')!
    expect(r.method).toBe('DELETE')
    expect(r.queryParams.map(p => p.name)).toEqual(['a', 'b'])
  })

  it('maps Referer, long-form headers and cookies', () => {
    const r = parseCurl('curl --request put https://x.test --header "X-A: 1" -e https://ref.test --cookie "s=1" --user-agent UA')!
    expect(r.method).toBe('PUT')
    expect(Object.fromEntries(r.headers.map(h => [h.name, h.value]))).toEqual({
      'X-A': '1', Referer: 'https://ref.test', Cookie: 's=1', 'User-Agent': 'UA',
    })
  })

  it('ignores header values without a colon', () => {
    expect(parseCurl('curl https://x.test -H "nocolon"')!.headers).toHaveLength(0)
  })

  it('skips unknown value-taking short flags and boolean flags', () => {
    const r = parseCurl('curl -L -s --compressed -m 5 https://x.test')!
    expect(r.url).toBe('https://x.test/')
  })

  it('falls back to GET for an unsupported method', () => {
    expect(parseCurl('curl -X PROPFIND https://x.test')!.method).toBe('GET')
  })

  it('parses query params from URLs that are not absolute', () => {
    const r = parseCurl('curl "/relative/path?a=1&b=2"')!
    expect(r.url).toBe('/relative/path')
    expect(r.queryParams.map(p => [p.name, p.value])).toEqual([['a', '1'], ['b', '2']])
  })

  it('keeps a relative URL without a query string unchanged', () => {
    const r = parseCurl('curl localhost:8080/x')!
    expect(r.queryParams).toHaveLength(0)
  })

  it('unescapes backslashes inside double quotes', () => {
    const r = parseCurl('curl https://x.test -H "X-Q: say \\"hi\\""')!
    expect(r.headers[0].value).toBe('say "hi"')
  })

  it('takes only the first bare argument as the URL', () => {
    expect(parseCurl('curl https://a.test https://b.test')!.url).toBe('https://a.test/')
  })
})
