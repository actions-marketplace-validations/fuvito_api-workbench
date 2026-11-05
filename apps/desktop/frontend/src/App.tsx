import { useState, useEffect, useCallback, useRef } from 'react'
import { FolderOpen, Save, FilePlus, Settings, Pencil, CopyPlus } from 'lucide-react'
import { MenuBar } from '@/components/MenuBar'
import type { MenuDef } from '@/components/MenuBar'
import { TabBar } from '@/components/TabBar'
import {
  executeRequest, getHistory, openWorkspace, newWorkspace, closeWorkspace, getWorkspacePath, listRequests,
  loadRequest, saveRequest, openRequest, deleteRequest, setHasDirtyTabs,
  createFolder, deleteFolder, renameFolder, renameRequest,
  listEnvironments, selectEnvironment, getActiveEnvironment,
  importOpenAPISpec, getSettings, saveSettings,
  startDemo, stopDemo, saveResponseCache, loadResponseCache,
  type HistoryEntry, type FileNode,
  type EnvInfo, type RequestFileIO, type AppSettings,
} from '@/lib/bridge'
import { SettingsPanel } from '@/components/SettingsPanel'
import { EnvironmentEditor } from '@/components/EnvironmentEditor'
import { cn } from '@/lib/utils'
import { RequestEditor } from '@/components/RequestEditor'
import { ResponseViewer } from '@/components/ResponseViewer'
import { Sidebar } from '@/components/Sidebar'
import type { SidebarTab } from '@/components/Sidebar'
import { SnippetButton } from '@/components/SnippetButton'
import { AboutDialog } from '@/components/AboutDialog'
import {
  DeleteRequestDialog,
  DeleteFolderDialog,
  NewFolderDialog,
  CloseDirtyTabDialog,
} from '@/components/ConfirmDialogs'
import {
  HTTP_METHODS, METHOD_COLOR, authStateToPayload, requestFileIOToState,
} from '@/lib/httpUtils'
import type { HttpMethod } from '@/lib/httpUtils'
import { parseCurl } from '@/lib/curlImport'
import { computeStateKey, createBlankTab, isTabDirty, isBlankTab } from '@/lib/tabState'
import type { TabState } from '@/lib/tabState'
import type { KVPair } from '@/types'

export default function App() {
  // --- Tabs ---
  const [{ tabs, activeTabId }, setTabsState] = useState(() => {
    const tab = createBlankTab()
    return { tabs: [tab] as TabState[], activeTabId: tab.id }
  })

  const tabsStateRef = useRef({ tabs, activeTabId })
  tabsStateRef.current = { tabs, activeTabId }

  // --- Resize state ---
  const [sidebarWidth, setSidebarWidth] = useState(224)
  const [responseWidth, setResponseWidth] = useState(0)
  const draggingRef = useRef<'sidebar' | 'response' | null>(null)

  // --- Global UI state ---
  const [settings, setSettings] = useState<AppSettings>({ theme: 'dark', fontSize: 13, defaultTimeoutMs: 30000, proxyUrl: '', tlsSkipVerify: false })
  const [showSettings, setShowSettings] = useState(false)
  const [showEnvEditor, setShowEnvEditor] = useState(false)
  const [editingEnvPath, setEditingEnvPath] = useState<string | null>(null)
  const [workspacePath, setWorkspacePath] = useState<string>('')
  const [fileTree, setFileTree] = useState<FileNode[]>([])
  const [environments, setEnvironments] = useState<EnvInfo[]>([])
  const [activeEnv, setActiveEnv] = useState<EnvInfo | null>(null)
  const [sidebarTab, setSidebarTab] = useState<SidebarTab>('workspace')
  const [importing, setImporting] = useState(false)
  const [history, setHistory] = useState<HistoryEntry[]>([])
  const [closingDirtyTabId, setClosingDirtyTabId] = useState<string | null>(null)
  const [deletingFilePath, setDeletingFilePath] = useState<string | null>(null)
  const [showAbout, setShowAbout] = useState(false)
  const [demoRunning, setDemoRunning] = useState(false)
  const [demoPort, setDemoPort] = useState(0)
  const [demoStarting, setDemoStarting] = useState(false)
  const [deletingFolderPath, setDeletingFolderPath] = useState<string | null>(null)
  const [newFolderDialog, setNewFolderDialog] = useState<{ parentPath: string; parentLabel: string } | null>(null)
  const [newFolderName, setNewFolderName] = useState('')
  const [newFolderError, setNewFolderError] = useState('')

  // --- Derived ---
  const activeTab = tabs.find(t => t.id === activeTabId) ?? tabs[0]

  // --- Tab helpers ---

  function updateTab(id: string, patch: Partial<TabState>) {
    setTabsState(prev => ({
      ...prev,
      tabs: prev.tabs.map(t => t.id === id ? { ...t, ...patch } : t),
    }))
  }

  function switchTo(id: string) {
    setTabsState(prev => ({ ...prev, activeTabId: id }))
  }

  function openFileAsTab(r: RequestFileIO, path: string): string {
    const state = requestFileIOToState(r)
    const savedKey = computeStateKey(state.method, state.url, state.queryParams, state.headers, state.body, state.auth, r.name || null)
    const newTab: TabState = {
      id: crypto.randomUUID(),
      ...state,
      name: r.name || null,
      filePath: path,
      targetDir: null,
      savedKey,
      response: null,
      responseCachedAt: null,
      loading: false,
      saveError: null,
      editingName: false,
    }
    let resolvedId = newTab.id
    setTabsState(prev => {
      const active = prev.tabs.find(t => t.id === prev.activeTabId)
      if (active && isBlankTab(active)) {
        resolvedId = prev.activeTabId
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
            ? { ...t, response, responseCachedAt: cachedAt }
            : t,
        ),
      }))
    } catch { /* cache miss is not an error */ }
  }

  // ---- Init ----

  const refreshWorkspace = useCallback(async () => {
    try {
      const [path, tree, envs, activeE] = await Promise.all([
        getWorkspacePath() as Promise<string>,
        listRequests(),
        listEnvironments(),
        getActiveEnvironment(),
      ])
      setWorkspacePath(path ?? '')
      setFileTree(tree ?? [])
      setEnvironments(envs ?? [])
      setActiveEnv(activeE)
    } catch {
      // best-effort
    }
  }, [])

  const refreshHistory = useCallback(async () => {
    try { setHistory(await getHistory()) } catch { /* best-effort */ }
  }, [])

  useEffect(() => {
    refreshWorkspace()
    refreshHistory()
    getSettings().then(s => {
      setSettings(s)
      applySettings(s)
    }).catch(() => {})
  }, [refreshWorkspace, refreshHistory])

  const hasAnyDirtyTab = tabs.some(t => isTabDirty(t))
  useEffect(() => {
    setHasDirtyTabs(hasAnyDirtyTab).catch(() => {})
  }, [hasAnyDirtyTab])

  // ---- Resize ----

  useEffect(() => {
    function onMouseMove(e: MouseEvent) {
      if (draggingRef.current === 'sidebar') {
        setSidebarWidth(prev => Math.max(160, Math.min(480, prev + e.movementX)))
      } else if (draggingRef.current === 'response') {
        setResponseWidth(prev => {
          const current = prev > 0 ? prev : window.innerWidth * 0.45
          return Math.max(200, Math.min(window.innerWidth - 400, current - e.movementX))
        })
      }
    }
    function onMouseUp() {
      draggingRef.current = null
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
    return () => {
      document.removeEventListener('mousemove', onMouseMove)
      document.removeEventListener('mouseup', onMouseUp)
    }
  }, [])

  // ---- Keyboard shortcuts ----

  const saveTabByIdRef = useRef<((id: string) => Promise<boolean>) | null>(null)
  const handleSendRef = useRef<(() => void) | null>(null)
  const handleOpenRequestRef = useRef<(() => void) | null>(null)

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault()
        saveTabByIdRef.current?.(tabsStateRef.current.activeTabId)
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
        e.preventDefault()
        handleNewTab()
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'w') {
        e.preventDefault()
        handleCloseTab(tabsStateRef.current.activeTabId)
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault()
        handleSendRef.current?.()
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'o') {
        e.preventDefault()
        handleOpenRequestRef.current?.()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, []) // registered once; reads from refs

  // ---- Tab operations ----

  function handleNewTab() {
    const tab = createBlankTab()
    setTabsState(prev => ({ tabs: [...prev.tabs, tab], activeTabId: tab.id }))
  }

  function handleDuplicateTab() {
    const { tabs: currentTabs, activeTabId: currentActiveId } = tabsStateRef.current
    const tab = currentTabs.find(t => t.id === currentActiveId)
    if (!tab) return
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
    if (isTabDirty(tab)) {
      setClosingDirtyTabId(id)
    } else {
      removeTab(id)
    }
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

  // ---- Workspace operations ----

  async function handleOpenWorkspace() {
    try {
      const path = await openWorkspace()
      if (path) {
        setWorkspacePath(path)
        await refreshWorkspace()
      }
    } catch (err) {
      console.error('open workspace:', err)
    }
  }

  async function handleNewWorkspace() {
    try {
      const path = await newWorkspace()
      if (path) {
        setWorkspacePath(path)
        await refreshWorkspace()
      }
    } catch (err) {
      console.error('new workspace:', err)
    }
  }

  async function handleCloseWorkspace() {
    try {
      const path = workspacePath
      await closeWorkspace()
      setWorkspacePath('')
      setFileTree([])
      setEnvironments([])
      setActiveEnv(null)
      if (path) {
        setTabsState(prev => {
          const norm = (p: string) => p.replace(/\\/g, '/')
          const prefix = norm(path).replace(/\/?$/, '/')
          const remaining = prev.tabs.filter(t => !norm(t.filePath ?? '').startsWith(prefix))
          if (remaining.length === prev.tabs.length) return prev
          if (remaining.length === 0) {
            const blank = createBlankTab()
            return { tabs: [blank], activeTabId: blank.id }
          }
          const activeStillOpen = remaining.some(t => t.id === prev.activeTabId)
          return {
            tabs: remaining,
            activeTabId: activeStillOpen ? prev.activeTabId : remaining[remaining.length - 1].id,
          }
        })
      }
    } catch (err) {
      console.error('close workspace:', err)
    }
  }

  async function handleSelectFile(path: string) {
    const existing = tabsStateRef.current.tabs.find(t => t.filePath === path)
    if (existing) {
      switchTo(existing.id)
      return
    }
    try {
      const r = await loadRequest(path)
      if (!r) return
      const tabId = openFileAsTab(r, path)
      loadCacheIntoTab(tabId, path)
    } catch (err) {
      console.error('load request:', err)
    }
  }

  async function handleOpenRequest() {
    try {
      const r = await openRequest()
      if (!r || !r.filePath) return
      const existing = tabsStateRef.current.tabs.find(t => t.filePath === r.filePath)
      if (existing) { switchTo(existing.id); return }
      const tabId = openFileAsTab(r, r.filePath)
      loadCacheIntoTab(tabId, r.filePath)
    } catch (err) {
      console.error('open request:', err)
    }
  }

  handleOpenRequestRef.current = handleOpenRequest

  function handleNewRequest() {
    handleNewTab()
  }

  async function saveTabById(tabId: string): Promise<boolean> {
    const tab = tabsStateRef.current.tabs.find(t => t.id === tabId)
    if (!tab) return false
    updateTab(tabId, { saveError: null })
    const name = tab.name?.trim() || tab.url.trim().split('/').filter(Boolean).pop() || 'request'
    try {
      const req: RequestFileIO = {
        schemaVersion: '1',
        name,
        method: tab.method,
        url: tab.url.trim(),
        headers: tab.headers.filter(h => h.enabled && h.name.trim()).map(h => ({ name: h.name, value: h.value })),
        queryParams: tab.queryParams.filter(p => p.enabled && p.name.trim()).map(p => ({ name: p.name, value: p.value })),
        body: tab.body.type === 'none' ? undefined : { type: tab.body.type, content: tab.body.content },
        auth: authStateToPayload(tab.auth) ?? undefined,
        filePath: tab.filePath ?? '',
        targetDir: tab.targetDir ?? '',
      }
      const savedPath = await saveRequest(req)
      if (savedPath) {
        const savedKey = computeStateKey(tab.method, tab.url, tab.queryParams, tab.headers, tab.body, tab.auth, name)
        updateTab(tabId, { name, filePath: savedPath, savedKey })
        await refreshWorkspace()
        return true
      }
      return false
    } catch (err) {
      updateTab(tabId, { saveError: String(err).replace(/^Error: /, '') })
      return false
    }
  }

  saveTabByIdRef.current = saveTabById

  async function handleSaveRequest(): Promise<boolean> {
    return saveTabById(activeTabId)
  }

  // ---- Environment operations ----

  async function handleEnvChange(path: string) {
    try {
      await selectEnvironment(path)
      const envs = await listEnvironments()
      setEnvironments(envs ?? [])
      const active = path ? envs.find(e => e.path === path) ?? null : null
      setActiveEnv(active)
    } catch (err) {
      console.error('select environment:', err)
    }
  }

  // ---- History ----

  function handleSelectHistory(entry: HistoryEntry) {
    const method = (HTTP_METHODS as readonly string[]).includes(entry.method)
      ? entry.method as HttpMethod : 'GET'
    const savedKey = computeStateKey(method, entry.url, [], [], { type: 'none', content: '' }, { type: 'none' }, null)
    const newTab: TabState = {
      id: crypto.randomUUID(),
      method,
      url: entry.url,
      queryParams: [],
      headers: [],
      body: { type: 'none', content: '' },
      auth: { type: 'none' },
      name: null,
      filePath: null,
      targetDir: null,
      savedKey,
      response: null,
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

  // ---- Settings ----

  function applySettings(s: AppSettings) {
    const root = document.documentElement
    root.style.setProperty('--font-size-code', `${s.fontSize}px`)
    if (s.theme === 'light') {
      root.classList.add('light')
      root.classList.remove('dark')
    } else {
      root.classList.add('dark')
      root.classList.remove('light')
    }
  }

  async function handleToggleTheme() {
    const next = { ...settings, theme: settings.theme === 'dark' ? 'light' : 'dark' }
    await handleSaveSettings(next as AppSettings)
  }

  async function handleSaveSettings(s: AppSettings) {
    try {
      const saved = await saveSettings(s)
      setSettings(saved)
      applySettings(saved)
    } catch (err) {
      console.error('save settings:', err)
    }
  }

  // ---- OpenAPI import ----

  async function handleImportOpenAPI() {
    setImporting(true)
    try {
      const result = await importOpenAPISpec()
      if (result.error) {
        console.error('import error:', result.error)
      } else {
        await refreshWorkspace()
      }
    } catch (err) {
      console.error('import failed:', err)
    } finally {
      setImporting(false)
    }
  }

  // ---- Duplicate request file ----

  async function handleDuplicateFile(path: string) {
    try {
      const req = await loadRequest(path)
      const lastSep = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'))
      const targetDir = lastSep !== -1 ? path.slice(0, lastSep) : ''
      await saveRequest({ ...req, name: (req.name || 'request') + ' Copy', filePath: '', targetDir })
      await refreshWorkspace()
    } catch (err) {
      console.error('duplicate request:', err)
    }
  }

  // ---- Delete request file ----

  function handleDeleteFile(path: string) {
    setDeletingFilePath(path)
  }

  async function confirmDeleteFile() {
    if (!deletingFilePath) return
    const path = deletingFilePath
    setDeletingFilePath(null)
    try {
      await deleteRequest(path)
      setTabsState(prev => {
        const remaining = prev.tabs.filter(t => t.filePath !== path)
        if (remaining.length === prev.tabs.length) return prev
        if (remaining.length === 0) {
          const blank = createBlankTab()
          return { tabs: [blank], activeTabId: blank.id }
        }
        const activeStillExists = remaining.some(t => t.id === prev.activeTabId)
        return {
          tabs: remaining,
          activeTabId: activeStillExists ? prev.activeTabId : remaining[remaining.length - 1].id,
        }
      })
      await refreshWorkspace()
    } catch (err) {
      console.error('delete request:', err)
    }
  }

  // ---- Folder operations ----

  function handleCreateFolder(parentPath: string) {
    const parentLabel = parentPath
      ? parentPath.replace(/\\/g, '/').split('/').pop() ?? 'requests'
      : 'requests'
    setNewFolderDialog({ parentPath, parentLabel })
    setNewFolderName('')
    setNewFolderError('')
  }

  async function confirmCreateFolder() {
    if (!newFolderDialog || !newFolderName.trim()) return
    try {
      await createFolder(newFolderDialog.parentPath, newFolderName.trim())
      setNewFolderDialog(null)
      setNewFolderName('')
      setNewFolderError('')
      await refreshWorkspace()
    } catch (err) {
      setNewFolderError(err instanceof Error ? err.message : String(err))
    }
  }

  function handleDeleteFolder(path: string) {
    setDeletingFolderPath(path)
  }

  async function confirmDeleteFolder() {
    if (!deletingFolderPath) return
    const path = deletingFolderPath
    setDeletingFolderPath(null)
    try {
      await deleteFolder(path)
      await refreshWorkspace()
    } catch (err) {
      console.error('delete folder:', err)
    }
  }

  function handleNewRequestInFolder(folderPath: string) {
    const tab = createBlankTab(folderPath)
    setTabsState(prev => ({ tabs: [...prev.tabs, tab], activeTabId: tab.id }))
  }

  // ---- Rename operations ----

  function rebasePath(oldBase: string, newBase: string, p: string | null): string | null {
    if (!p) return p
    if (p === oldBase) return newBase
    if (p.startsWith(oldBase + '\\') || p.startsWith(oldBase + '/')) return newBase + p.slice(oldBase.length)
    return p
  }

  async function handleRenameFile(path: string, newName: string) {
    try {
      const newPath = await renameRequest(path, newName)
      if (!newPath) return
      setTabsState(prev => ({
        ...prev,
        tabs: prev.tabs.map(t => {
          if (t.filePath !== path) return t
          const savedKey = computeStateKey(t.method, t.url, t.queryParams, t.headers, t.body, t.auth, newName)
          return { ...t, name: newName, filePath: newPath, savedKey }
        }),
      }))
      await refreshWorkspace()
    } catch (err) {
      console.error('rename request:', err)
    }
  }

  async function handleRenameFolder(path: string, newName: string) {
    try {
      const newPath = await renameFolder(path, newName)
      if (!newPath) return
      setTabsState(prev => ({
        ...prev,
        tabs: prev.tabs.map(t => {
          const fp = rebasePath(path, newPath, t.filePath)
          const td = rebasePath(path, newPath, t.targetDir)
          if (fp === t.filePath && td === t.targetDir) return t
          return { ...t, filePath: fp, targetDir: td }
        }),
      }))
      await refreshWorkspace()
    } catch (err) {
      console.error('rename folder:', err)
    }
  }

  // ---- URL paste: curl import or query-param extraction ----

  function handleUrlPaste(raw: string) {
    try {
      // curl import
      const curlResult = parseCurl(raw)
      if (curlResult) {
        updateTab(activeTabId, {
          method: curlResult.method,
          url: curlResult.url,
          queryParams: curlResult.queryParams,
          headers: curlResult.headers,
          body: curlResult.body,
          auth: curlResult.auth,
        })
        return
      }

      // Strip query params from pasted URL
      const qIdx = raw.indexOf('?')
      if (qIdx === -1) return
      const baseUrl = raw.slice(0, qIdx)
      const search = raw.slice(qIdx + 1)
      const parsed = new URLSearchParams(search)
      const extracted: KVPair[] = []
      parsed.forEach((value, name) => {
        extracted.push({ id: crypto.randomUUID(), name, value, enabled: true })
      })
      updateTab(activeTabId, { url: baseUrl })
      if (extracted.length > 0) {
        const names = new Set(extracted.map(p => p.name.toLowerCase()))
        const kept = activeTab.queryParams.filter(p => !names.has(p.name.toLowerCase()))
        updateTab(activeTabId, { queryParams: [...kept, ...extracted] })
      }
    } catch {
      updateTab(activeTabId, { url: raw })
    }
  }

  // ---- Request execution ----

  async function handleSend() {
    if (!activeTab.url.trim() || activeTab.loading) return
    const tabId = activeTabId
    updateTab(tabId, { loading: true, response: null, responseCachedAt: null })
    try {
      const result = await executeRequest({
        method: activeTab.method,
        url: activeTab.url.trim(),
        headers: activeTab.headers.filter(h => h.enabled && h.name.trim()).map(h => ({ name: h.name, value: h.value })),
        queryParams: activeTab.queryParams.filter(p => p.enabled && p.name.trim()).map(p => ({ name: p.name, value: p.value })),
        body: activeTab.body.type === 'none' ? undefined : { type: activeTab.body.type, content: activeTab.body.content },
        auth: authStateToPayload(activeTab.auth),
        timeoutMs: 0,
      })
      updateTab(tabId, { response: result })
      const filePath = tabsStateRef.current.tabs.find(t => t.id === tabId)?.filePath
      if (filePath && !result.error) {
        saveResponseCache(filePath, result).catch(() => {})
      }
    } catch (err) {
      updateTab(tabId, { response: { statusCode: 0, status: '', headers: [], body: '', durationMs: 0, size: 0, timing: { dnsMs: 0, connectMs: 0, tlsMs: 0, ttfbMs: 0, transferMs: 0 }, error: String(err) } })
    } finally {
      updateTab(tabId, { loading: false })
      refreshHistory()
    }
  }

  handleSendRef.current = handleSend

  // ---- Demo server ----

  async function handleStartDemo() {
    if (demoRunning || demoStarting) return
    setDemoStarting(true)
    try {
      const result = await startDemo()
      if (result.error) {
        console.error('demo start failed:', result.error)
        return
      }
      setDemoRunning(true)
      setDemoPort(result.port)
      await refreshWorkspace()
    } catch (err) {
      console.error('demo start error:', err)
    } finally {
      setDemoStarting(false)
    }
  }

  async function handleStopDemo() {
    try {
      const port = demoPort
      await stopDemo()
      setDemoRunning(false)
      setDemoPort(0)
      // Close demo tabs, then restore the previous workspace state from Go.
      if (port > 0) {
        setTabsState(prev => {
          const prefix = `http://localhost:${port}`
          const remaining = prev.tabs.filter(t => !t.url.startsWith(prefix))
          if (remaining.length === prev.tabs.length) return prev
          if (remaining.length === 0) {
            const blank = createBlankTab()
            return { tabs: [blank], activeTabId: blank.id }
          }
          const activeStillOpen = remaining.some(t => t.id === prev.activeTabId)
          return {
            tabs: remaining,
            activeTabId: activeStillOpen ? prev.activeTabId : remaining[remaining.length - 1].id,
          }
        })
      }
      await refreshWorkspace()
    } catch (err) {
      console.error('demo stop error:', err)
    }
  }

  // ---- Menus ----

  const menus: MenuDef[] = [
    {
      label: 'File',
      items: [
        { kind: 'action', label: 'New Tab',           shortcut: 'Ctrl+N', onClick: handleNewTab },
        { kind: 'action', label: 'Open Request…',     shortcut: 'Ctrl+O', onClick: handleOpenRequest },
        { kind: 'action', label: 'Save Request',      shortcut: 'Ctrl+S', onClick: handleSaveRequest },
        { kind: 'separator' },
        { kind: 'action', label: 'Import OpenAPI Spec…', onClick: handleImportOpenAPI, disabled: !workspacePath },
        { kind: 'separator' },
        { kind: 'action', label: 'Open Workspace…',   onClick: handleOpenWorkspace },
        { kind: 'action', label: 'New Workspace…',    onClick: handleNewWorkspace },
        { kind: 'action', label: 'Close Workspace',   onClick: handleCloseWorkspace, disabled: !workspacePath },
      ],
    },
    {
      label: 'View',
      items: [
        { kind: 'action', label: settings.theme === 'dark' ? 'Switch to Light Theme' : 'Switch to Dark Theme', onClick: handleToggleTheme },
        { kind: 'separator' },
        { kind: 'action', label: 'Settings…', onClick: () => setShowSettings(true) },
      ],
    },
    {
      label: 'Help',
      items: [
        {
          kind: 'action',
          label: demoStarting ? 'Starting Demo…' : demoRunning ? 'Stop Demo Server' : 'Start Demo Server',
          onClick: demoRunning ? handleStopDemo : handleStartDemo,
          disabled: demoStarting,
        },
        { kind: 'separator' },
        { kind: 'action', label: 'About ApiDesk', onClick: () => setShowAbout(true) },
      ],
    },
  ]

  const tabInfos = tabs.map(t => ({
    id: t.id,
    name: t.name,
    method: t.method,
    isDirty: isTabDirty(t),
  }))

  const closingTab = closingDirtyTabId ? tabs.find(t => t.id === closingDirtyTabId) : null

  // ---- Render ----

  return (
    <div className="flex flex-col h-screen w-screen overflow-hidden">
      <MenuBar menus={menus} />

      {deletingFilePath && (
        <DeleteRequestDialog
          filePath={deletingFilePath}
          onCancel={() => setDeletingFilePath(null)}
          onConfirm={confirmDeleteFile}
        />
      )}

      {showAbout && <AboutDialog onClose={() => setShowAbout(false)} />}

      {newFolderDialog && (
        <NewFolderDialog
          parentLabel={newFolderDialog.parentLabel}
          value={newFolderName}
          onChange={v => { setNewFolderName(v); setNewFolderError('') }}
          onCancel={() => { setNewFolderDialog(null); setNewFolderError('') }}
          onConfirm={confirmCreateFolder}
          error={newFolderError}
        />
      )}

      {deletingFolderPath && (
        <DeleteFolderDialog
          folderPath={deletingFolderPath}
          onCancel={() => setDeletingFolderPath(null)}
          onConfirm={confirmDeleteFolder}
        />
      )}

      {closingTab && (
        <CloseDirtyTabDialog
          tabName={closingTab.name}
          onCancel={() => setClosingDirtyTabId(null)}
          onDiscard={() => { setClosingDirtyTabId(null); removeTab(closingTab.id) }}
          onSave={async () => {
            const ok = await saveTabById(closingTab.id)
            if (ok) { setClosingDirtyTabId(null); removeTab(closingTab.id) }
          }}
        />
      )}

      <div className="flex flex-1 overflow-hidden">
        {showSettings && (
          <SettingsPanel
            settings={settings}
            onSave={handleSaveSettings}
            onPreview={applySettings}
            onClose={() => { applySettings(settings); setShowSettings(false) }}
          />
        )}
        {showEnvEditor && (
          <EnvironmentEditor
            envPath={editingEnvPath}
            workspacePath={workspacePath}
            onClose={() => setShowEnvEditor(false)}
            onSaved={async saved => {
              setShowEnvEditor(false)
              await refreshWorkspace()
              try { await selectEnvironment(saved.path) } catch { /* best-effort */ }
              await refreshWorkspace()
            }}
            onDeleted={async () => {
              setShowEnvEditor(false)
              await refreshWorkspace()
            }}
          />
        )}

        <Sidebar
          width={sidebarWidth}
          tab={sidebarTab}
          onTabChange={setSidebarTab}
          workspacePath={workspacePath}
          fileTree={fileTree}
          history={history}
          onOpenWorkspace={handleOpenWorkspace}
          onNewWorkspace={handleNewWorkspace}
          onSelectFile={handleSelectFile}
          onDeleteFile={handleDeleteFile}
          onDuplicateFile={handleDuplicateFile}
          onDeleteFolder={handleDeleteFolder}
          onNewRequestInFolder={handleNewRequestInFolder}
          onCreateFolder={handleCreateFolder}
          onRenameFile={handleRenameFile}
          onRenameFolder={handleRenameFolder}
          onSelectHistory={handleSelectHistory}
          onImportOpenAPI={handleImportOpenAPI}
          importing={importing}
          activeFilePath={activeTab.filePath ?? undefined}
          dirtyFilePaths={new Set(tabs.filter(t => t.filePath && isTabDirty(t)).map(t => t.filePath!))}
        />

        {/* Sidebar resize handle */}
        <div
          className="w-1 shrink-0 cursor-col-resize hover:bg-[var(--color-accent)] transition-colors"
          onMouseDown={e => {
            e.preventDefault()
            draggingRef.current = 'sidebar'
            document.body.style.cursor = 'col-resize'
            document.body.style.userSelect = 'none'
          }}
        />

        <main className="flex-1 flex flex-col overflow-hidden min-w-0">
          <TabBar
            tabs={tabInfos}
            activeTabId={activeTabId}
            onSelect={handleSelectTab}
            onClose={handleCloseTab}
            onNew={handleNewTab}
          />

          {/* Request header bar */}
          <header className="flex items-center gap-2 px-3 py-1.5 border-b border-[var(--color-border)] bg-[var(--color-card)] shrink-0">
            <div className="flex items-center gap-1.5 min-w-0 flex-1">
              {isTabDirty(activeTab) && (
                <span className="shrink-0 w-2 h-2 rounded-full bg-amber-400" title="Unsaved changes" />
              )}
              {activeTab.name !== null && (
                <span className={cn('text-xs font-mono font-bold shrink-0', METHOD_COLOR[activeTab.method])}>
                  {activeTab.method}
                </span>
              )}
              {activeTab.editingName ? (
                <input
                  autoFocus
                  type="text"
                  value={activeTab.name ?? ''}
                  onChange={e => updateTab(activeTabId, { name: e.target.value })}
                  onBlur={() => {
                    if (!activeTab.name?.trim()) updateTab(activeTabId, { name: null, editingName: false })
                    else updateTab(activeTabId, { editingName: false })
                  }}
                  onKeyDown={e => {
                    if (e.key === 'Enter' || e.key === 'Escape') {
                      if (!activeTab.name?.trim()) updateTab(activeTabId, { name: null, editingName: false })
                      else updateTab(activeTabId, { editingName: false })
                    }
                  }}
                  className={cn(
                    'text-xs font-medium bg-transparent outline-none',
                    'border-b border-[var(--color-accent)]',
                    'text-[var(--color-foreground)] w-40',
                  )}
                />
              ) : activeTab.name !== null ? (
                <button
                  type="button"
                  onClick={() => updateTab(activeTabId, { editingName: true, saveError: null })}
                  title="Click to rename"
                  className="text-xs font-medium text-[var(--color-foreground)] hover:text-[var(--color-accent)] truncate transition-colors text-left"
                >
                  {activeTab.name}
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => updateTab(activeTabId, { name: '', editingName: true })}
                  title="Click to name this request"
                  className="text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
                >
                  New Request
                </button>
              )}
              {activeTab.saveError && (
                <span className="text-label text-[var(--color-destructive)] truncate ml-1">
                  {activeTab.saveError}
                </span>
              )}
            </div>

            <div className="ml-auto flex items-center gap-2 shrink-0">
              <button
                type="button"
                onClick={handleOpenRequest}
                title="Open request file"
                className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
              >
                <FolderOpen size={13} />
              </button>
              <button
                type="button"
                onClick={handleSaveRequest}
                title="Save request (Ctrl+S)"
                className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
              >
                <Save size={13} />
              </button>
              <button
                type="button"
                onClick={handleNewRequest}
                title="New request tab (Ctrl+N)"
                className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
              >
                <FilePlus size={13} />
              </button>
              <button
                type="button"
                onClick={handleDuplicateTab}
                title="Duplicate tab"
                className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
              >
                <CopyPlus size={13} />
              </button>
              <SnippetButton tab={activeTab} />
              <button
                type="button"
                onClick={() => setShowSettings(true)}
                title="Settings"
                className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
              >
                <Settings size={13} />
              </button>

              {environments.length > 0 && (
                <div className="w-px h-4 bg-[var(--color-border)]" />
              )}

              {environments.length > 0 && (
                <select
                  value={activeEnv?.path ?? ''}
                  onChange={e => handleEnvChange(e.target.value)}
                  className={cn(
                    'bg-[var(--color-muted)] text-xs rounded px-2 py-0.5',
                    'border border-[var(--color-border)] outline-none cursor-pointer',
                    activeEnv ? 'text-[var(--color-accent)]' : 'text-[var(--color-muted-foreground)]',
                  )}
                >
                  <option value="">No environment</option>
                  {environments.map(env => (
                    <option key={env.path} value={env.path}>{env.name}</option>
                  ))}
                </select>
              )}

              {workspacePath && (
                <button
                  type="button"
                  onClick={() => { setEditingEnvPath(activeEnv?.path ?? null); setShowEnvEditor(true) }}
                  title={activeEnv ? `Edit environment: ${activeEnv.name}` : 'New environment'}
                  className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
                >
                  <Pencil size={13} />
                </button>
              )}

              {demoRunning && (
                <>
                  <div className="w-px h-4 bg-[var(--color-border)]" />
                  <div className="flex items-center gap-1.5 text-xs">
                    <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse shrink-0" />
                    <span className="text-emerald-400 font-mono">Demo :{demoPort}</span>
                    <button
                      type="button"
                      onClick={handleStopDemo}
                      title="Stop demo server"
                      className="text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors leading-none"
                    >
                      ✕
                    </button>
                  </div>
                </>
              )}
            </div>
          </header>

          {/* Request + Response split */}
          <div className="flex-1 flex overflow-hidden">
            <div className="flex-1 flex flex-col overflow-hidden min-w-0">
              <RequestEditor
                method={activeTab.method}
                url={activeTab.url}
                loading={activeTab.loading}
                queryParams={activeTab.queryParams}
                headers={activeTab.headers}
                body={activeTab.body}
                auth={activeTab.auth}
                onMethodChange={m => updateTab(activeTabId, { method: m })}
                onUrlChange={u => updateTab(activeTabId, { url: u })}
                onUrlPaste={handleUrlPaste}
                onQueryParamsChange={p => updateTab(activeTabId, { queryParams: p })}
                onHeadersChange={h => updateTab(activeTabId, { headers: h })}
                onBodyChange={b => updateTab(activeTabId, { body: b })}
                onAuthChange={a => updateTab(activeTabId, { auth: a })}
                onSend={handleSend}
              />
            </div>

            {/* Response resize handle */}
            <div
              className="w-1 shrink-0 cursor-col-resize hover:bg-[var(--color-accent)] transition-colors"
              onMouseDown={e => {
                e.preventDefault()
                draggingRef.current = 'response'
                document.body.style.cursor = 'col-resize'
                document.body.style.userSelect = 'none'
              }}
            />

            <div
              className={cn('flex flex-col overflow-hidden min-w-0 shrink-0', responseWidth === 0 && 'w-[45%]')}
              style={responseWidth > 0 ? { width: responseWidth } : {}}
            >
              <ResponseViewer response={activeTab.response} loading={activeTab.loading} cachedAt={activeTab.responseCachedAt} />
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}
