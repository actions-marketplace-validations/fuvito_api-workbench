import { describe, it, expect } from 'vitest'
import { parseOp, specLabel, csvCell, qualityColumns, buildCsvRows, generateMarkdownReport } from './coverage-report'
import type { AccumulatedCoverage } from './bridge'

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

// ── csvCell ───────────────────────────────────────────────────────────────────

describe('csvCell', () => {
  it('passes through plain values', () => {
    expect(csvCell('GET')).toBe('GET')
    expect(csvCell('/users')).toBe('/users')
  })

  it('wraps values containing commas', () => {
    expect(csvCell('a,b')).toBe('"a,b"')
  })

  it('wraps values containing double quotes and escapes them', () => {
    expect(csvCell('say "hi"')).toBe('"say ""hi"""')
  })

  it('wraps values containing newlines', () => {
    expect(csvCell('line1\nline2')).toBe('"line1\nline2"')
  })
})

// ── qualityColumns ────────────────────────────────────────────────────────────

describe('qualityColumns', () => {
  it('returns Never exercised for uncovered operations', () => {
    expect(qualityColumns('GET /users', false, undefined)).toEqual(['Never exercised', ''])
  })

  it('returns empty strings when no quality data', () => {
    expect(qualityColumns('GET /users', true, undefined)).toEqual(['', ''])
  })

  it('reports happy-path-only flag', () => {
    const q = {
      neverExercised: [],
      happyPathOnly: ['GET /users'],
      securedNeverAuthed: [],
      errorsNeverTriggered: [],
    }
    const [quality, missing] = qualityColumns('GET /users', true, q)
    expect(quality).toBe('Happy-path only')
    expect(missing).toBe('')
  })

  it('reports auth gap flag', () => {
    const q = {
      neverExercised: [],
      happyPathOnly: [],
      securedNeverAuthed: ['POST /login'],
      errorsNeverTriggered: [],
    }
    expect(qualityColumns('POST /login', true, q)[0]).toBe('No auth tested')
  })

  it('reports errors never triggered with missing codes', () => {
    const q = {
      neverExercised: [],
      happyPathOnly: [],
      securedNeverAuthed: [],
      errorsNeverTriggered: [{ operation: 'DELETE /users/{id}', missing: [404, 403] }],
    }
    const [quality, missing] = qualityColumns('DELETE /users/{id}', true, q)
    expect(quality).toBe('Errors never triggered')
    expect(missing).toBe('404 403')
  })

  it('combines multiple flags with semicolon', () => {
    const q = {
      neverExercised: [],
      happyPathOnly: ['GET /users'],
      securedNeverAuthed: ['GET /users'],
      errorsNeverTriggered: [],
    }
    const [quality] = qualityColumns('GET /users', true, q)
    expect(quality).toBe('Happy-path only; No auth tested')
  })

  it('returns empty flags for clean operation', () => {
    const q = {
      neverExercised: [],
      happyPathOnly: [],
      securedNeverAuthed: [],
      errorsNeverTriggered: [],
    }
    expect(qualityColumns('GET /health', true, q)).toEqual(['', ''])
  })
})

// ── buildCsvRows ──────────────────────────────────────────────────────────────

function minimalEntry(overrides: Partial<AccumulatedCoverage> = {}): AccumulatedCoverage {
  return {
    specFile: '/ws/openapi.yaml',
    percent: 50,
    covered: 1,
    total: 2,
    totalRuns: 3,
    passedRuns: 2,
    failedRuns: 1,
    exercised: ['GET /users'],
    missed: ['POST /users'],
    operationResults: {},
    ...overrides,
  }
}

describe('buildCsvRows', () => {
  it('first row is the header', () => {
    const rows = buildCsvRows(minimalEntry())
    expect(rows[0]).toBe('Method,Path,Coverage,Last Result,Last Run,Quality,Missing Codes')
  })

  it('covered operation gets "Covered"', () => {
    const rows = buildCsvRows(minimalEntry())
    const covered = rows.find(r => r.includes('GET') && r.includes('Covered'))
    expect(covered).toBeDefined()
  })

  it('missed operation gets "Not Covered" and Never exercised quality', () => {
    const rows = buildCsvRows(minimalEntry())
    const missed = rows.find(r => r.includes('POST') && r.includes('Not Covered'))
    expect(missed).toBeDefined()
    expect(missed).toContain('Never exercised')
  })

  it('rows are sorted by path then method', () => {
    const entry = minimalEntry({
      exercised: ['POST /users', 'GET /health'],
      missed: [],
    })
    const rows = buildCsvRows(entry)
    const paths = rows.slice(1).map(r => r.split(',')[1])
    expect(paths).toEqual(['/health', '/users'])
  })

  it('includes quality columns when quality data present', () => {
    const entry = minimalEntry({
      quality: {
        neverExercised: [],
        happyPathOnly: ['GET /users'],
        securedNeverAuthed: [],
        errorsNeverTriggered: [],
      },
    })
    const rows = buildCsvRows(entry)
    const row = rows.find(r => r.includes('GET') && r.includes('Covered'))
    expect(row).toContain('Happy-path only')
  })

  it('includes missing error codes in last column', () => {
    const entry = minimalEntry({
      quality: {
        neverExercised: [],
        happyPathOnly: [],
        securedNeverAuthed: [],
        errorsNeverTriggered: [{ operation: 'GET /users', missing: [404, 500] }],
      },
    })
    const rows = buildCsvRows(entry)
    const row = rows.find(r => r.startsWith('GET,'))
    expect(row).toContain('404 500')
  })
})

// ── generateMarkdownReport ────────────────────────────────────────────────────

describe('generateMarkdownReport', () => {
  const fixedNow = new Date('2026-11-16T12:00:00Z')

  it('includes spec filename in heading', () => {
    const md = generateMarkdownReport('/ws/openapi.yaml', minimalEntry(), fixedNow)
    expect(md).toContain('# Contracts Report — openapi.yaml')
  })

  it('includes coverage summary', () => {
    const md = generateMarkdownReport('/ws/openapi.yaml', minimalEntry(), fixedNow)
    expect(md).toContain('**1 / 2 operations covered (50%)**')
  })

  it('includes operations table', () => {
    const md = generateMarkdownReport('/ws/openapi.yaml', minimalEntry(), fixedNow)
    expect(md).toContain('## Operations')
    expect(md).toContain('`GET`')
    expect(md).toContain('`/users`')
    expect(md).toContain('✅ Covered')
  })

  it('includes missed operations as uncovered', () => {
    const md = generateMarkdownReport('/ws/openapi.yaml', minimalEntry(), fixedNow)
    expect(md).toContain('`POST`')
    expect(md).not.toContain('✅ Covered\n')
  })

  it('includes quality gaps section when gaps exist', () => {
    const entry = minimalEntry({
      quality: {
        neverExercised: [],
        happyPathOnly: ['GET /users'],
        securedNeverAuthed: [],
        errorsNeverTriggered: [],
      },
    })
    const md = generateMarkdownReport('/ws/openapi.yaml', entry, fixedNow)
    expect(md).toContain('## Quality Gaps')
    expect(md).toContain('### Happy-path only (1)')
    expect(md).toContain('`GET /users`')
  })

  it('includes error gap table with missing codes', () => {
    const entry = minimalEntry({
      quality: {
        neverExercised: [],
        happyPathOnly: [],
        securedNeverAuthed: [],
        errorsNeverTriggered: [{ operation: 'GET /users', missing: [404, 403] }],
      },
    })
    const md = generateMarkdownReport('/ws/openapi.yaml', entry, fixedNow)
    expect(md).toContain('### Error responses never triggered (1)')
    expect(md).toContain('404, 403')
  })

  it('reports all-clear when no quality gaps', () => {
    const entry = minimalEntry({
      quality: {
        neverExercised: [],
        happyPathOnly: [],
        securedNeverAuthed: [],
        errorsNeverTriggered: [],
      },
    })
    const md = generateMarkdownReport('/ws/openapi.yaml', entry, fixedNow)
    expect(md).toContain('_All quality signals clear._')
  })

  it('omits quality section when no quality data', () => {
    const md = generateMarkdownReport('/ws/openapi.yaml', minimalEntry(), fixedNow)
    expect(md).not.toContain('## Quality Gaps')
  })
})
