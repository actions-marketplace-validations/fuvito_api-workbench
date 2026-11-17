import { describe, it, expect } from 'vitest'
import { parseOp, specLabel } from './coverage-report'

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
