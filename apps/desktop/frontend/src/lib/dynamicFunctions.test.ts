import { describe, it, expect, beforeEach } from 'vitest'
import { setDynamicFunctions, suggest } from './dynamicFunctions'

const fns = [
  { name: 'guid', usage: '$guid', description: 'Random UUID', example: '$guid' },
  { name: 'randomInt', usage: '$randomInt [min max]', description: 'Random whole number', example: '$randomInt 1 100' },
  { name: 'date', usage: '$date', description: 'Current date', example: '$date +7d' },
]

describe('suggest', () => {
  beforeEach(() => setDynamicFunctions(fns))

  it('lists variables first, then every function, right after {{', () => {
    const s = suggest('', ['baseUrl'])
    expect(s.map(x => x.label)).toEqual(['baseUrl', '$guid', '$randomInt', '$date'])
  })

  it('offers only functions once a $ is typed, filtered by name', () => {
    const s = suggest('$ran', ['randomVar'])
    expect(s.map(x => x.label)).toEqual(['$randomInt'])
    expect(s[0].insert).toBe('$randomInt 1 100')
    expect(s[0].hint).toBe('Random whole number')
  })

  it('stops matching functions after the first argument is typed', () => {
    expect(suggest('$randomInt 1', []).map(x => x.label)).toEqual(['$randomInt'])
  })

  it('does not offer functions while typing a plain variable name', () => {
    expect(suggest('base', ['baseUrl', 'token']).map(x => x.label)).toEqual(['baseUrl'])
  })

  it('offers nothing when no catalog is loaded and no variables match', () => {
    setDynamicFunctions([])
    expect(suggest('', [])).toEqual([])
  })
})
