import { useState, useEffect, useRef } from 'react'
import { MenuBar } from '@/components/MenuBar'
import type { MenuDef } from '@/components/MenuBar'
import { TabBar } from '@/components/TabBar'
import { AppToolbar } from '@/components/AppToolbar'
import { SettingsPanel } from '@/components/SettingsPanel'
import { HelpPanel } from '@/components/HelpPanel'
import { EnvironmentEditor } from '@/components/EnvironmentEditor'
import { RequestEditor } from '@/components/RequestEditor'
import { ResponseViewer } from '@/components/ResponseViewer'
import { ScenarioViewer } from '@/components/ScenarioViewer'
import { Sidebar } from '@/components/Sidebar'
import type { SidebarTab } from '@/components/Sidebar'
import { AboutDialog } from '@/components/AboutDialog'
import {
  DeleteRequestDialog,
  DeleteFolderDialog,
  NewFolderDialog,
  CloseDirtyTabDialog,
  DiscardChangesDialog,
  DeleteScenarioDialog,
  ImportFromURLDialog,
} from '@/components/ConfirmDialogs'
import { useTabsState } from '@/hooks/useTabsState'
import { useWorkspace } from '@/hooks/useWorkspace'
import { useSettings } from '@/hooks/useSettings'
import { useDemo } from '@/hooks/useDemo'
import { executeRequest, cancelRequest, saveResponseCache, clearResponseCache, selectEnvironment, createScenario, duplicateScenario, deleteScenario, renameScenario, runScenario, runFolder, quitApp, saveContentToFile } from '@/lib/bridge'
import type { FolderRunState } from '@/components/Sidebar'
import { authStateToPayload } from '@/lib/httpUtils'
import { parseCurl } from '@/lib/curlImport'
import { isTabDirty } from '@/lib/tabState'
import { cn } from '@/lib/utils'
import type { KVPair } from '@/types'

export default function App() {
  // Cross-dependency ref: useTabsState needs refreshWorkspace before useWorkspace is initialized
  const refreshWorkspaceRef = useRef<() => Promise<void>>(async () => {})

  const tabs = useTabsState({ refreshWorkspaceRef })
  const workspace = useWorkspace({
    setTabsState: tabs.setTabsState,
    tabsStateRef: tabs.tabsStateRef,
    openFileAsTab: tabs.openFileAsTab,
    loadCacheIntoTab: tabs.loadCacheIntoTab,
  })
  const settings = useSettings()
  const demo = useDemo({
    refreshWorkspace: workspace.refreshWorkspace,
    setTabsState: tabs.setTabsState,
  })

  // Wire cross-dependency after both hooks are initialized
  // eslint-disable-next-line react/refs
  refreshWorkspaceRef.current = workspace.refreshFileTree

  // Local UI state
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    const v = parseInt(localStorage.getItem('ui.sidebarWidth') ?? '')
    return isNaN(v) ? 224 : Math.max(160, Math.min(480, v))
  })
  const [responseWidth, setResponseWidth] = useState(() => {
    const v = parseInt(localStorage.getItem('ui.responseWidth') ?? '')
    return isNaN(v) ? 0 : Math.max(0, v)
  })
  const draggingRef = useRef<'sidebar' | 'response' | null>(null)
  const sidebarWidthRef = useRef(sidebarWidth)
  const responseWidthRef = useRef(responseWidth)
  useEffect(() => { sidebarWidthRef.current = sidebarWidth }, [sidebarWidth])
  useEffect(() => { responseWidthRef.current = responseWidth }, [responseWidth])
  const [sidebarTab, setSidebarTab] = useState<SidebarTab>('workspace')
  const [showAbout, setShowAbout] = useState(false)
  const [showHelp, setShowHelp] = useState(false)
  const [showEnvEditor, setShowEnvEditor] = useState(false)
  const [editingEnvPath, setEditingEnvPath] = useState<string | null>(null)
  const [deletingScenario, setDeletingScenario] = useState<{ path: string; name: string; isDirty: boolean } | null>(null)
  const [scenarioRenameError, setScenarioRenameError] = useState<string | null>(null)

  // Run All Scenarios state
  const [runAllState, setRunAllState] = useState<import('@/components/Sidebar').RunAllState | null>(null)

  // Run all requests in a folder
  const [folderRunState, setFolderRunState] = useState<FolderRunState | null>(null)

  async function handleRunFolder(folderPath: string) {
    const folderName = folderPath.replace(/\\/g, '/').split('/').filter(Boolean).pop() ?? folderPath
    setFolderRunState({ folderPath, folderName, running: true, results: [] })
    try {
      const results = await runFolder(folderPath)
      setFolderRunState({ folderPath, folderName, running: false, results })
    } catch (e) {
      setFolderRunState({ folderPath, folderName, running: false, results: [{ name: 'Error', path: '', method: '', statusCode: 0, durationMs: 0, error: String(e), passed: false }] })
    }
  }

  async function handleRunAll() {
    const list = workspace.scenarioList
    if (!list.length) return
    const envPath = workspace.activeEnv?.path ?? ''
    setRunAllState({ running: true, done: 0, total: list.length, passed: 0, results: [] })
    let passedCount = 0
    for (const s of list) {
      try {
        const r = await runScenario(s.path, envPath)
        if (r.passed) passedCount++
        setRunAllState(prev => prev ? {
          ...prev,
          done: prev.done + 1,
          passed: passedCount,
          results: [...prev.results, { path: s.path, name: s.title, passed: r.passed, durationMs: r.durationMs }],
        } : prev)
      } catch (e) {
        setRunAllState(prev => prev ? {
          ...prev,
          done: prev.done + 1,
          results: [...prev.results, { path: s.path, name: s.title, passed: false, durationMs: 0, error: String(e) }],
        } : prev)
      }
    }
    setRunAllState(prev => prev ? { ...prev, running: false } : prev)
  }

  // Init
  useEffect(() => {
    workspace.refreshWorkspace()
    workspace.refreshHistory()
    settings.loadSettings()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Resize
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
      if (draggingRef.current === 'sidebar') {
        localStorage.setItem('ui.sidebarWidth', String(sidebarWidthRef.current))
      } else if (draggingRef.current === 'response') {
        localStorage.setItem('ui.responseWidth', String(responseWidthRef.current))
      }
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

  // Request execution
  const handleSendRef = useRef<(() => void) | null>(null)

  async function handleClearResponse() {
    const { activeTab, activeTabId, updateTab } = tabs
    updateTab(activeTabId, { response: null, responseCachedAt: null })
    if (activeTab.filePath) {
      clearResponseCache(activeTab.filePath).catch(() => {})
    }
  }

  async function handleSend() {
    const { activeTab, activeTabId, updateTab, tabsStateRef } = tabs
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
        timeoutMs: activeTab.timeoutMs,
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
      workspace.refreshHistory()
    }
  }

  // eslint-disable-next-line react/refs
  handleSendRef.current = handleSend

  function handleCancel() {
    cancelRequest().catch(() => {})
  }

  function handleUrlPaste(raw: string) {
    const { activeTabId, activeTab, updateTab } = tabs
    try {
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
      updateTab(activeTabId, { url: raw, saveError: 'Could not extract params from pasted URL' })
      setTimeout(() => updateTab(activeTabId, { saveError: null }), 3000)
    }
  }

  // Keyboard shortcuts — registered once, reads from refs
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key === 'S' && e.shiftKey) {
        e.preventDefault()
        tabs.handleSaveAll()
      } else if ((e.ctrlKey || e.metaKey) && e.key === 's' && !e.shiftKey) {
        e.preventDefault()
        tabs.saveTabByIdRef.current(tabs.tabsStateRef.current.activeTabId)
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
        e.preventDefault()
        tabs.handleNewTab()
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'w') {
        e.preventDefault()
        tabs.handleCloseTab(tabs.tabsStateRef.current.activeTabId)
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        const el = document.activeElement
        if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) return
        e.preventDefault()
        handleSendRef.current?.()
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'o') {
        e.preventDefault()
        workspace.handleOpenRequest()
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'Tab') {
        e.preventDefault()
        const { tabs: tList, activeTabId } = tabs.tabsStateRef.current
        if (tList.length < 2) return
        const idx = tList.findIndex(t => t.id === activeTabId)
        const next = e.shiftKey
          ? (idx - 1 + tList.length) % tList.length
          : (idx + 1) % tList.length
        tabs.switchTo(tList[next].id)
      }
      const digit = parseInt(e.key)
      if ((e.ctrlKey || e.metaKey) && digit >= 1 && digit <= 9) {
        const { tabs: tList } = tabs.tabsStateRef.current
        const target = tList[digit - 1]
        if (target) { e.preventDefault(); tabs.switchTo(target.id) }
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Menus
  const menus: MenuDef[] = [
    {
      label: 'File',
      items: [
        { kind: 'action', label: 'Open Workspace…',  onClick: workspace.handleOpenWorkspace },
        { kind: 'action', label: 'New Workspace…',   onClick: workspace.handleNewWorkspace },
        { kind: 'action', label: 'Close Workspace',  onClick: workspace.handleCloseWorkspace, disabled: !workspace.workspacePath },
        {
          kind: 'submenu',
          label: 'Recent Workspaces',
          disabled: workspace.recentWorkspaces.length === 0,
          items: workspace.recentWorkspaces.map(p => ({
            kind: 'action' as const,
            label: p.split(/[\\/]/).pop() ?? p,
            title: p,
            onClick: () => workspace.handleOpenWorkspaceByPath(p),
          })),
        },
        { kind: 'separator' },
        { kind: 'action', label: 'New Tab',    shortcut: 'Ctrl+N',       onClick: tabs.handleNewTab },
        { kind: 'action', label: 'Open Request…', shortcut: 'Ctrl+O',    onClick: workspace.handleOpenRequest },
        { kind: 'action', label: 'Save Request',  shortcut: 'Ctrl+S',    onClick: tabs.handleSaveRequest },
        { kind: 'action', label: 'Save All',      shortcut: 'Ctrl+Shift+S', onClick: tabs.handleSaveAll, disabled: tabs.dirtyFilePaths.size === 0 },
        { kind: 'action', label: 'Save As…',                              onClick: tabs.handleSaveAsRequest },
        { kind: 'separator' },
        {
          kind: 'submenu',
          label: 'Import',
          disabled: !workspace.workspacePath,
          items: [
            { kind: 'action', label: 'OpenAPI Spec…',      onClick: workspace.handleImportOpenAPI },
            { kind: 'action', label: 'OpenAPI from URL…',  onClick: () => { workspace.setImportFromURLInput(''); workspace.setImportFromURLError(''); workspace.setImportFromURLDialog(true) } },
            { kind: 'separator' },
            { kind: 'action', label: 'Postman Collection…', onClick: workspace.handleImportPostman },
            { kind: 'action', label: 'Insomnia Export…',    onClick: workspace.handleImportInsomnia },
          ],
        },
        {
          kind: 'submenu',
          label: 'Export',
          disabled: !tabs.activeTab.response,
          items: [
            {
              kind: 'action',
              label: 'Save Response Body…',
              onClick: () => {
                const body = tabs.activeTab.response?.body ?? ''
                const ext = body.trimStart().startsWith('{') || body.trimStart().startsWith('[') ? '.json' : '.txt'
                saveContentToFile(body, `response${ext}`)
              },
            },
          ],
        },
        { kind: 'separator' },
        { kind: 'action', label: 'Exit', onClick: quitApp },
      ],
    },
    {
      label: 'View',
      items: [
        { kind: 'action', label: settings.settings.theme === 'dark' ? 'Switch to Light Theme' : 'Switch to Dark Theme', onClick: settings.handleToggleTheme },
        { kind: 'separator' },
        { kind: 'action', label: 'Settings…', onClick: () => settings.setShowSettings(true) },
      ],
    },
    {
      label: 'Help',
      items: [
        {
          kind: 'action',
          label: demo.demoStarting ? 'Starting Demo…' : demo.demoRunning ? 'Stop Demo Server' : 'Start Demo Server',
          onClick: demo.demoRunning ? demo.handleStopDemo : demo.handleStartDemo,
          disabled: demo.demoStarting,
        },
        { kind: 'separator' },
        { kind: 'action', label: 'User Guide', onClick: () => setShowHelp(true) },
        { kind: 'separator' },
        { kind: 'action', label: 'About Clear Signal', onClick: () => setShowAbout(true) },
      ],
    },
  ]

  const { tabs: tabList, activeTabId, activeTab, updateTab } = tabs
  const closingTab = tabs.closingDirtyTabId ? tabList.find(t => t.id === tabs.closingDirtyTabId) : null
  const tabInfos = tabList.map(t => ({
    id: t.id,
    name: t.name,
    method: t.method,
    isDirty: isTabDirty(t),
    kind: t.kind ?? 'request' as const,
  }))

  return (
    <div className="flex flex-col h-screen w-screen overflow-hidden">
      <MenuBar menus={menus} />

      {workspace.deletingFilePath && (
        <DeleteRequestDialog
          filePath={workspace.deletingFilePath}
          onCancel={() => workspace.setDeletingFilePath(null)}
          onConfirm={workspace.confirmDeleteFile}
        />
      )}

      {showAbout && <AboutDialog onClose={() => setShowAbout(false)} />}
      {showHelp && <HelpPanel onClose={() => setShowHelp(false)} />}

      {workspace.importFromURLDialog && (
        <ImportFromURLDialog
          value={workspace.importFromURLInput}
          onChange={v => { workspace.setImportFromURLInput(v); workspace.setImportFromURLError('') }}
          onCancel={() => { workspace.setImportFromURLDialog(false); workspace.setImportFromURLError('') }}
          onConfirm={workspace.handleImportFromURL}
          error={workspace.importFromURLError}
        />
      )}

      {workspace.newFolderDialog && (
        <NewFolderDialog
          parentLabel={workspace.newFolderDialog.parentLabel}
          value={workspace.newFolderName}
          onChange={v => { workspace.setNewFolderName(v); workspace.setNewFolderError('') }}
          onCancel={() => { workspace.setNewFolderDialog(null); workspace.setNewFolderError('') }}
          onConfirm={workspace.confirmCreateFolder}
          error={workspace.newFolderError}
        />
      )}

      {workspace.deletingFolderPath && (
        <DeleteFolderDialog
          folderPath={workspace.deletingFolderPath}
          onCancel={() => workspace.setDeletingFolderPath(null)}
          onConfirm={workspace.confirmDeleteFolder}
        />
      )}

      {closingTab && (
        <CloseDirtyTabDialog
          tabName={closingTab.name}
          onCancel={() => tabs.setClosingDirtyTabId(null)}
          onDiscard={() => { tabs.setClosingDirtyTabId(null); tabs.removeTab(closingTab.id) }}
          onSave={async () => {
            const ok = await tabs.saveTabById(closingTab.id)
            if (ok) { tabs.setClosingDirtyTabId(null); tabs.removeTab(closingTab.id) }
          }}
        />
      )}

      {tabs.closingDirtyScenarioTabId && (() => {
        const t = tabList.find(x => x.id === tabs.closingDirtyScenarioTabId)
        return t ? (
          <DiscardChangesDialog
            title="Close scenario"
            message={`"${t.name ?? 'This scenario'}" has unsaved changes. Close and discard them?`}
            onCancel={() => tabs.setClosingDirtyScenarioTabId(null)}
            onDiscard={() => { tabs.removeTab(t.id); tabs.setClosingDirtyScenarioTabId(null) }}
          />
        ) : null
      })()}

      {tabs.pendingBulkClose && (() => {
        const dirtyCount = tabs.pendingBulkClose.tabIds.filter(id => {
          const t = tabList.find(x => x.id === id)
          return t && (isTabDirty(t) || (t.kind === 'scenario' && t.scenarioDirty))
        }).length
        const noun = dirtyCount === 1 ? 'tab has' : 'tabs have'
        return (
          <DiscardChangesDialog
            title="Close tabs"
            message={`${dirtyCount} ${noun} unsaved changes. Close and discard them?`}
            onCancel={() => tabs.setPendingBulkClose(null)}
            onDiscard={tabs.confirmBulkClose}
          />
        )
      })()}

      {deletingScenario && (
        <DeleteScenarioDialog
          name={deletingScenario.name}
          isDirty={deletingScenario.isDirty}
          onCancel={() => setDeletingScenario(null)}
          onConfirm={async () => {
            const { path } = deletingScenario
            setDeletingScenario(null)
            const matchTab = tabList.find(t => t.kind === 'scenario' && t.scenarioPath === path)
            const err = await deleteScenario(path)
            if (!err) {
              if (matchTab) tabs.removeTab(matchTab.id)
              await workspace.refreshWorkspace()
            }
          }}
        />
      )}

      <div className="flex flex-1 overflow-hidden">
        {settings.showSettings && (
          <SettingsPanel
            settings={settings.settings}
            onSave={settings.handleSaveSettings}
            onReset={settings.handleResetSettings}
            onPreview={settings.applySettings}
            onClose={() => { settings.applySettings(settings.settings); settings.setShowSettings(false) }}
          />
        )}

        {showEnvEditor && (
          <EnvironmentEditor
            envPath={editingEnvPath}
            workspacePath={workspace.workspacePath}
            onClose={() => setShowEnvEditor(false)}
            onSaved={async saved => {
              setShowEnvEditor(false)
              await workspace.refreshWorkspace()
              try { await selectEnvironment(saved.path) } catch { /* best-effort */ }
              await workspace.refreshWorkspace()
            }}
            onDeleted={async () => {
              setShowEnvEditor(false)
              await workspace.refreshWorkspace()
            }}
          />
        )}

        <Sidebar
          width={sidebarWidth}
          tab={sidebarTab}
          onTabChange={setSidebarTab}
          workspacePath={workspace.workspacePath}
          fileTree={workspace.fileTree}
          history={workspace.history}
          onOpenWorkspace={workspace.handleOpenWorkspace}
          onNewWorkspace={workspace.handleNewWorkspace}
          onSelectFile={workspace.handleSelectFile}
          onDeleteFile={workspace.handleDeleteFile}
          onDuplicateFile={workspace.handleDuplicateFile}
          onDeleteFolder={workspace.handleDeleteFolder}
          onNewRequestInFolder={tabs.handleNewRequestInFolder}
          onCreateFolder={workspace.handleCreateFolder}
          onRenameFile={workspace.handleRenameFile}
          onRenameFolder={workspace.handleRenameFolder}
          onMoveFile={workspace.handleMoveFile}
          onSelectHistory={tabs.handleSelectHistory}
          onClearHistory={workspace.handleClearHistory}
          onImportOpenAPI={workspace.handleImportOpenAPI}
          onImportPostman={workspace.handleImportPostman}
          onImportInsomnia={workspace.handleImportInsomnia}
          onTryDemo={demo.handleStartDemo}
          demoStarting={demo.demoStarting}
          onRefresh={workspace.refreshWorkspace}
          importing={workspace.importing}
          activeFilePath={activeTab.filePath ?? undefined}
          dirtyFilePaths={tabs.dirtyFilePaths}
          scenarios={workspace.scenarioList}
          onSelectScenario={tabs.openScenarioAsTab}
          scenarioError={scenarioRenameError}
          runAllState={runAllState ?? undefined}
          onRunAllScenarios={handleRunAll}
          onDismissRunAll={() => setRunAllState(null)}
          onRunFolder={handleRunFolder}
          folderRunState={folderRunState}
          onDismissFolderRun={() => setFolderRunState(null)}
          onNewScenario={async (name) => {
            const result = await createScenario(name)
            if (!result.error && result.path) {
              await workspace.refreshWorkspace()
              tabs.openScenarioAsTab(result.path, name)
            }
          }}
          onDuplicateScenario={async (path) => {
            const result = await duplicateScenario(path)
            if (!result.error && result.path) {
              await workspace.refreshWorkspace()
              const name = result.path.split(/[\\/]/).pop()?.replace(/\.(yaml|yml|json)$/i, '') ?? 'copy'
              tabs.openScenarioAsTab(result.path, name)
            }
          }}
          onDeleteScenario={(path) => {
            const matchTab = tabs.tabs.find(t => t.kind === 'scenario' && t.scenarioPath === path)
            const name = workspace.scenarioList.find(s => s.path === path)?.title ?? path.split(/[\\/]/).pop()?.replace(/\.(yaml|yml|json)$/i, '') ?? 'scenario'
            setDeletingScenario({ path, name, isDirty: matchTab?.scenarioDirty ?? false })
          }}
          onRenameScenario={async (path, newName) => {
            const result = await renameScenario(path, newName)
            if (!result.startsWith('error:')) {
              const newPath = result
              tabs.setTabsState(prev => ({
                ...prev,
                tabs: prev.tabs.map(t =>
                  t.kind === 'scenario' && t.scenarioPath === path
                    ? { ...t, scenarioPath: newPath, scenarioName: newName, name: newName }
                    : t
                ),
              }))
              await workspace.refreshWorkspace()
            } else {
              setScenarioRenameError(result.replace(/^error:\s*/i, ''))
              setTimeout(() => setScenarioRenameError(null), 4000)
            }
          }}
        />

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
            onSelect={tabs.handleSelectTab}
            onClose={tabs.handleCloseTab}
            onNew={tabs.handleNewTab}
            onSave={tabs.saveTabById}
            onCloseOthers={tabs.handleCloseOthers}
            onCloseToRight={tabs.handleCloseToRight}
          />

          <AppToolbar
            activeTab={activeTab}
            activeTabId={activeTabId}
            updateTab={updateTab}
            environments={workspace.environments}
            activeEnv={workspace.activeEnv}
            workspacePath={workspace.workspacePath}
            demoRunning={demo.demoRunning}
            demoPort={demo.demoPort}
            onOpenRequest={workspace.handleOpenRequest}
            onSaveRequest={tabs.handleSaveRequest}
            onNewRequest={tabs.handleNewTab}
            onDuplicateTab={tabs.handleDuplicateTab}
            onShowSettings={() => settings.setShowSettings(true)}
            onShowHelp={() => setShowHelp(true)}
            onEnvChange={workspace.handleEnvChange}
            onEditEnv={() => { setEditingEnvPath(workspace.activeEnv?.path ?? null); setShowEnvEditor(true) }}
            onStopDemo={demo.handleStopDemo}
          />

          <div className="flex-1 flex overflow-hidden relative">
            {/* Scenario viewers: keep mounted once opened so edit state survives tab switches */}
            {tabList.filter(t => t.kind === 'scenario').map(t => (
              <div
                key={t.id}
                className="absolute inset-0 overflow-hidden"
                style={{ display: t.id === activeTabId ? 'flex' : 'none' }}
              >
                <ScenarioViewer
                  scenarioPath={t.scenarioPath ?? ''}
                  scenarioName={t.scenarioName ?? ''}
                  result={t.scenarioResult ?? null}
                  running={t.scenarioRunning ?? false}
                  theme={settings.settings.theme}
                  environments={workspace.environments}
                  activeEnvPath={workspace.activeEnv?.path ?? ''}
                  onResultChange={r => updateTab(t.id, { scenarioResult: r })}
                  onRunningChange={running => updateTab(t.id, { scenarioRunning: running })}
                  onDirtyChange={dirty => updateTab(t.id, { scenarioDirty: dirty })}
                />
              </div>
            ))}

            {/* Request panel: only mount when active tab is a request */}
            {activeTab.kind !== 'scenario' && (
              <>
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
                    timeoutMs={activeTab.timeoutMs}
                    onTimeoutChange={ms => updateTab(activeTabId, { timeoutMs: ms })}
                    envVarMap={workspace.envVarMap}
                    onSend={handleSend}
                    onCancel={handleCancel}
                  />
                </div>

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
                  <ResponseViewer response={activeTab.response} loading={activeTab.loading} cachedAt={activeTab.responseCachedAt} onClear={handleClearResponse} />
                </div>
              </>
            )}
          </div>
        </main>
      </div>
    </div>
  )
}
