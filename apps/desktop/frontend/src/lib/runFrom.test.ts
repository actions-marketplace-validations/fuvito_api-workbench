import { describe, it, expect } from 'vitest'
import { missingOnRunFrom, describeMissing } from './runFrom'
import type { StepDetail } from '@/lib/bridge'

function step(over: Partial<StepDetail> = {}): StepDetail {
  return { label: 's', method: 'GET', url: '/a', assertions: [], extractions: [], ...over }
}

const login = step({ label: 'Login', extractions: [{ name: 'token', source: 'jsonpath', path: '$.token' }] })
const useToken = step({ label: 'Use', headers: [{ name: 'Authorization', value: 'Bearer {{token}}' }] })

describe('missingOnRunFrom', () => {
  it('reports an extracted value used by a later step when the start skips its extractor', () => {
    expect(missingOnRunFrom([login, useToken], undefined, 1)).toEqual([{ name: 'token', extractedIn: [1], usedIn: [2] }])
  })

  it('reports nothing when starting at the first step', () => {
    expect(missingOnRunFrom([login, useToken], undefined, 0)).toEqual([])
  })

  it('ignores scenario variables, which are always defined', () => {
    expect(missingOnRunFrom([login, useToken], [{ name: 'token', value: 'x' }], 1)).toEqual([])
  })

  it('ignores values that no remaining step uses', () => {
    expect(missingOnRunFrom([login, step()], undefined, 1)).toEqual([])
  })

  it('does not report a name that a step from the start point extracts before it is used', () => {
    const reLogin = step({ extractions: [{ name: 'token', source: 'jsonpath', path: '$.t' }] })
    expect(missingOnRunFrom([login, reLogin, useToken], undefined, 1)).toEqual([])
  })

  it('finds references in urls, bodies, query params and assertions, and skips disabled steps', () => {
    const steps = [
      login,
      step({ url: '/u/{{token}}', disabled: true }),
      step({ bodyContent: '{"t":"{{ token }}"}' }),
      step({ assertions: [{ type: 'status', operator: 'eq', expected: '{{token}}' }] }),
    ]
    expect(missingOnRunFrom(steps, undefined, 1)[0].usedIn).toEqual([3, 4])
  })
})

describe('describeMissing', () => {
  it('words one and several values', () => {
    expect(describeMissing([{ name: 'a', extractedIn: [1], usedIn: [2] }]))
      .toBe('Starting here skips earlier steps, so {{a}} (extracted in step 1) is not set.')
    expect(describeMissing([
      { name: 'a', extractedIn: [1], usedIn: [3] },
      { name: 'b', extractedIn: [1, 2], usedIn: [3] },
    ])).toBe('Starting here skips earlier steps, so {{a}} (extracted in step 1), {{b}} (extracted in steps 1, 2) are not set.')
  })
})
