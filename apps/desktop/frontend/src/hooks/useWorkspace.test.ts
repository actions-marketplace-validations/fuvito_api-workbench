import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import type { FileNode, RequestFileIO } from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({
  openWorkspace: vi.fn(), newWorkspace: vi.fn(), closeWorkspace: vi.fn(), getWorkspacePath: vi.fn(), listRequests: vi.fn(),
  loadRequest: vi.fn(), saveRequest: vi.fn(), openRequest: vi.fn(), deleteRequest: vi.fn(),
  createFolder: vi.fn(), deleteFolder: vi.fn(), renameFolder: vi.fn(), renameRequest: vi.fn(), moveRequest: vi.fn(),
  listEnvironments: vi.fn(), selectEnvironment: vi.fn(), getActiveEnvironment: vi.fn(), getEnvironment: vi.fn(),
  importOpenAPISpec: vi.fn(), importOpenAPISpecFromURL: vi.fn(), importPostmanCollection: vi.fn(), importInsomniaExport: vi.fn(),
  getHistory: vi.fn(), clearHistory: vi.fn(), listScenarios: vi.fn(),
  getRecentWorkspaces: vi.fn(), openWorkspaceByPath: vi.fn(),
  loadResponseCache: vi.fn(), setHasDirtyTabs: vi.fn(),
}))

import * as bridge from '@/lib/bridge'
import { useTabsState } from './useTabsState'
import { useWorkspace } from './useWorkspace'

const b = vi.mocked(bridge, true)

const req = (over: Partial<RequestFileIO> = {}): RequestFileIO => ({
  schemaVersion: '1', title: 'r', method: 'GET', url: 'https://x.test', ...over,
})
const tree: FileNode[] = [
  { name: 'sub', path: '/ws/requests/sub', isDir: true, children: [{ name: 'a.yaml', path: '/ws/requests/sub/a.yaml', isDir: false }] },
  { name: 'b.yaml', path: '/ws/requests/b.yaml', isDir: false },
]

function setup(onWorkspaceOpened?: () => void) {
  return renderHook(() => {
    const tabs = useTabsState({ refreshWorkspaceRef: { current: async () => {} } })
    const ws = useWorkspace({
      setTabsState: tabs.setTabsState, tabsStateRef: tabs.tabsStateRef,
      openFileAsTab: tabs.openFileAsTab, loadCacheIntoTab: tabs.loadCacheIntoTab, onWorkspaceOpened,
    })
    return { tabs, ws }
  })
}

function primeRefresh(over: { path?: string; tree?: FileNode[] } = {}) {
  b.getWorkspacePath.mockResolvedValue((over.path ?? '/ws') as never)
  b.listRequests.mockResolvedValue(over.tree ?? tree)
  b.listEnvironments.mockResolvedValue([{ name: 'local', path: '/ws/environments/local.yaml' }])
  b.getActiveEnvironment.mockResolvedValue({ name: 'local', path: '/ws/environments/local.yaml' })
  b.listScenarios.mockResolvedValue([])
  b.getRecentWorkspaces.mockResolvedValue(['/ws'])
  b.getEnvironment.mockResolvedValue({ path: '', name: 'local', variables: [{ name: 'host', value: 'h', secret: false }] } as never)
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.spyOn(console, 'error').mockImplementation(() => {})
  b.loadResponseCache.mockResolvedValue(null as never)
  b.setHasDirtyTabs.mockResolvedValue(undefined)
  b.getHistory.mockResolvedValue([])
})
afterEach(() => vi.restoreAllMocks())

describe('refreshWorkspace', () => {
  it('loads everything and exposes environment variables', async () => {
    primeRefresh()
    const { result } = setup()
    let returned: FileNode[] = []
    await act(async () => { returned = await result.current.ws.refreshWorkspace() })

    expect(returned).toEqual(tree)
    const ws = result.current.ws
    expect(ws.workspacePath).toBe('/ws')
    expect(ws.fileTree).toEqual(tree)
    expect(ws.environments).toHaveLength(1)
    expect(ws.activeEnv?.name).toBe('local')
    expect(ws.envVarMap).toEqual({ host: 'h' })
    expect(ws.recentWorkspaces).toEqual(['/ws'])
  })

  it('tolerates null lists from the backend', async () => {
    primeRefresh()
    b.listRequests.mockResolvedValue(null as never)
    b.listEnvironments.mockResolvedValue(null as never)
    b.listScenarios.mockResolvedValue(null as never)
    b.getRecentWorkspaces.mockResolvedValue(null as never)
    b.getActiveEnvironment.mockResolvedValue(null as never)
    const { result } = setup()
    await act(async () => { await result.current.ws.refreshWorkspace() })
    expect(result.current.ws.fileTree).toEqual([])
    expect(result.current.ws.envVarMap).toEqual({})
  })

  it('returns an empty tree when the backend fails', async () => {
    b.getWorkspacePath.mockRejectedValue(new Error('x'))
    b.listRequests.mockResolvedValue([])
    b.listEnvironments.mockResolvedValue([])
    b.getActiveEnvironment.mockResolvedValue(null as never)
    b.listScenarios.mockResolvedValue([])
    b.getRecentWorkspaces.mockResolvedValue([])
    const { result } = setup()
    let returned: FileNode[] = [tree[0]]
    await act(async () => { returned = await result.current.ws.refreshWorkspace() })
    expect(returned).toEqual([])
  })

  it('clears the variable map if loading the environment fails', async () => {
    primeRefresh()
    b.getEnvironment.mockRejectedValue(new Error('bad env'))
    const { result } = setup()
    await act(async () => { await result.current.ws.refreshWorkspace() })
    expect(result.current.ws.envVarMap).toEqual({})
  })

  it('refreshFileTree and refreshHistory are best-effort', async () => {
    const { result } = setup()
    b.listRequests.mockRejectedValueOnce(new Error('x'))
    await act(async () => { await result.current.ws.refreshFileTree() })
    b.getHistory.mockRejectedValueOnce(new Error('x'))
    await act(async () => { await result.current.ws.refreshHistory() })
    b.listRequests.mockResolvedValueOnce(null as never)
    await act(async () => { await result.current.ws.refreshFileTree() })
    expect(result.current.ws.fileTree).toEqual([])
  })
})

describe('opening workspaces', () => {
  it.each([
    ['handleOpenWorkspace', () => b.openWorkspace, undefined],
    ['handleNewWorkspace', () => b.newWorkspace, undefined],
    ['handleOpenWorkspaceByPath', () => b.openWorkspaceByPath, '/ws'],
  ] as const)('%s opens the first request and notifies the app', async (name, mock, arg) => {
    primeRefresh()
    mock().mockResolvedValue('/ws')
    b.loadRequest.mockResolvedValue(req({ title: 'first' }))
    const onOpened = vi.fn()
    const { result } = setup(onOpened)
    await act(async () => { await (result.current.ws[name] as (a?: string) => Promise<void>)(arg) })

    expect(b.loadRequest).toHaveBeenCalledWith('/ws/requests/sub/a.yaml')
    expect(result.current.tabs.activeTab.name).toBe('first')
    expect(result.current.ws.workspacePath).toBe('/ws')
    expect(onOpened).toHaveBeenCalledTimes(1)
    expect(b.getHistory).toHaveBeenCalled()
  })

  it('does nothing when the dialog is cancelled', async () => {
    b.openWorkspace.mockResolvedValue('' as never)
    const onOpened = vi.fn()
    const { result } = setup(onOpened)
    await act(async () => { await result.current.ws.handleOpenWorkspace() })
    expect(onOpened).not.toHaveBeenCalled()
    expect(b.listRequests).not.toHaveBeenCalled()
  })

  it('still opens the first request when a page tab such as Workspace is already open', async () => {
    primeRefresh()
    b.openWorkspace.mockResolvedValue('/ws')
    b.loadRequest.mockResolvedValue(req({ title: 'first' }))
    const { result } = setup()
    act(() => { result.current.tabs.openWorkspaceTab() })
    await act(async () => { await result.current.ws.handleOpenWorkspace() })
    expect(b.loadRequest).toHaveBeenCalledWith('/ws/requests/sub/a.yaml')
    expect(result.current.tabs.tabs.map(t => t.kind ?? 'request')).toEqual(['workspace', 'request', 'request'])
  })

  it('does not replace tabs the user already has open', async () => {
    primeRefresh()
    b.openWorkspace.mockResolvedValue('/ws')
    const { result } = setup()
    act(() => result.current.tabs.updateTab(result.current.tabs.activeTabId, { url: 'https://in-progress.test' }))
    await act(async () => { await result.current.ws.handleOpenWorkspace() })
    expect(b.loadRequest).not.toHaveBeenCalled()
    expect(result.current.tabs.tabs).toHaveLength(1)
  })

  it('skips auto-open for an empty workspace and survives load failures', async () => {
    primeRefresh({ tree: [{ name: 'sub', path: '/ws/requests/sub', isDir: true, children: [] }] })
    b.openWorkspace.mockResolvedValue('/ws')
    const { result } = setup()
    await act(async () => { await result.current.ws.handleOpenWorkspace() })
    expect(b.loadRequest).not.toHaveBeenCalled()

    primeRefresh()
    b.loadRequest.mockRejectedValue(new Error('unreadable'))
    await act(async () => { await result.current.ws.handleOpenWorkspace() })
    expect(result.current.tabs.tabs).toHaveLength(1)
  })

  it('logs rather than throws when a dialog fails', async () => {
    b.openWorkspace.mockRejectedValue(new Error('x'))
    b.newWorkspace.mockRejectedValue(new Error('x'))
    b.openWorkspaceByPath.mockRejectedValue(new Error('x'))
    const { result } = setup()
    await act(async () => {
      await result.current.ws.handleOpenWorkspace()
      await result.current.ws.handleNewWorkspace()
      await result.current.ws.handleOpenWorkspaceByPath('/x')
    })
    expect(console.error).toHaveBeenCalledTimes(3)
  })
})

describe('closing a workspace', () => {
  it('clears state and closes only tabs that belong to it', async () => {
    primeRefresh()
    const { result } = setup()
    await act(async () => { await result.current.ws.refreshWorkspace() })
    act(() => { result.current.tabs.openFileAsTab(req({ title: 'inside' }), '/ws/requests/a.yaml') })
    act(() => { result.current.tabs.openFileAsTab(req({ title: 'outside' }), '/elsewhere/b.yaml') })
    act(() => { result.current.tabs.openScenarioAsTab('/ws/scenarios/s.yaml', 'S') })
    act(() => { result.current.tabs.openFileAsTab(req({ title: 'sibling' }), '/ws-other/c.yaml') })

    await act(async () => { await result.current.ws.handleCloseWorkspace() })

    expect(b.closeWorkspace).toHaveBeenCalled()
    expect(result.current.ws).toMatchObject({ workspacePath: '', fileTree: [], environments: [], activeEnv: null, envVarMap: {} })
    const names = result.current.tabs.tabs.map(t => t.name)
    expect(names).toEqual(['outside', 'sibling'])
    expect(result.current.tabs.tabs.some(t => t.id === result.current.tabs.activeTabId)).toBe(true)
  })

  it('handles Windows-style separators', async () => {
    primeRefresh({ path: 'C:\\ws' })
    const { result } = setup()
    await act(async () => { await result.current.ws.refreshWorkspace() })
    act(() => { result.current.tabs.openFileAsTab(req(), 'C:\\ws\\requests\\a.yaml') })
    await act(async () => { await result.current.ws.handleCloseWorkspace() })
    expect(result.current.tabs.tabs.every(t => t.filePath === null)).toBe(true)
  })

  it('leaves a blank tab when every tab belonged to the workspace', async () => {
    primeRefresh()
    const { result } = setup()
    await act(async () => { await result.current.ws.refreshWorkspace() })
    act(() => { result.current.tabs.openFileAsTab(req(), '/ws/requests/a.yaml') })
    await act(async () => { await result.current.ws.handleCloseWorkspace() })
    expect(result.current.tabs.tabs).toHaveLength(1)
    expect(result.current.tabs.activeTab.filePath).toBeNull()
  })

  it('logs when closing fails and keeps the workspace', async () => {
    primeRefresh()
    b.closeWorkspace.mockRejectedValue(new Error('x'))
    const { result } = setup()
    await act(async () => { await result.current.ws.refreshWorkspace() })
    await act(async () => { await result.current.ws.handleCloseWorkspace() })
    expect(result.current.ws.workspacePath).toBe('/ws')
  })
})

describe('selecting files', () => {
  it('opens a file once and switches to it afterwards', async () => {
    b.loadRequest.mockResolvedValue(req({ title: 'one' }))
    const { result } = setup()
    await act(async () => { await result.current.ws.handleSelectFile('/ws/a.yaml') })
    const id = result.current.tabs.activeTabId
    act(() => result.current.tabs.handleNewTab())
    await act(async () => { await result.current.ws.handleSelectFile('/ws/a.yaml') })
    expect(b.loadRequest).toHaveBeenCalledTimes(1)
    expect(result.current.tabs.activeTabId).toBe(id)
    expect(b.loadResponseCache).toHaveBeenCalledWith('/ws/a.yaml')
  })

  it('ignores empty results and logs errors', async () => {
    const { result } = setup()
    b.loadRequest.mockResolvedValueOnce(null as never)
    await act(async () => { await result.current.ws.handleSelectFile('/ws/a.yaml') })
    b.loadRequest.mockRejectedValueOnce(new Error('x'))
    await act(async () => { await result.current.ws.handleSelectFile('/ws/b.yaml') })
    expect(result.current.tabs.tabs).toHaveLength(1)
    expect(console.error).toHaveBeenCalled()
  })

  it('open-request dialog loads a file, reuses open tabs, and ignores cancellation', async () => {
    const { result } = setup()
    b.openRequest.mockResolvedValueOnce(null as never)
    await act(async () => { await result.current.ws.handleOpenRequest() })
    b.openRequest.mockResolvedValueOnce(req({ filePath: '' }))
    await act(async () => { await result.current.ws.handleOpenRequest() })
    expect(result.current.tabs.tabs[0].filePath).toBeNull()

    b.openRequest.mockResolvedValue(req({ title: 'picked', filePath: '/p/a.yaml' }))
    await act(async () => { await result.current.ws.handleOpenRequest() })
    const id = result.current.tabs.activeTabId
    act(() => result.current.tabs.handleNewTab())
    await act(async () => { await result.current.ws.handleOpenRequest() })
    expect(result.current.tabs.activeTabId).toBe(id)

    b.openRequest.mockRejectedValue(new Error('x'))
    await act(async () => { await result.current.ws.handleOpenRequest() })
    expect(console.error).toHaveBeenCalled()
  })
})

describe('environments', () => {
  it('switches environments and reloads variables', async () => {
    b.listEnvironments.mockResolvedValue([{ name: 'a', path: '/a' }, { name: 'b', path: '/b' }])
    b.getEnvironment.mockResolvedValue({ path: '/b', name: 'b', variables: [{ name: 'k', value: 'v', secret: false }] } as never)
    const { result } = setup()
    await act(async () => { await result.current.ws.handleEnvChange('/b') })
    expect(b.selectEnvironment).toHaveBeenCalledWith('/b')
    expect(result.current.ws.activeEnv?.name).toBe('b')
    expect(result.current.ws.envVarMap).toEqual({ k: 'v' })
  })

  it('selecting "no environment" clears the selection', async () => {
    b.listEnvironments.mockResolvedValue([{ name: 'a', path: '/a' }])
    const { result } = setup()
    await act(async () => { await result.current.ws.handleEnvChange('') })
    expect(result.current.ws.activeEnv).toBeNull()
    expect(result.current.ws.envVarMap).toEqual({})
  })

  it('logs when the backend rejects the selection', async () => {
    b.selectEnvironment.mockRejectedValue(new Error('x'))
    const { result } = setup()
    await act(async () => { await result.current.ws.handleEnvChange('/z') })
    expect(console.error).toHaveBeenCalled()
  })
})

describe('history', () => {
  it('clears history and tolerates failure', async () => {
    b.getHistory.mockResolvedValue([{ id: 1 } as never])
    const { result } = setup()
    await act(async () => { await result.current.ws.refreshHistory() })
    expect(result.current.ws.history).toHaveLength(1)
    await act(async () => { await result.current.ws.handleClearHistory() })
    expect(result.current.ws.history).toEqual([])

    b.clearHistory.mockRejectedValueOnce(new Error('x'))
    b.getHistory.mockResolvedValue([{ id: 2 } as never])
    await act(async () => { await result.current.ws.refreshHistory() })
    await act(async () => { await result.current.ws.handleClearHistory() })
    expect(result.current.ws.history).toHaveLength(1)
  })
})

describe('file operations', () => {
  it('duplicate saves a copy next to the original with a derived title', async () => {
    b.loadRequest.mockResolvedValue(req({ title: 'users' }))
    primeRefresh()
    const { result } = setup()
    await act(async () => { await result.current.ws.handleDuplicateFile('/ws/requests/users.yaml') })
    expect(b.saveRequest).toHaveBeenCalledWith(expect.objectContaining({ title: 'users Copy', filePath: '', targetDir: '/ws/requests' }))

    b.loadRequest.mockResolvedValue(req({ title: '' }))
    await act(async () => { await result.current.ws.handleDuplicateFile('C:\\ws\\requests\\x.yaml') })
    expect(b.saveRequest).toHaveBeenLastCalledWith(expect.objectContaining({ title: 'request Copy', targetDir: 'C:\\ws\\requests' }))

    b.loadRequest.mockRejectedValue(new Error('x'))
    await act(async () => { await result.current.ws.handleDuplicateFile('/x') })
    expect(console.error).toHaveBeenCalled()
  })

  it('delete asks for confirmation, then removes the file and its tab', async () => {
    primeRefresh()
    const { result } = setup()
    act(() => { result.current.tabs.openFileAsTab(req({ title: 'a' }), '/ws/a.yaml') })
    act(() => result.current.ws.handleDeleteFile('/ws/a.yaml'))
    expect(result.current.ws.deletingFilePath).toBe('/ws/a.yaml')
    expect(b.deleteRequest).not.toHaveBeenCalled()

    await act(async () => { await result.current.ws.confirmDeleteFile() })
    expect(b.deleteRequest).toHaveBeenCalledWith('/ws/a.yaml')
    expect(result.current.ws.deletingFilePath).toBeNull()
    expect(result.current.tabs.tabs).toHaveLength(1)
    expect(result.current.tabs.activeTab.filePath).toBeNull()
  })

  it('deleting a file keeps other tabs and an unrelated active tab', async () => {
    primeRefresh()
    const { result } = setup()
    act(() => { result.current.tabs.openFileAsTab(req({ title: 'a' }), '/ws/a.yaml') })
    act(() => { result.current.tabs.openFileAsTab(req({ title: 'b' }), '/ws/b.yaml') })
    const keep = result.current.tabs.activeTabId
    act(() => result.current.ws.handleDeleteFile('/ws/a.yaml'))
    await act(async () => { await result.current.ws.confirmDeleteFile() })
    expect(result.current.tabs.tabs.map(t => t.filePath)).toEqual(['/ws/b.yaml'])
    expect(result.current.tabs.activeTabId).toBe(keep)

    act(() => result.current.ws.handleDeleteFile('/ws/not-open.yaml'))
    await act(async () => { await result.current.ws.confirmDeleteFile() })
    expect(result.current.tabs.tabs).toHaveLength(1)
  })

  it('confirmDeleteFile without a pending file does nothing, and failures are logged', async () => {
    const { result } = setup()
    await act(async () => { await result.current.ws.confirmDeleteFile() })
    expect(b.deleteRequest).not.toHaveBeenCalled()
    b.deleteRequest.mockRejectedValue(new Error('x'))
    act(() => result.current.ws.handleDeleteFile('/ws/a.yaml'))
    await act(async () => { await result.current.ws.confirmDeleteFile() })
    expect(console.error).toHaveBeenCalled()
  })

  it('renames update open tabs and keep them clean', async () => {
    primeRefresh()
    b.renameRequest.mockResolvedValue('/ws/new.yaml')
    const { result } = setup()
    act(() => { result.current.tabs.openFileAsTab(req({ title: 'old' }), '/ws/old.yaml') })
    await act(async () => { await result.current.ws.handleRenameFile('/ws/old.yaml', 'new') })
    expect(result.current.tabs.activeTab).toMatchObject({ name: 'new', filePath: '/ws/new.yaml' })
    expect(result.current.tabs.dirtyFilePaths.size).toBe(0)

    b.renameRequest.mockResolvedValue('')
    await act(async () => { await result.current.ws.handleRenameFile('/ws/new.yaml', 'x') })
    expect(result.current.tabs.activeTab.name).toBe('new')
    b.renameRequest.mockRejectedValue(new Error('x'))
    await act(async () => { await result.current.ws.handleRenameFile('/ws/new.yaml', 'x') })
    expect(console.error).toHaveBeenCalled()
  })

  it('renaming a request that has a timeout does not mark the tab dirty', async () => {
    primeRefresh()
    b.renameRequest.mockResolvedValue('/ws/new.yaml')
    const { result } = setup()
    act(() => { result.current.tabs.openFileAsTab(req({ title: 'old', timeoutMs: 5000 }), '/ws/old.yaml') })
    expect(result.current.tabs.dirtyFilePaths.size).toBe(0)
    await act(async () => { await result.current.ws.handleRenameFile('/ws/old.yaml', 'new') })
    expect(result.current.tabs.dirtyFilePaths.size).toBe(0)
  })

  it('renaming a folder rebases file paths and target dirs of open tabs', async () => {
    primeRefresh()
    b.renameFolder.mockResolvedValue('/ws/requests/renamed')
    const { result } = setup()
    act(() => { result.current.tabs.openFileAsTab(req({ title: 'a' }), '/ws/requests/old/a.yaml') })
    act(() => { result.current.tabs.openFileAsTab(req({ title: 'b' }), '/ws/requests/old-other/b.yaml') })
    act(() => result.current.tabs.handleNewRequestInFolder('/ws/requests/old'))
    await act(async () => { await result.current.ws.handleRenameFolder('/ws/requests/old', 'renamed') })

    const byName = Object.fromEntries(result.current.tabs.tabs.map(t => [t.name ?? 'new', t]))
    expect(byName.a.filePath).toBe('/ws/requests/renamed/a.yaml')
    expect(byName.b.filePath).toBe('/ws/requests/old-other/b.yaml')
    expect(byName.new.targetDir).toBe('/ws/requests/renamed')

    b.renameFolder.mockResolvedValue('')
    await act(async () => { await result.current.ws.handleRenameFolder('/ws/requests/renamed', 'x') })
    b.renameFolder.mockRejectedValue(new Error('x'))
    await act(async () => { await result.current.ws.handleRenameFolder('/ws/requests/renamed', 'x') })
    expect(console.error).toHaveBeenCalled()
  })

  it('moving a file repoints its tab', async () => {
    primeRefresh()
    b.moveRequest.mockResolvedValue('/ws/requests/dest/a.yaml')
    const { result } = setup()
    act(() => { result.current.tabs.openFileAsTab(req(), '/ws/requests/a.yaml') })
    await act(async () => { await result.current.ws.handleMoveFile('/ws/requests/a.yaml', '/ws/requests/dest') })
    expect(result.current.tabs.activeTab.filePath).toBe('/ws/requests/dest/a.yaml')
    b.moveRequest.mockResolvedValue('')
    await act(async () => { await result.current.ws.handleMoveFile('/ws/requests/dest/a.yaml', '/x') })
    expect(result.current.tabs.activeTab.filePath).toBe('/ws/requests/dest/a.yaml')
    b.moveRequest.mockRejectedValue(new Error('x'))
    await act(async () => { await result.current.ws.handleMoveFile('/a', '/b') })
    expect(console.error).toHaveBeenCalled()
  })
})

describe('folders', () => {
  it('create-folder dialog derives a label and validates the name', async () => {
    primeRefresh()
    const { result } = setup()
    act(() => result.current.ws.handleCreateFolder('C:\\ws\\requests\\auth'))
    expect(result.current.ws.newFolderDialog).toEqual({ parentPath: 'C:\\ws\\requests\\auth', parentLabel: 'auth' })
    act(() => result.current.ws.handleCreateFolder(''))
    expect(result.current.ws.newFolderDialog?.parentLabel).toBe('requests')

    await act(async () => { await result.current.ws.confirmCreateFolder() })
    expect(b.createFolder).not.toHaveBeenCalled()

    act(() => result.current.ws.setNewFolderName('  new  '))
    await act(async () => { await result.current.ws.confirmCreateFolder() })
    expect(b.createFolder).toHaveBeenCalledWith('', 'new')
    expect(result.current.ws.newFolderDialog).toBeNull()
  })

  it('keeps the dialog open and shows the error when creation fails', async () => {
    b.createFolder.mockRejectedValueOnce(new Error('a folder named "x" already exists')).mockRejectedValueOnce('plain string')
    const { result } = setup()
    act(() => result.current.ws.handleCreateFolder('/ws/requests'))
    act(() => result.current.ws.setNewFolderName('x'))
    await act(async () => { await result.current.ws.confirmCreateFolder() })
    expect(result.current.ws.newFolderError).toBe('a folder named "x" already exists')
    expect(result.current.ws.newFolderDialog).not.toBeNull()
    await act(async () => { await result.current.ws.confirmCreateFolder() })
    expect(result.current.ws.newFolderError).toBe('plain string')
  })

  it('delete folder confirms first', async () => {
    primeRefresh()
    const { result } = setup()
    await act(async () => { await result.current.ws.confirmDeleteFolder() })
    expect(b.deleteFolder).not.toHaveBeenCalled()
    act(() => result.current.ws.handleDeleteFolder('/ws/requests/f'))
    expect(result.current.ws.deletingFolderPath).toBe('/ws/requests/f')
    await act(async () => { await result.current.ws.confirmDeleteFolder() })
    expect(b.deleteFolder).toHaveBeenCalledWith('/ws/requests/f')
    expect(result.current.ws.deletingFolderPath).toBeNull()

    b.deleteFolder.mockRejectedValue(new Error('x'))
    act(() => result.current.ws.handleDeleteFolder('/ws/requests/g'))
    await act(async () => { await result.current.ws.confirmDeleteFolder() })
    expect(console.error).toHaveBeenCalled()
  })
})

describe('imports', () => {
  beforeEach(() => { vi.useFakeTimers({ shouldAdvanceTime: true }) })
  afterEach(() => { vi.useRealTimers() })

  it('OpenAPI import refreshes on success and toggles the importing flag', async () => {
    primeRefresh()
    b.importOpenAPISpec.mockResolvedValue({ imported: [] } as never)
    const { result } = setup()
    await act(async () => { await result.current.ws.handleImportOpenAPI() })
    expect(b.listRequests).toHaveBeenCalled()
    expect(result.current.ws.importing).toBe(false)
    expect(result.current.ws.importError).toBe('')
  })

  it('OpenAPI import errors are shown and auto-dismissed after 8 seconds', async () => {
    b.importOpenAPISpec.mockResolvedValue({ error: 'bad spec' } as never)
    const { result } = setup()
    await act(async () => { await result.current.ws.handleImportOpenAPI() })
    expect(result.current.ws.importError).toBe('bad spec')
    act(() => { vi.advanceTimersByTime(8001) })
    expect(result.current.ws.importError).toBe('')

    b.importOpenAPISpec.mockRejectedValue(new Error('thrown'))
    await act(async () => { await result.current.ws.handleImportOpenAPI() })
    expect(result.current.ws.importError).toBe('thrown')
    b.importOpenAPISpec.mockRejectedValue('text')
    await act(async () => { await result.current.ws.handleImportOpenAPI() })
    expect(result.current.ws.importError).toBe('text')
    expect(result.current.ws.importing).toBe(false)
  })

  it('import from URL ignores blank input, reopens the dialog on error, and clears on success', async () => {
    primeRefresh()
    const { result } = setup()
    await act(async () => { await result.current.ws.handleImportFromURL() })
    expect(b.importOpenAPISpecFromURL).not.toHaveBeenCalled()

    act(() => { result.current.ws.setImportFromURLInput('  https://x.test/spec.yaml  ') })
    b.importOpenAPISpecFromURL.mockResolvedValueOnce({ error: '404' } as never)
    await act(async () => { await result.current.ws.handleImportFromURL() })
    expect(b.importOpenAPISpecFromURL).toHaveBeenCalledWith('https://x.test/spec.yaml')
    expect(result.current.ws.importFromURLError).toBe('404')
    expect(result.current.ws.importFromURLDialog).toBe(true)
    expect(result.current.ws.importFromURLInput).not.toBe('')

    b.importOpenAPISpecFromURL.mockResolvedValueOnce({ imported: [] } as never)
    await act(async () => { await result.current.ws.handleImportFromURL() })
    expect(result.current.ws.importFromURLInput).toBe('')

    b.importOpenAPISpecFromURL.mockRejectedValueOnce(new Error('x'))
    act(() => { result.current.ws.setImportFromURLInput('https://y.test') })
    await act(async () => { await result.current.ws.handleImportFromURL() })
    expect(result.current.ws.importing).toBe(false)
  })

  it.each([
    ['handleImportPostman', () => b.importPostmanCollection],
    ['handleImportInsomnia', () => b.importInsomniaExport],
  ] as const)('%s refreshes only when the import succeeded', async (name, mock) => {
    primeRefresh()
    const { result } = setup()
    mock().mockResolvedValueOnce({ error: 'nope' } as never)
    await act(async () => { await result.current.ws[name]() })
    expect(b.listRequests).not.toHaveBeenCalled()

    mock().mockResolvedValueOnce({ imported: [] } as never)
    await act(async () => { await result.current.ws[name]() })
    expect(b.listRequests).toHaveBeenCalledTimes(1)

    mock().mockRejectedValueOnce(new Error('x'))
    await act(async () => { await result.current.ws[name]() })
    expect(result.current.ws.importing).toBe(false)
  })
})
