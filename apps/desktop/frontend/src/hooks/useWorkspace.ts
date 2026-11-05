import { useState, useCallback } from 'react'
import {
  openWorkspace, newWorkspace, closeWorkspace, getWorkspacePath, listRequests,
  loadRequest, saveRequest, openRequest, deleteRequest,
  createFolder, deleteFolder, renameFolder, renameRequest,
  listEnvironments, selectEnvironment, getActiveEnvironment,
  importOpenAPISpec, importOpenAPISpecFromURL,
  getHistory, clearHistory,
} from '@/lib/bridge'
import type { HistoryEntry, FileNode, EnvInfo, RequestFileIO } from '@/lib/bridge'
import { createBlankTab, computeStateKey } from '@/lib/tabState'
import type { TabsStateTuple } from '@/hooks/useTabsState'

export function useWorkspace(deps: {
  setTabsState: React.Dispatch<React.SetStateAction<TabsStateTuple>>
  tabsStateRef: React.RefObject<TabsStateTuple>
  openFileAsTab: (r: RequestFileIO, path: string) => string
  loadCacheIntoTab: (tabId: string, path: string) => Promise<void>
}) {
  const { setTabsState, tabsStateRef, openFileAsTab, loadCacheIntoTab } = deps

  const [workspacePath, setWorkspacePath] = useState('')
  const [fileTree, setFileTree] = useState<FileNode[]>([])
  const [environments, setEnvironments] = useState<EnvInfo[]>([])
  const [activeEnv, setActiveEnv] = useState<EnvInfo | null>(null)
  const [importing, setImporting] = useState(false)
  const [history, setHistory] = useState<HistoryEntry[]>([])

  // Dialog state
  const [deletingFilePath, setDeletingFilePath] = useState<string | null>(null)
  const [deletingFolderPath, setDeletingFolderPath] = useState<string | null>(null)
  const [newFolderDialog, setNewFolderDialog] = useState<{ parentPath: string; parentLabel: string } | null>(null)
  const [newFolderName, setNewFolderName] = useState('')
  const [newFolderError, setNewFolderError] = useState('')
  const [importFromURLDialog, setImportFromURLDialog] = useState(false)
  const [importFromURLInput, setImportFromURLInput] = useState('')
  const [importFromURLError, setImportFromURLError] = useState('')

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
    } catch { /* best-effort */ }
  }, [])

  const refreshHistory = useCallback(async () => {
    try { setHistory(await getHistory()) } catch { /* best-effort */ }
  }, [])

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
      setTabsState(prev => ({ ...prev, activeTabId: existing.id }))
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
      if (existing) {
        setTabsState(prev => ({ ...prev, activeTabId: existing.id }))
        return
      }
      const tabId = openFileAsTab(r, r.filePath)
      loadCacheIntoTab(tabId, r.filePath)
    } catch (err) {
      console.error('open request:', err)
    }
  }

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

  async function handleClearHistory() {
    try {
      await clearHistory()
      setHistory([])
    } catch (err) {
      console.error('clear history:', err)
    }
  }

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

  async function handleImportFromURL() {
    const url = importFromURLInput.trim()
    if (!url) return
    setImportFromURLDialog(false)
    setImporting(true)
    setImportFromURLError('')
    try {
      const result = await importOpenAPISpecFromURL(url)
      if (result.error) {
        setImportFromURLError(result.error)
        setImportFromURLDialog(true)
      } else {
        setImportFromURLInput('')
        await refreshWorkspace()
      }
    } catch (err) {
      console.error('import from URL failed:', err)
    } finally {
      setImporting(false)
    }
  }

  return {
    workspacePath, fileTree, environments, activeEnv, importing, history,
    refreshWorkspace, refreshHistory,
    handleOpenWorkspace, handleNewWorkspace, handleCloseWorkspace,
    handleSelectFile, handleOpenRequest, handleEnvChange,
    handleClearHistory, handleDuplicateFile,
    deletingFilePath, setDeletingFilePath, confirmDeleteFile, handleDeleteFile,
    handleCreateFolder,
    newFolderDialog, setNewFolderDialog,
    newFolderName, setNewFolderName,
    newFolderError, setNewFolderError,
    confirmCreateFolder, handleDeleteFolder,
    deletingFolderPath, setDeletingFolderPath, confirmDeleteFolder,
    handleRenameFile, handleRenameFolder,
    handleImportOpenAPI,
    importFromURLDialog, setImportFromURLDialog,
    importFromURLInput, setImportFromURLInput,
    importFromURLError, setImportFromURLError,
    handleImportFromURL,
  }
}
