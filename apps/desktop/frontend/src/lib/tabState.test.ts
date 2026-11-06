import { describe, it, expect } from 'vitest'
import { computeStateKey, createBlankTab, isTabDirty, isBlankTab } from './tabState'
import type { TabState } from './tabState'

const BLANK_KEY = computeStateKey('GET', '', [], [], { type: 'none', content: '' }, { type: 'none' }, null, 0)

describe('createBlankTab', () => {
  it('has GET method and empty url', () => {
    const tab = createBlankTab()
    expect(tab.method).toBe('GET')
    expect(tab.url).toBe('')
  })

  it('is not dirty on creation', () => {
    expect(isTabDirty(createBlankTab())).toBe(false)
  })

  it('assigns targetDir', () => {
    const tab = createBlankTab('/workspace/requests')
    expect(tab.targetDir).toBe('/workspace/requests')
  })

  it('savedKey matches blank state', () => {
    expect(createBlankTab().savedKey).toBe(BLANK_KEY)
  })
})

describe('computeStateKey', () => {
  it('trims url whitespace', () => {
    const a = computeStateKey('GET', '  https://example.com  ', [], [], { type: 'none', content: '' }, { type: 'none' }, null)
    const b = computeStateKey('GET', 'https://example.com', [], [], { type: 'none', content: '' }, { type: 'none' }, null)
    expect(a).toBe(b)
  })

  it('filters out params with empty names', () => {
    const a = computeStateKey('GET', '', [{ id: '1', name: '', value: 'x', enabled: true }], [], { type: 'none', content: '' }, { type: 'none' }, null)
    const b = computeStateKey('GET', '', [], [], { type: 'none', content: '' }, { type: 'none' }, null)
    expect(a).toBe(b)
  })

  it('includes timeoutMs', () => {
    const a = computeStateKey('GET', '', [], [], { type: 'none', content: '' }, { type: 'none' }, null, 0)
    const b = computeStateKey('GET', '', [], [], { type: 'none', content: '' }, { type: 'none' }, null, 5000)
    expect(a).not.toBe(b)
  })

  it('defaults timeoutMs to 0', () => {
    const a = computeStateKey('GET', '', [], [], { type: 'none', content: '' }, { type: 'none' }, null)
    const b = computeStateKey('GET', '', [], [], { type: 'none', content: '' }, { type: 'none' }, null, 0)
    expect(a).toBe(b)
  })
})

describe('isTabDirty', () => {
  function dirtyTab(overrides: Partial<TabState>): TabState {
    return { ...createBlankTab(), ...overrides }
  }

  it('returns false for a fresh blank tab', () => {
    expect(isTabDirty(createBlankTab())).toBe(false)
  })

  it('returns true when url changes', () => {
    const tab = dirtyTab({ url: 'https://example.com' })
    expect(isTabDirty(tab)).toBe(true)
  })

  it('returns true when method changes', () => {
    const tab = dirtyTab({ method: 'POST' })
    expect(isTabDirty(tab)).toBe(true)
  })

  it('returns true when timeoutMs changes', () => {
    const tab = dirtyTab({ timeoutMs: 3000 })
    expect(isTabDirty(tab)).toBe(true)
  })

  it('returns false after savedKey is updated to match', () => {
    const tab = dirtyTab({ url: 'https://api.example.com' })
    tab.savedKey = computeStateKey(tab.method, tab.url, tab.queryParams, tab.headers, tab.body, tab.auth, tab.name, tab.timeoutMs)
    expect(isTabDirty(tab)).toBe(false)
  })
})

describe('isBlankTab', () => {
  it('returns true for a fresh blank tab', () => {
    expect(isBlankTab(createBlankTab())).toBe(true)
  })

  it('returns false when url is set', () => {
    const tab = createBlankTab()
    tab.url = 'https://example.com'
    expect(isBlankTab(tab)).toBe(false)
  })

  it('returns false when filePath is set', () => {
    const tab = createBlankTab()
    tab.filePath = '/workspace/req.yaml'
    expect(isBlankTab(tab)).toBe(false)
  })

  it('returns false when name is set', () => {
    const tab = createBlankTab()
    tab.name = 'My Request'
    expect(isBlankTab(tab)).toBe(false)
  })
})
