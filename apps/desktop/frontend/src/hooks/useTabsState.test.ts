import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import type { HistoryEntry, RequestFileIO } from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({
  saveRequest: vi.fn(),
  loadResponseCache: vi.fn(),
  setHasDirtyTabs: vi.fn(() => Promise.resolve()),
}))

import { saveRequest, loadResponseCache, setHasDirtyTabs } from '@/lib/bridge'
import { useTabsState } from './useTabsState'

function setup() {
  const refresh = vi.fn(() => Promise.resolve())
  const hook = renderHook(() => useTabsState({ refreshWorkspaceRef: { current: refresh } }))
  return { ...hook, refresh }
}

type Hook = ReturnType<typeof setup>['result']

const file = (over: Partial<RequestFileIO> = {}): RequestFileIO => ({
  schemaVersion: '1', title: 'Get users', method: 'post', url: 'https://x.test/users', ...over,
})

const entry = (over: Partial<HistoryEntry> = {}): HistoryEntry => ({
  id: 1, executedAt: '', method: 'GET', url: 'https://x.test/a', statusCode: 200, statusText: 'OK',
  durationMs: 1, sizeBytes: 1, headers: [], bodyType: 'none', bodyContent: '', ...over,
})

function dirty(result: Hook, id = result.current.activeTabId) {
  act(() => result.current.updateTab(id, { url: 'https://changed.test' }))
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(setHasDirtyTabs).mockResolvedValue(undefined)
})

describe('tab lifecycle', () => {
  it('starts with one blank, clean tab', () => {
    const { result } = setup()
    expect(result.current.tabs).toHaveLength(1)
    expect(result.current.activeTab.id).toBe(result.current.activeTabId)
    expect(result.current.dirtyFilePaths.size).toBe(0)
  })

  it('new tab is appended and activated', () => {
    const { result } = setup()
    const first = result.current.activeTabId
    act(() => result.current.handleNewTab())
    expect(result.current.tabs).toHaveLength(2)
    expect(result.current.activeTabId).not.toBe(first)
    act(() => result.current.handleSelectTab(first))
    expect(result.current.activeTabId).toBe(first)
  })

  it('closing the only tab leaves a fresh blank tab with the same id', () => {
    const { result } = setup()
    const id = result.current.activeTabId
    act(() => result.current.removeTab(id))
    expect(result.current.tabs).toHaveLength(1)
    expect(result.current.activeTabId).toBe(id)
  })

  it('closing the active tab activates its neighbour', () => {
    const { result } = setup()
    act(() => result.current.handleNewTab())
    act(() => result.current.handleNewTab())
    const [a, b, c] = result.current.tabs.map(t => t.id)
    act(() => result.current.handleSelectTab(b))
    act(() => result.current.removeTab(b))
    expect(result.current.tabs.map(t => t.id)).toEqual([a, c])
    expect(result.current.activeTabId).toBe(c)
    act(() => result.current.removeTab(c))
    expect(result.current.activeTabId).toBe(a)
  })

  it('closing an inactive tab keeps the active one', () => {
    const { result } = setup()
    act(() => result.current.handleNewTab())
    const [a, b] = result.current.tabs.map(t => t.id)
    act(() => result.current.removeTab(a))
    expect(result.current.activeTabId).toBe(b)
  })

  it('duplicate copies the request but detaches it from the file', () => {
    const { result } = setup()
    act(() => result.current.openFileAsTab(file(), '/ws/a.yaml'))
    act(() => result.current.updateTab(result.current.activeTabId, { response: { statusCode: 200 } as never }))
    act(() => result.current.handleDuplicateTab())
    const copy = result.current.activeTab
    expect(result.current.tabs).toHaveLength(2)
    expect(copy.name).toBe('Get users copy')
    expect(copy.filePath).toBeNull()
    expect(copy.response).toBeNull()
    expect(copy.url).toBe('https://x.test/users')
  })

  it('duplicate is a no-op for scenario and coverage tabs', () => {
    const { result } = setup()
    act(() => result.current.openCoverageTab())
    act(() => result.current.handleDuplicateTab())
    expect(result.current.tabs.filter(t => t.kind === 'coverage')).toHaveLength(1)
    expect(result.current.tabs).toHaveLength(2)
  })

  it('new request in folder remembers the target directory', () => {
    const { result } = setup()
    act(() => result.current.handleNewRequestInFolder('/ws/requests/auth'))
    expect(result.current.activeTab.targetDir).toBe('/ws/requests/auth')
  })
})

describe('closing with unsaved changes', () => {
  it('closes a clean tab immediately', () => {
    const { result } = setup()
    act(() => result.current.handleNewTab())
    const id = result.current.activeTabId
    act(() => result.current.handleCloseTab(id))
    expect(result.current.tabs).toHaveLength(1)
    expect(result.current.closingDirtyTabId).toBeNull()
  })

  it('asks before closing a dirty request tab', () => {
    const { result } = setup()
    const id = result.current.activeTabId
    dirty(result)
    act(() => result.current.handleCloseTab(id))
    expect(result.current.closingDirtyTabId).toBe(id)
    expect(result.current.tabs).toHaveLength(1)
    expect(result.current.tabs[0].url).toBe('https://changed.test')
  })

  it('asks before closing a dirty scenario tab with its own dialog', () => {
    const { result } = setup()
    act(() => { result.current.openScenarioAsTab('/s.yaml', 'S') })
    const id = result.current.activeTabId
    act(() => result.current.updateTab(id, { scenarioDirty: true }))
    act(() => result.current.handleCloseTab(id))
    expect(result.current.closingDirtyScenarioTabId).toBe(id)
    expect(result.current.closingDirtyTabId).toBeNull()
  })

  it('ignores unknown ids', () => {
    const { result } = setup()
    act(() => result.current.handleCloseTab('missing'))
    expect(result.current.tabs).toHaveLength(1)
  })

  it('close others keeps one tab and activates it', () => {
    const { result } = setup()
    act(() => result.current.handleNewTab())
    act(() => result.current.handleNewTab())
    const keep = result.current.tabs[1].id
    act(() => result.current.handleCloseOthers(keep))
    expect(result.current.tabs.map(t => t.id)).toEqual([keep])
    expect(result.current.activeTabId).toBe(keep)
  })

  it('close others with nothing else open is a no-op', () => {
    const { result } = setup()
    act(() => result.current.handleCloseOthers(result.current.activeTabId))
    expect(result.current.pendingBulkClose).toBeNull()
  })

  it('close others with dirty tabs waits for confirmation, then returns to the kept tab', () => {
    const { result } = setup()
    const first = result.current.activeTabId
    dirty(result)
    act(() => result.current.handleNewTab())
    const keep = result.current.activeTabId
    act(() => result.current.handleCloseOthers(keep))
    expect(result.current.pendingBulkClose).toEqual({ tabIds: [first], returnToId: keep })
    expect(result.current.tabs).toHaveLength(2)
    act(() => result.current.confirmBulkClose())
    expect(result.current.tabs.map(t => t.id)).toEqual([keep])
    expect(result.current.activeTabId).toBe(keep)
    expect(result.current.pendingBulkClose).toBeNull()
  })

  it('close to the right closes only later tabs', () => {
    const { result } = setup()
    act(() => result.current.handleNewTab())
    act(() => result.current.handleNewTab())
    const [a, b] = result.current.tabs.map(t => t.id)
    act(() => result.current.handleCloseToRight(b))
    expect(result.current.tabs.map(t => t.id)).toEqual([a, b])
    act(() => result.current.handleCloseToRight(b))
    expect(result.current.tabs).toHaveLength(2)
    act(() => result.current.handleCloseToRight('missing'))
    expect(result.current.tabs).toHaveLength(2)
  })

  it('close to the right asks first when a later tab is dirty', () => {
    const { result } = setup()
    act(() => result.current.handleNewTab())
    const [a, b] = result.current.tabs.map(t => t.id)
    dirty(result, b)
    act(() => result.current.handleCloseToRight(a))
    expect(result.current.pendingBulkClose).toEqual({ tabIds: [b] })
  })

  it('close all resets to a single blank tab, or asks when something is dirty', () => {
    const { result } = setup()
    act(() => result.current.handleNewTab())
    act(() => result.current.handleCloseAll())
    expect(result.current.tabs).toHaveLength(1)

    act(() => result.current.handleNewTab())
    dirty(result)
    act(() => result.current.handleCloseAll())
    expect(result.current.pendingBulkClose?.tabIds).toHaveLength(2)
    expect(result.current.tabs).toHaveLength(2)
  })

  it('confirmBulkClose without a pending request does nothing', () => {
    const { result } = setup()
    act(() => result.current.confirmBulkClose())
    expect(result.current.tabs).toHaveLength(1)
  })

  it('treats dirty scenario tabs as dirty for bulk close', () => {
    const { result } = setup()
    act(() => { result.current.openScenarioAsTab('/s.yaml', 'S') })
    act(() => result.current.updateTab(result.current.activeTabId, { scenarioDirty: true }))
    act(() => result.current.handleCloseAll())
    expect(result.current.pendingBulkClose).not.toBeNull()
  })
})

describe('opening things', () => {
  it('opening a file in a blank tab reuses that tab', () => {
    const { result } = setup()
    const blank = result.current.activeTabId
    let id = ''
    act(() => { id = result.current.openFileAsTab(file(), '/ws/a.yaml') })
    expect(id).toBe(blank)
    expect(result.current.tabs).toHaveLength(1)
    expect(result.current.activeTab).toMatchObject({ method: 'POST', filePath: '/ws/a.yaml', name: 'Get users' })
  })

  it('opening a file when the active tab has content adds a new tab', () => {
    const { result } = setup()
    dirty(result)
    let id = ''
    act(() => { id = result.current.openFileAsTab(file(), '/ws/a.yaml') })
    expect(result.current.tabs).toHaveLength(2)
    expect(result.current.activeTabId).toBe(id)
  })

  it('a freshly opened file is not dirty', () => {
    const { result } = setup()
    act(() => { result.current.openFileAsTab(file({ timeoutMs: 5000, headers: [{ name: 'A', value: 'b' }] }), '/ws/a.yaml') })
    expect(result.current.tabs.some(t => t.filePath && t.savedKey === '')).toBe(false)
    dirty(result)
    expect(result.current.dirtyFilePaths.has('/ws/a.yaml')).toBe(true)
  })

  it('runner and coverage tabs are singletons', () => {
    const { result } = setup()
    let a = '', b = ''
    act(() => { a = result.current.openRunnerTab() })
    act(() => result.current.handleNewTab())
    act(() => { b = result.current.openRunnerTab() })
    expect(b).toBe(a)
    expect(result.current.activeTabId).toBe(a)
    expect(result.current.tabs.filter(t => t.kind === 'runner')).toHaveLength(1)

    let c = '', d = ''
    act(() => { c = result.current.openCoverageTab() })
    act(() => { d = result.current.openCoverageTab() })
    expect(d).toBe(c)
    expect(result.current.tabs.filter(t => t.kind === 'coverage')).toHaveLength(1)
    expect(result.current.tabs.find(t => t.id === c)).toMatchObject({ name: 'Contracts', kind: 'coverage' })
  })

  it('the CLI builder tab is a singleton page tab', () => {
    const { result } = setup()
    let a = '', b = ''
    act(() => { a = result.current.openCliTab() })
    act(() => result.current.handleNewTab())
    act(() => { b = result.current.openCliTab() })
    expect(b).toBe(a)
    expect(result.current.activeTabId).toBe(a)
    expect(result.current.tabs.filter(t => t.kind === 'cli')).toHaveLength(1)
    expect(result.current.tabs.find(t => t.id === a)).toMatchObject({ name: 'CLI Builder', kind: 'cli' })
  })

  it('opening the same scenario twice switches to the existing tab', () => {
    const { result } = setup()
    let a = '', b = ''
    act(() => { a = result.current.openScenarioAsTab('/s.yaml', 'S') })
    act(() => result.current.handleNewTab())
    act(() => { b = result.current.openScenarioAsTab('/s.yaml', 'S') })
    expect(b).toBe(a)
    expect(result.current.activeTabId).toBe(a)
    act(() => { result.current.openScenarioAsTab('/other.yaml', 'O') })
    expect(result.current.tabs.filter(t => t.kind === 'scenario')).toHaveLength(2)
  })

  it('scenario tabs are never dirty from request fields', () => {
    const { result } = setup()
    act(() => { result.current.openScenarioAsTab('/s.yaml', 'S') })
    act(() => result.current.updateTab(result.current.activeTabId, { url: 'changed' }))
    expect(result.current.dirtyFilePaths.size).toBe(0)
  })
})

describe('response cache', () => {
  it('fills an empty response from the cache', async () => {
    vi.mocked(loadResponseCache).mockResolvedValue({ cachedAt: '2026-01-01T00:00:00Z', statusCode: 200, body: 'x' } as never)
    const { result } = setup()
    const id = result.current.activeTabId
    await act(() => result.current.loadCacheIntoTab(id, '/ws/a.yaml'))
    expect(result.current.activeTab.response).toMatchObject({ statusCode: 200, body: 'x' })
    expect(result.current.activeTab.responseCachedAt).toBe('2026-01-01T00:00:00Z')
  })

  it('never overwrites a live response with a cached one', async () => {
    vi.mocked(loadResponseCache).mockResolvedValue({ cachedAt: 'then', statusCode: 500 } as never)
    const { result } = setup()
    const id = result.current.activeTabId
    act(() => result.current.updateTab(id, { response: { statusCode: 200 } as never }))
    await act(() => result.current.loadCacheIntoTab(id, '/p'))
    expect(result.current.activeTab.response).toMatchObject({ statusCode: 200 })
  })

  it('treats a missing cache or a bridge error as a quiet miss', async () => {
    const { result } = setup()
    const id = result.current.activeTabId
    vi.mocked(loadResponseCache).mockResolvedValueOnce(null as never)
    await act(() => result.current.loadCacheIntoTab(id, '/p'))
    vi.mocked(loadResponseCache).mockRejectedValueOnce(new Error('x'))
    await act(() => result.current.loadCacheIntoTab(id, '/p'))
    expect(result.current.activeTab.response).toBeNull()
  })
})

describe('saving', () => {
  it('saves enabled, named fields only and marks the tab clean', async () => {
    vi.mocked(saveRequest).mockResolvedValue('/ws/requests/users.yaml')
    const { result, refresh } = setup()
    const id = result.current.activeTabId
    act(() => result.current.updateTab(id, {
      url: 'https://x.test/api/users/', method: 'PUT', timeoutMs: 2500,
      headers: [
        { id: '1', name: 'A', value: '1', enabled: true },
        { id: '2', name: 'B', value: '2', enabled: false },
        { id: '3', name: '  ', value: '3', enabled: true },
      ],
      queryParams: [{ id: '4', name: 'q', value: 'z', enabled: true }],
      body: { type: 'json', content: '{}' },
      auth: { type: 'bearer', token: 't' },
    }))
    let ok = false
    await act(async () => { ok = await result.current.handleSaveRequest() })

    expect(ok).toBe(true)
    const sent = vi.mocked(saveRequest).mock.calls[0][0]
    expect(sent).toMatchObject({
      title: 'users', method: 'PUT', url: 'https://x.test/api/users/', timeoutMs: 2500,
      headers: [{ name: 'A', value: '1' }], queryParams: [{ name: 'q', value: 'z' }],
      body: { type: 'json', content: '{}' }, filePath: '',
    })
    expect(sent.auth).toMatchObject({ type: 'bearer', token: 't' })
    expect(result.current.activeTab).toMatchObject({ filePath: '/ws/requests/users.yaml', name: 'users' })
    expect(result.current.dirtyFilePaths.size).toBe(0)
    expect(refresh).toHaveBeenCalled()
  })

  it('omits body, auth and timeout when they are unset, and falls back to the name "request"', async () => {
    vi.mocked(saveRequest).mockResolvedValue('/p')
    const { result } = setup()
    await act(async () => { await result.current.handleSaveRequest() })
    const sent = vi.mocked(saveRequest).mock.calls[0][0]
    expect(sent.title).toBe('request')
    expect(sent.body).toBeUndefined()
    expect(sent.auth).toBeUndefined()
    expect(sent.timeoutMs).toBeUndefined()
  })

  it('returns false and keeps the tab dirty when the user cancels', async () => {
    vi.mocked(saveRequest).mockResolvedValue('')
    const { result, refresh } = setup()
    dirty(result)
    let ok = true
    await act(async () => { ok = await result.current.handleSaveRequest() })
    expect(ok).toBe(false)
    expect(refresh).not.toHaveBeenCalled()
    expect(result.current.activeTab.filePath).toBeNull()
  })

  it('surfaces save errors on the tab without the "Error: " prefix', async () => {
    vi.mocked(saveRequest).mockRejectedValue(new Error('a request named "x" already exists'))
    const { result } = setup()
    let ok = true
    await act(async () => { ok = await result.current.handleSaveRequest() })
    expect(ok).toBe(false)
    expect(result.current.activeTab.saveError).toBe('a request named "x" already exists')
  })

  it('clears a previous save error when saving again', async () => {
    vi.mocked(saveRequest).mockRejectedValueOnce(new Error('boom')).mockResolvedValueOnce('/p')
    const { result } = setup()
    await act(async () => { await result.current.handleSaveRequest() })
    expect(result.current.activeTab.saveError).toBe('boom')
    await act(async () => { await result.current.handleSaveRequest() })
    expect(result.current.activeTab.saveError).toBeNull()
  })

  it('save-as always sends an empty filePath so the backend picks a new file', async () => {
    vi.mocked(saveRequest).mockResolvedValue('/new.yaml')
    const { result } = setup()
    act(() => { result.current.openFileAsTab(file(), '/ws/old.yaml') })
    let ok = false
    await act(async () => { ok = await result.current.handleSaveAsRequest() })
    expect(ok).toBe(true)
    expect(vi.mocked(saveRequest).mock.calls[0][0].filePath).toBe('')
    expect(result.current.activeTab.filePath).toBe('/new.yaml')
  })

  it('save-as reports failures and cancellations', async () => {
    const { result } = setup()
    vi.mocked(saveRequest).mockResolvedValueOnce('')
    await act(async () => { expect(await result.current.handleSaveAsRequest()).toBe(false) })
    vi.mocked(saveRequest).mockRejectedValueOnce(new Error('nope'))
    await act(async () => { expect(await result.current.handleSaveAsRequest()).toBe(false) })
    expect(result.current.activeTab.saveError).toBe('nope')
  })

  it('saving a scenario tab only bumps its save signal', async () => {
    const { result } = setup()
    act(() => { result.current.openScenarioAsTab('/s.yaml', 'S') })
    const id = result.current.activeTabId
    await act(async () => { expect(await result.current.saveTabById(id)).toBe(true) })
    await act(async () => { await result.current.saveTabById(id) })
    expect(result.current.activeTab.scenarioSaveSignal).toBe(2)
    expect(saveRequest).not.toHaveBeenCalled()
  })

  it('saveTabById is false for unknown tabs', async () => {
    const { result } = setup()
    await act(async () => { expect(await result.current.saveTabById('missing')).toBe(false) })
  })

  it('save all saves every dirty file-backed tab but skips unsaved ones', async () => {
    vi.mocked(saveRequest).mockImplementation(async r => r.filePath ?? '')
    const { result } = setup()
    act(() => { result.current.openFileAsTab(file({ title: 'a' }), '/ws/a.yaml') })
    dirty(result)
    act(() => result.current.handleNewTab())
    dirty(result)
    await act(async () => { await result.current.handleSaveAll() })
    expect(saveRequest).toHaveBeenCalledTimes(1)
    expect(vi.mocked(saveRequest).mock.calls[0][0].filePath).toBe('/ws/a.yaml')
  })
})

describe('dirty-state notification', () => {
  it('tells the backend when the dirty state changes', async () => {
    const { result } = setup()
    await act(async () => {})
    expect(setHasDirtyTabs).toHaveBeenLastCalledWith(false)
    dirty(result)
    await act(async () => {})
    expect(setHasDirtyTabs).toHaveBeenLastCalledWith(true)
  })

  it('survives the bridge rejecting', async () => {
    vi.mocked(setHasDirtyTabs).mockRejectedValue(new Error('x'))
    const { result } = setup()
    dirty(result)
    await act(async () => {})
    expect(result.current.tabs).toHaveLength(1)
  })
})

describe('history selection', () => {
  it('rebuilds method, headers, body and query params from a history entry', () => {
    const { result } = setup()
    act(() => result.current.handleSelectHistory(entry({
      method: 'POST', url: 'https://x.test/a?x=1&y=a%20b&flag',
      headers: [{ name: 'H', value: 'v' }], bodyType: 'json', bodyContent: '{"a":1}',
    })))
    const t = result.current.activeTab
    expect(t).toMatchObject({ method: 'POST', url: 'https://x.test/a', body: { type: 'json', content: '{"a":1}' } })
    expect(t.queryParams.map(p => [p.name, p.value])).toEqual([['x', '1'], ['y', 'a b'], ['flag', '']])
    expect(t.headers.map(h => h.name)).toEqual(['H'])
  })

  it('does not mark a freshly loaded history entry as dirty', () => {
    const { result } = setup()
    act(() => result.current.handleSelectHistory(entry({ url: 'https://x.test/a?x=1' })))
    expect(result.current.dirtyFilePaths.size).toBe(0)
    expect(result.current.tabs.every(t => t.savedKey !== '')).toBe(true)
  })

  it('falls back to safe defaults for unknown methods and body types', () => {
    const { result } = setup()
    act(() => result.current.handleSelectHistory(entry({ method: 'BREW', bodyType: 'weird', headers: undefined as never })))
    expect(result.current.activeTab).toMatchObject({ method: 'GET', body: { type: 'none' } })
  })

  it('restores variable-referencing auth saved with the entry', () => {
    const { result } = setup()
    const auth = (a: object) => ({ type: '', token: '', username: '', password: '', key: '', value: '', location: '', ...a })
    act(() => result.current.handleSelectHistory(entry({ auth: auth({ type: 'bearer', token: '{{t}}' }) })))
    expect(result.current.activeTab.auth).toEqual({ type: 'bearer', token: '{{t}}' })
    act(() => result.current.handleSelectHistory(entry({ url: 'https://x.test/b', auth: auth({ type: 'basic', username: 'u', password: '{{p}}' }) })))
    expect(result.current.activeTab.auth).toEqual({ type: 'basic', username: 'u', password: '{{p}}' })
    act(() => result.current.handleSelectHistory(entry({ url: 'https://x.test/c', auth: auth({ type: 'apikey', key: 'k', value: '{{v}}' }) })))
    expect(result.current.activeTab.auth).toEqual({ type: 'apikey', key: 'k', value: '{{v}}', location: 'header' })
    act(() => result.current.handleSelectHistory(entry({ url: 'https://x.test/d', auth: auth({ type: 'none' }) })))
    expect(result.current.activeTab.auth).toEqual({ type: 'none' })
  })

  it('reuses a blank active tab, otherwise opens a new one', () => {
    const { result } = setup()
    act(() => result.current.handleSelectHistory(entry()))
    expect(result.current.tabs).toHaveLength(1)
    act(() => result.current.handleSelectHistory(entry({ url: 'https://x.test/other' })))
    expect(result.current.tabs).toHaveLength(2)
  })

  it('does not crash on a malformed percent-escape in a stored URL', () => {
    const { result } = setup()
    expect(() => act(() => result.current.handleSelectHistory(entry({ url: 'https://x.test/a?q=100%' })))).not.toThrow()
  })
})
