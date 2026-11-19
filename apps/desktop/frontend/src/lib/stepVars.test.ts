import { describe, it, expect } from 'vitest'
import { buildStepVarInfo, varsIn } from './stepVars'
import type { StepDetail } from '@/lib/bridge'

function step(over: Partial<StepDetail> = {}): StepDetail {
  return { label: 's', method: 'GET', url: '/a', assertions: [], extractions: [], ...over }
}

const login = step({ extractions: [{ name: 'token', source: 'jsonpath', path: '$.t' }] })
const create = step({ extractions: [{ name: 'id', source: 'jsonpath', path: '$.id' }] })

describe('buildStepVarInfo', () => {
  const steps = [login, create, step()]
  const info = buildStepVarInfo(
    steps, 2,
    [{ name: 'email', value: 'a@b.c' }, { name: 'pw', value: 'x', secret: true }],
    { host: { value: 'http://h', secret: false }, key: { value: 'k', secret: true } },
  )

  it('offers scenario variables, earlier extractions and environment names', () => {
    expect(info.names).toEqual(['email', 'pw', 'token', 'id', 'host', 'key'])
  })

  it('explains each origin', () => {
    expect(info.hint('token')).toEqual({ kind: 'extracted', text: 'extracted in step 1' })
    expect(info.hint('email')).toEqual({ kind: 'scenario', text: 'a@b.c' })
    expect(info.hint('host')).toEqual({ kind: 'env', text: 'http://h' })
    expect(info.hint('nope')).toEqual({ kind: 'missing', text: 'not defined' })
  })

  it('masks secret scenario and environment values', () => {
    expect(info.hint('pw').text).toBe('••••')
    expect(info.hint('key').text).toBe('••••')
  })

  it('prefers an earlier extraction over a scenario variable and the environment', () => {
    const i = buildStepVarInfo([login, step()], 1, [{ name: 'token', value: 'v' }], { token: { value: 'e', secret: false } })
    expect(i.hint('token').kind).toBe('extracted')
  })

  it('flags a name that only this or a later step extracts', () => {
    const i = buildStepVarInfo([step(), create, step({ extractions: [{ name: 'late', source: 'status', path: '' }] })], 1, [], {})
    expect(i.hint('id')).toEqual({ kind: 'later', text: 'extracted in step 2, not set yet here' })
    expect(i.hint('late').kind).toBe('later')
    expect(i.names).not.toContain('id')
  })

  it('truncates long values', () => {
    const i = buildStepVarInfo([step()], 0, [{ name: 'long', value: 'x'.repeat(60) }], {})
    expect(i.hint('long').text).toBe('x'.repeat(40) + '…')
  })
})

describe('varsIn', () => {
  it('lists distinct variables and skips dynamic functions', () => {
    expect(varsIn('{{base}}/u/{{ id }}?a={{$guid}}&b={{base}}&c={{$randomInt 1 5}}')).toEqual(['base', 'id'])
  })

  it('returns nothing for plain text', () => {
    expect(varsIn('/plain')).toEqual([])
  })
})
