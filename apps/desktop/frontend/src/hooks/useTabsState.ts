import { useState, useRef, useEffect } from 'react'
import { saveRequest, loadResponseCache, setHasDirtyTabs } from '@/lib/bridge'
import type { RequestFileIO } from '@/lib/bridge'
import { computeStateKey, createBlankTab, isTabDirty, isBlankTab } from '@/lib/tabState'
import type { TabState } from '@/lib/tabState'
import { authStateToPayload, requestFileIOToState, HTTP_METHODS } from '@/lib/httpUtils'
import type { HttpMethod } from '@/lib/httpUtils'
import type { HistoryEntry } from '@/lib/bridge'
import type { BodyState } from '@/types'

export type TabsStateTuple = { tabs: TabState[]; activeTabId: string }

export function useTabsState(deps: {
  refreshWorkspaceRef: React.RefObject<() => Promise<void>>
}) {
  const { refreshWorkspaceRef } = deps

  const [{ tabs, activeTabId }, setTabsState] = useState<TabsStateTuple>(() => {
    const tab = createBlankTab()
    return { tabs: [tab], activeTabId: tab.id }
  })

  const tabsStateRef = useRef<TabsStateTuple>({ tabs, activeTabId })
  // eslint-disable-next-line react/refs
  tabsStateRef.current = { tabs, activeTabId }

  const activeTab = tabs.find(t => t.id === activeTabId) ?? tabs[0]

  const [closingDirtyTabId, setClosingDirtyTabId] = useState<string | null>(null)
  const [closingDirtyScenarioTabId, setClosingDirtyScenarioTabId] = useState<string | null>(null)
  const [pendingBulkClose, setPendingBulkClose] = useState<{ tabIds: string[]; returnToId?: string } | null>(null)

  const hasAnyDirtyTab = tabs.some(t => isTabDirty(t))
  useEffect(() => {
    setHasDirtyTabs(hasAnyDirtyTab).catch(() => {})
  }, [hasAnyDirtyTab])

  function updateTab(id: string, patch: Partial<TabState>) {
    setTabsState(prev => ({
      ...prev,
      tabs: prev.tabs.map(t => t.id === id ? { ...t, ...patch } : t),
    }))
  }

  function switchTo(id: string) {
    setTabsState(prev => ({ ...prev, activeTabId: id }))
  }

  function removeTab(id: string) {
    setTabsState(prev => {
      if (prev.tabs.length === 1) {
        const blank = createBlankTab()
        return { tabs: [{ ...blank, id: prev.tabs[0].id }], activeTabId: prev.tabs[0].id }
      }
      const idx = prev.tabs.findIndex(t => t.id === id)
      const remaining = prev.tabs.filter(t => t.id !== id)
      const newActiveId = prev.activeTabId === id
        ? (remaining[Math.min(idx, remaining.length - 1)]?.id ?? remaining[0].id)
        : prev.activeTabId
      return { tabs: remaining, activeTabId: newActiveId }
    })
  }

  function handleNewTab() {
    const tab = createBlankTab()
    setTabsState(prev => ({ tabs: [...prev.tabs, tab], activeTabId: tab.id }))
  }

  function handleDuplicateTab() {
    const { tabs: currentTabs, activeTabId: currentActiveId } = tabsStateRef.current
    const tab = currentTabs.find(t => t.id === currentActiveId)
    if (!tab) return
    if (tab.kind === 'scenario') return
    const newTab: TabState = {
      ...tab,
      id: crypto.randomUUID(),
      filePath: null,
      name: tab.name ? `${tab.name} copy` : null,
      savedKey: '',
      response: null,
      loading: false,
      saveError: null,
      editingName: false,
    }
    setTabsState(prev => ({ tabs: [...prev.tabs, newTab], activeTabId: newTab.id }))
  }

  function handleSelectTab(id: string) {
    switchTo(id)
  }

  function handleCloseTab(id: string) {
    const { tabs: currentTabs } = tabsStateRef.current
    const tab = currentTabs.find(t => t.id === id)
    if (!tab) return
    if (tab.kind === 'scenario' && tab.scenarioDirty) {
      setClosingDirtyScenarioTabId(id)
      return
    }
    if (isTabDirty(tab)) {
      setClosingDirtyTabId(id)
    } else {
      removeTab(id)
    }
  }

  function handleCloseOthers(tabId: string) {
    const { tabs: currentTabs } = tabsStateRef.current
    const toClose = currentTabs.filter(t => t.id !== tabId)
    if (toClose.length === 0) return
    const hasDirty = toClose.some(t => isTabDirty(t) || (t.kind === 'scenario' && t.scenarioDirty))
    if (hasDirty) {
      setPendingBulkClose({ tabIds: toClose.map(t => t.id), returnToId: tabId })
      return
    }
    toClose.forEach(t => removeTab(t.id))
    switchTo(tabId)
  }

  function handleCloseToRight(tabId: string) {
    const { tabs: currentTabs } = tabsStateRef.current
    const idx = currentTabs.findIndex(t => t.id === tabId)
    if (idx === -1) return
    const toClose = currentTabs.slice(idx + 1)
    if (toClose.length === 0) return
    const hasDirty = toClose.some(t => isTabDirty(t) || (t.kind === 'scenario' && t.scenarioDirty))
    if (hasDirty) {
      setPendingBulkClose({ tabIds: toClose.map(t => t.id) })
      return
    }
    toClose.forEach(t => removeTab(t.id))
  }

  function confirmBulkClose() {
    if (!pendingBulkClose) return
    const { tabIds, returnToId } = pendingBulkClose
    tabIds.forEach(id => removeTab(id))
    if (returnToId) switchTo(returnToId)
    setPendingBulkClose(null)
  }

  function openFileAsTab(r: RequestFileIO, path: string): string {
    const state = requestFileIOToState(r)
    const timeoutMs = r.timeoutMs ?? 0
    const savedKey = computeStateKey(state.method, state.url, state.queryParams, state.headers, state.body, state.auth, r.title || null, timeoutMs)
    const newTab: TabState = {
      id: crypto.randomUUID(),
      ...state,
      timeoutMs,
      name: r.title || null,
      description: r.description ?? null,
      filePath: path,
      targetDir: null,
      savedKey,
      response: null,
      previousResponse: null,
      responseCachedAt: null,
      loading: false,
      saveError: null,
      editingName: false,
    }
    // Read current state synchronously to determine which ID to return.
    // The updater below mirrors this logic without mutating outer state.
    const currentState = tabsStateRef.current
    const currentActive = currentState.tabs.find(t => t.id === currentState.activeTabId)
    const reuseBlank = !!(currentActive && isBlankTab(currentActive))
    const resolvedId = reuseBlank ? currentState.activeTabId : newTab.id
    setTabsState(prev => {
      const active = prev.tabs.find(t => t.id === prev.activeTabId)
      if (active && isBlankTab(active)) {
        return {
          tabs: prev.tabs.map(t => t.id === prev.activeTabId ? { ...newTab, id: prev.activeTabId } : t),
          activeTabId: prev.activeTabId,
        }
      }
      return { tabs: [...prev.tabs, newTab], activeTabId: newTab.id }
    })
    return resolvedId
  }

  async function loadCacheIntoTab(tabId: string, path: string) {
    try {
      const cached = await loadResponseCache(path)
      if (!cached) return
      const { cachedAt, ...response } = cached
      setTabsState(prev => ({
        ...prev,
        tabs: prev.tabs.map(t =>
          t.id === tabId && t.response === null
            ? { ...t, response, previousResponse: null, responseCachedAt: cachedAt }
            : t,
        ),
      }))
    } catch { /* cache miss is not an error */ }
  }

  function tabToRequestFileIO(tab: TabState, overrideFilePath?: string): RequestFileIO {
    const name = tab.name?.trim() || tab.url.trim().split('/').filter(Boolean).pop() || 'request'
    return {
      schemaVersion: '1',
      title: name,
      method: tab.method,
      url: tab.url.trim(),
      headers: tab.headers.filter(h => h.enabled && h.name.trim()).map(h => ({ name: h.name, value: h.value })),
      queryParams: tab.queryParams.filter(p => p.enabled && p.name.trim()).map(p => ({ name: p.name, value: p.value })),
      body: tab.body.type === 'none' ? undefined : { type: tab.body.type, content: tab.body.content },
      auth: authStateToPayload(tab.auth) ?? undefined,
      timeoutMs: tab.timeoutMs || undefined,
      filePath: overrideFilePath ?? tab.filePath ?? '',
      targetDir: tab.targetDir ?? '',
    }
  }

  async function saveTabById(tabId: string): Promise<boolean> {
    const tab = tabsStateRef.current.tabs.find(t => t.id === tabId)
    if (!tab) return false
    if (tab.kind === 'scenario') return false
    updateTab(tabId, { saveError: null })
    const name = tab.name?.trim() || tab.url.trim().split('/').filter(Boolean).pop() || 'request'
    try {
      const req = tabToRequestFileIO(tab)
      const savedPath = await saveRequest(req)
      if (savedPath) {
        const savedKey = computeStateKey(tab.method, tab.url, tab.queryParams, tab.headers, tab.body, tab.auth, name, tab.timeoutMs)
        updateTab(tabId, { name, filePath: savedPath, savedKey })
        await refreshWorkspaceRef.current?.()
        return true
      }
      return false
    } catch (err) {
      updateTab(tabId, { saveError: String(err).replace(/^Error: /, '') })
      return false
    }
  }

  const saveTabByIdRef = useRef(saveTabById)
  // eslint-disable-next-line react/refs
  saveTabByIdRef.current = saveTabById

  async function handleSaveRequest(): Promise<boolean> {
    return saveTabById(tabsStateRef.current.activeTabId)
  }

  async function handleSaveAsRequest(): Promise<boolean> {
    const { activeTabId: currentActiveId } = tabsStateRef.current
    const tab = tabsStateRef.current.tabs.find(t => t.id === currentActiveId)
    if (!tab) return false
    updateTab(currentActiveId, { saveError: null })
    const name = tab.name?.trim() || tab.url.trim().split('/').filter(Boolean).pop() || 'request'
    try {
      const req = tabToRequestFileIO(tab, '')
      const savedPath = await saveRequest(req)
      if (savedPath) {
        const savedKey = computeStateKey(tab.method, tab.url, tab.queryParams, tab.headers, tab.body, tab.auth, name, tab.timeoutMs)
        updateTab(currentActiveId, { name, filePath: savedPath, savedKey })
        await refreshWorkspaceRef.current?.()
        return true
      }
      return false
    } catch (err) {
      updateTab(currentActiveId, { saveError: String(err).replace(/^Error: /, '') })
      return false
    }
  }

  function handleSelectHistory(entry: HistoryEntry) {
    const method = (HTTP_METHODS as readonly string[]).includes(entry.method)
      ? entry.method as HttpMethod : 'GET'
    const headers = (entry.headers ?? []).map(h => ({
      id: crypto.randomUUID(), name: h.name, value: h.value, enabled: true,
    }))
    const bodyType = (['none', 'json', 'raw', 'form-urlencoded', 'multipart'] as const).includes(entry.bodyType as never)
      ? entry.bodyType as BodyState['type'] : 'none'
    const body: BodyState = { type: bodyType, content: entry.bodyContent ?? '' }
    const savedKey = computeStateKey(method, entry.url, [], headers, body, { type: 'none' }, null, 0)
    const newTab: TabState = {
      id: crypto.randomUUID(),
      method,
      url: entry.url,
      queryParams: [],
      headers,
      body,
      auth: { type: 'none' },
      timeoutMs: 0,
      name: null,
      filePath: null,
      targetDir: null,
      savedKey,
      response: null,
      previousResponse: null,
      responseCachedAt: null,
      loading: false,
      saveError: null,
      editingName: false,
    }
    setTabsState(prev => {
      const active = prev.tabs.find(t => t.id === prev.activeTabId)
      if (active && isBlankTab(active)) {
        return {
          tabs: prev.tabs.map(t => t.id === prev.activeTabId ? { ...newTab, id: prev.activeTabId } : t),
          activeTabId: prev.activeTabId,
        }
      }
      return { tabs: [...prev.tabs, newTab], activeTabId: newTab.id }
    })
  }

  function handleNewRequestInFolder(folderPath: string) {
    const tab = createBlankTab(folderPath)
    setTabsState(prev => ({ tabs: [...prev.tabs, tab], activeTabId: tab.id }))
  }

  function openScenarioAsTab(scenarioPath: string, scenarioName: string): string {
    // If an identical scenario tab already exists, switch to it
    const existing = tabsStateRef.current.tabs.find(
      t => t.kind === 'scenario' && t.scenarioPath === scenarioPath
    )
    if (existing) {
      switchTo(existing.id)
      return existing.id
    }
    const blankKey = computeStateKey('GET', '', [], [], { type: 'none', content: '' }, { type: 'none' }, null, 0)
    const newTab: TabState = {
      id: crypto.randomUUID(),
      kind: 'scenario',
      scenarioPath,
      scenarioName,
      scenarioResult: null,
      scenarioRunning: false,
      // request fields kept at defaults
      method: 'GET',
      url: '',
      queryParams: [],
      headers: [],
      body: { type: 'none', content: '' },
      auth: { type: 'none' },
      timeoutMs: 0,
      name: scenarioName,
      filePath: null,
      targetDir: null,
      savedKey: blankKey,
      response: null,
      previousResponse: null,
      responseCachedAt: null,
      loading: false,
      saveError: null,
      editingName: false,
    }
    const currentState = tabsStateRef.current
    const currentActive = currentState.tabs.find(t => t.id === currentState.activeTabId)
    const reuseBlank = !!(currentActive && isBlankTab(currentActive))
    const resolvedId = reuseBlank ? currentState.activeTabId : newTab.id
    setTabsState(prev => {
      const active = prev.tabs.find(t => t.id === prev.activeTabId)
      if (active && isBlankTab(active)) {
        return {
          tabs: prev.tabs.map(t => t.id === prev.activeTabId ? { ...newTab, id: prev.activeTabId } : t),
          activeTabId: prev.activeTabId,
        }
      }
      return { tabs: [...prev.tabs, newTab], activeTabId: newTab.id }
    })
    return resolvedId
  }

  const dirtyFilePaths = new Set(tabs.filter(t => t.filePath && isTabDirty(t)).map(t => t.filePath!))

  async function handleSaveAll(): Promise<void> {
    const dirtyTabs = tabsStateRef.current.tabs.filter(t => t.filePath && isTabDirty(t))
    for (const tab of dirtyTabs) {
      await saveTabById(tab.id)
    }
  }

  return {
    tabs, activeTabId, activeTab,
    setTabsState, tabsStateRef,
    updateTab, switchTo, removeTab,
    handleNewTab, handleDuplicateTab, handleSelectTab, handleCloseTab, handleCloseOthers, handleCloseToRight, confirmBulkClose,
    openFileAsTab, loadCacheIntoTab,
    saveTabById, saveTabByIdRef,
    handleSaveRequest, handleSaveAsRequest, handleSaveAll,
    handleSelectHistory, handleNewRequestInFolder,
    openScenarioAsTab,
    closingDirtyTabId, setClosingDirtyTabId,
    closingDirtyScenarioTabId, setClosingDirtyScenarioTabId,
    pendingBulkClose, setPendingBulkClose,
    dirtyFilePaths,
  }
}
