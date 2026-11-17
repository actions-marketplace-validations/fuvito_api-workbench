import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useState } from 'react'
import type { FileNode } from '@/lib/bridge'
import { createBlankTab } from '@/lib/tabState'
import type { TabState } from '@/lib/tabState'

vi.mock('@/lib/bridge', () => ({ startDemo: vi.fn(), stopDemo: vi.fn() }))

import { startDemo, stopDemo } from '@/lib/bridge'
import { useDemo } from './useDemo'

const tree: FileNode[] = [{ name: 'a.yaml', path: '/tmp/demo/requests/a.yaml', isDir: false }]

function tab(over: Partial<TabState>): TabState {
  return { ...createBlankTab(), ...over }
}

function setup(initial: TabState[] = [createBlankTab()]) {
  const refreshWorkspace = vi.fn(async () => tree)
  const refreshHistory = vi.fn(async () => {})
  const openFirstFileInTree = vi.fn(async () => {})
  const onWorkspaceOpened = vi.fn()
  const hook = renderHook(() => {
    const [state, setTabsState] = useState({ tabs: initial, activeTabId: initial[0].id })
    const demo = useDemo({ refreshWorkspace, refreshHistory, setTabsState, openFirstFileInTree, onWorkspaceOpened })
    return { state, demo }
  })
  return { ...hook, refreshWorkspace, refreshHistory, openFirstFileInTree, onWorkspaceOpened }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.spyOn(console, 'error').mockImplementation(() => {})
})
afterEach(() => vi.restoreAllMocks())

describe('handleStartDemo', () => {
  it('records the port and workspace, then opens the first request', async () => {
    vi.mocked(startDemo).mockResolvedValue({ port: 8090, workspacePath: '/tmp/demo' } as never)
    const s = setup()
    await act(async () => { await s.result.current.demo.handleStartDemo() })

    expect(s.result.current.demo).toMatchObject({ demoRunning: true, demoPort: 8090, demoWorkspacePath: '/tmp/demo', demoStarting: false })
    expect(s.openFirstFileInTree).toHaveBeenCalledWith(tree)
    expect(s.refreshHistory).toHaveBeenCalled()
    expect(s.onWorkspaceOpened).toHaveBeenCalledTimes(1)
  })

  it('does not start twice', async () => {
    vi.mocked(startDemo).mockResolvedValue({ port: 8090, workspacePath: '/tmp/demo' } as never)
    const s = setup()
    await act(async () => { await s.result.current.demo.handleStartDemo() })
    await act(async () => { await s.result.current.demo.handleStartDemo() })
    expect(startDemo).toHaveBeenCalledTimes(1)
  })

  it('ignores a second click while the first is still starting', async () => {
    let release: (v: unknown) => void = () => {}
    vi.mocked(startDemo).mockReturnValue(new Promise(r => { release = r }) as never)
    const s = setup()
    let first: Promise<void> = Promise.resolve()
    act(() => { first = s.result.current.demo.handleStartDemo() })
    expect(s.result.current.demo.demoStarting).toBe(true)
    await act(async () => { await s.result.current.demo.handleStartDemo() })
    expect(startDemo).toHaveBeenCalledTimes(1)
    await act(async () => { release({ port: 1, workspacePath: '' }); await first })
    expect(s.result.current.demo.demoStarting).toBe(false)
  })

  it('stays stopped when the backend reports an error', async () => {
    vi.mocked(startDemo).mockResolvedValue({ port: 0, error: 'port busy' } as never)
    const s = setup()
    await act(async () => { await s.result.current.demo.handleStartDemo() })
    expect(s.result.current.demo).toMatchObject({ demoRunning: false, demoStarting: false })
    expect(s.onWorkspaceOpened).not.toHaveBeenCalled()
    expect(console.error).toHaveBeenCalled()
  })

  it('recovers when the call throws', async () => {
    vi.mocked(startDemo).mockRejectedValue(new Error('ipc'))
    const s = setup()
    await act(async () => { await s.result.current.demo.handleStartDemo() })
    expect(s.result.current.demo.demoStarting).toBe(false)
    expect(s.result.current.demo.demoRunning).toBe(false)
  })

  it('treats a missing workspace path as empty', async () => {
    vi.mocked(startDemo).mockResolvedValue({ port: 8090 } as never)
    const s = setup()
    await act(async () => { await s.result.current.demo.handleStartDemo() })
    expect(s.result.current.demo.demoWorkspacePath).toBe('')
  })
})

describe('handleStopDemo', () => {
  async function started(initial: TabState[]) {
    vi.mocked(startDemo).mockResolvedValue({ port: 8090, workspacePath: 'C:\\tmp\\demo' } as never)
    const s = setup(initial)
    await act(async () => { await s.result.current.demo.handleStartDemo() })
    return s
  }

  it('closes only demo tabs, matched by URL, file path or scenario path', async () => {
    const keep = tab({ id: 'keep', url: 'https://api.example.com', filePath: '/real/a.yaml' })
    const byUrl = tab({ id: 'url', url: 'http://localhost:8090/users' })
    const byFile = tab({ id: 'file', filePath: 'C:\\tmp\\demo\\requests\\a.yaml' })
    const byScenario = tab({ id: 'scn', kind: 'scenario', scenarioPath: 'C:\\tmp\\demo\\scenarios\\s.yaml' })
    const otherPort = tab({ id: 'port', url: 'http://localhost:9999/x' })
    const s = await started([keep, byUrl, byFile, byScenario, otherPort])

    await act(async () => { await s.result.current.demo.handleStopDemo() })

    expect(stopDemo).toHaveBeenCalled()
    expect(s.result.current.state.tabs.map(t => t.id)).toEqual(['keep', 'port'])
    expect(s.result.current.state.activeTabId).toBe('keep')
    expect(s.result.current.demo).toMatchObject({ demoRunning: false, demoPort: 0, demoWorkspacePath: '' })
    expect(s.refreshWorkspace).toHaveBeenCalledTimes(2)
  })

  it('moves the active tab to the last remaining one when the active tab was a demo tab', async () => {
    const a = tab({ id: 'a', url: 'https://one.test' })
    const b = tab({ id: 'b', url: 'https://two.test' })
    const d = tab({ id: 'd', url: 'http://localhost:8090/x' })
    const s = await started([d, a, b])
    await act(async () => { await s.result.current.demo.handleStopDemo() })
    expect(s.result.current.state.activeTabId).toBe('b')
  })

  it('leaves a single blank tab when every tab belonged to the demo', async () => {
    const s = await started([tab({ id: 'd', url: 'http://localhost:8090/x' })])
    await act(async () => { await s.result.current.demo.handleStopDemo() })
    expect(s.result.current.state.tabs).toHaveLength(1)
    expect(s.result.current.state.tabs[0].url).toBe('')
    expect(s.result.current.state.tabs[0].id).toBe(s.result.current.state.activeTabId)
  })

  it('does not touch tabs when none belong to the demo', async () => {
    const t = tab({ id: 'x', url: 'https://example.com' })
    const s = await started([t])
    const before = s.result.current.state
    await act(async () => { await s.result.current.demo.handleStopDemo() })
    expect(s.result.current.state).toBe(before)
  })

  it('logs and keeps running state if the stop call fails', async () => {
    const s = await started([createBlankTab()])
    vi.mocked(stopDemo).mockRejectedValue(new Error('x'))
    await act(async () => { await s.result.current.demo.handleStopDemo() })
    expect(s.result.current.demo.demoRunning).toBe(true)
    expect(console.error).toHaveBeenCalled()
  })

  it('works when stopped without a recorded port or workspace', async () => {
    const t = tab({ id: 'x', url: 'http://localhost:8090/x', filePath: '/tmp/x' })
    const s = setup([t])
    await act(async () => { await s.result.current.demo.handleStopDemo() })
    expect(s.result.current.state.tabs).toHaveLength(1)
    expect(s.result.current.state.tabs[0].id).toBe('x')
  })
})

describe('clearDemoState', () => {
  it('resets without calling the backend', async () => {
    vi.mocked(startDemo).mockResolvedValue({ port: 8090, workspacePath: '/tmp/demo' } as never)
    const s = setup()
    await act(async () => { await s.result.current.demo.handleStartDemo() })
    act(() => s.result.current.demo.clearDemoState())
    expect(s.result.current.demo).toMatchObject({ demoRunning: false, demoPort: 0, demoWorkspacePath: '' })
    expect(stopDemo).not.toHaveBeenCalled()
  })
})
