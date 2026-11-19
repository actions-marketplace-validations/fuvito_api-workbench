import { describe, it, expect } from 'vitest'
import { parseOp, specLabel, operationOutcomes, share } from './coverage-report'

// ── parseOp ──────────────────────────────────────────────────────────────────

describe('parseOp', () => {
  it('splits method and path', () => {
    expect(parseOp('GET /users')).toEqual({ method: 'GET', path: '/users' })
  })

  it('handles path with spaces in URL', () => {
    expect(parseOp('POST /users/123')).toEqual({ method: 'POST', path: '/users/123' })
  })

  it('handles no-space string', () => {
    expect(parseOp('GET')).toEqual({ method: 'GET', path: '' })
  })
})

// ── specLabel ─────────────────────────────────────────────────────────────────

describe('specLabel', () => {
  it('returns filename from Unix path', () => {
    expect(specLabel('/workspaces/my-api/openapi.yaml')).toBe('openapi.yaml')
  })

  it('returns filename from Windows path', () => {
    expect(specLabel('C:\\Users\\user\\api\\spec.yml')).toBe('spec.yml')
  })

  it('returns input unchanged when no separator', () => {
    expect(specLabel('openapi.yaml')).toBe('openapi.yaml')
  })

  it('works with URL', () => {
    expect(specLabel('http://localhost:8090/openapi.yaml')).toBe('openapi.yaml')
  })
})

describe('operationOutcomes', () => {
  const results = { 'GET /a': { lastPassed: true }, 'POST /b': { lastPassed: false } }

  it('splits covered operations into passing and failing by their last result', () => {
    expect(operationOutcomes({ total: 4, exercised: ['GET /a', 'POST /b'], missed: ['GET /c', 'GET /d'], operationResults: results }))
      .toEqual({ total: 4, covered: 2, passing: 1, failing: 1, noResult: 0, notCovered: 2 })
  })

  it('counts covered operations without a result separately', () => {
    expect(operationOutcomes({ total: 2, exercised: ['GET /a', 'POST /b'], missed: [] }))
      .toEqual({ total: 2, covered: 2, passing: 0, failing: 0, noResult: 2, notCovered: 0 })
  })

  it('derives never-tested from the total when the missed list is absent', () => {
    expect(operationOutcomes({ total: 5, exercised: ['GET /a'] }).notCovered).toBe(4)
  })
})

describe('share', () => {
  it('rounds to a whole percent and tolerates an empty total', () => {
    expect(share(1, 3)).toBe(33)
    expect(share(2, 3)).toBe(67)
    expect(share(0, 0)).toBe(0)
  })
})
