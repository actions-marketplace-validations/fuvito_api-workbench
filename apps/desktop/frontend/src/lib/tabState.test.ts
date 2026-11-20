import { describe, it, expect } from 'vitest'
import { computeStateKey, createBlankTab, isTabDirty, isBlankTab, isPageTab, withoutTabsIn } from './tabState'
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

describe('isPageTab', () => {
  it('is true for app-screen tabs and false for request and scenario tabs', () => {
    for (const kind of ['coverage', 'runner', 'cli'] as const) expect(isPageTab(kind)).toBe(true)
    for (const kind of ['request', 'scenario', undefined] as const) expect(isPageTab(kind)).toBe(false)
  })

  it('page tabs are never dirty or blank', () => {
    const tab = { ...createBlankTab(), kind: 'cli' as const, url: 'x' }
    expect(isTabDirty(tab)).toBe(false)
    expect(isBlankTab(tab)).toBe(false)
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

describe('withoutTabsIn', () => {
  const tab = (over: Partial<TabState>): TabState => ({ ...createBlankTab(), ...over })

  it('removes tabs under the workspace or on the local server, and keeps the others', () => {
    const inFile = tab({ filePath: String.raw`C:\tmp\clrsig-x\requests\a.yaml` })
    const inScenario = tab({ kind: 'scenario', scenarioPath: 'C:/tmp/clrsig-x/scenarios/s.yaml' })
    const onServer = tab({ url: 'http://localhost:8120/health' })
    const other = tab({ filePath: String.raw`C:\work\b.yaml` })
    const next = withoutTabsIn({ tabs: [inFile, inScenario, onServer, other], activeTabId: inFile.id }, 'http://localhost:8120', String.raw`C:\tmp\clrsig-x`)
    expect(next.tabs).toEqual([other])
    expect(next.activeTabId).toBe(other.id)
  })

  it('keeps the active tab when it survives and returns the same state when nothing matches', () => {
    const a = tab({ filePath: '/ws/a.yaml' })
    const b = tab({ filePath: '/tmp/x/b.yaml' })
    const prev = { tabs: [a, b], activeTabId: a.id }
    expect(withoutTabsIn(prev, null, '/nowhere')).toBe(prev)
    expect(withoutTabsIn(prev, null, '/tmp/x').activeTabId).toBe(a.id)
  })

  it('leaves one blank tab when everything is removed', () => {
    const only = tab({ filePath: '/tmp/x/a.yaml' })
    const next = withoutTabsIn({ tabs: [only], activeTabId: only.id }, null, '/tmp/x')
    expect(next.tabs).toHaveLength(1)
    expect(isBlankTab(next.tabs[0])).toBe(true)
    expect(next.activeTabId).toBe(next.tabs[0].id)
  })
})
