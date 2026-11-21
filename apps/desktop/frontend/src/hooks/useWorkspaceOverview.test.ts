import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useWorkspaceOverview } from './useWorkspaceOverview'
import type { WorkspaceOverview } from '@/lib/bridge'

const bridge = vi.hoisted(() => ({ getWorkspaceOverview: vi.fn() }))
vi.mock('@/lib/bridge', () => bridge)

const overview = { open: true, path: '/ws/shop', name: 'shop', attention: [] } as unknown as WorkspaceOverview

describe('useWorkspaceOverview', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getWorkspaceOverview.mockResolvedValue(overview)
  })

  it('reads the workspace when it is shown and keeps the snapshot', async () => {
    const { result } = renderHook(() => useWorkspaceOverview({ workspacePath: '/ws/shop', enabled: true }))
    await waitFor(() => expect(result.current.overview).toBe(overview))
    expect(result.current.error).toBeNull()
    expect(bridge.getWorkspaceOverview).toHaveBeenCalledTimes(1)
  })

  it('does not read while nothing that shows it is on screen, and reads when that changes', async () => {
    const { rerender } = renderHook(p => useWorkspaceOverview(p), { initialProps: { workspacePath: '/ws/shop', enabled: false, refreshTrigger: 0 } })
    expect(bridge.getWorkspaceOverview).not.toHaveBeenCalled()
    rerender({ workspacePath: '/ws/shop', enabled: false, refreshTrigger: 5 })
    expect(bridge.getWorkspaceOverview).not.toHaveBeenCalled()
    rerender({ workspacePath: '/ws/shop', enabled: true, refreshTrigger: 5 })
    await waitFor(() => expect(bridge.getWorkspaceOverview).toHaveBeenCalledTimes(1))
  })

  it('reads again when the workspace or the trigger changes, and on reload', async () => {
    const { result, rerender } = renderHook(p => useWorkspaceOverview(p), { initialProps: { workspacePath: '/ws/shop', enabled: true, refreshTrigger: 0 } })
    await waitFor(() => expect(bridge.getWorkspaceOverview).toHaveBeenCalledTimes(1))
    rerender({ workspacePath: '/ws/shop', enabled: true, refreshTrigger: 1 })
    await waitFor(() => expect(bridge.getWorkspaceOverview).toHaveBeenCalledTimes(2))
    rerender({ workspacePath: '/ws/other', enabled: true, refreshTrigger: 1 })
    await waitFor(() => expect(bridge.getWorkspaceOverview).toHaveBeenCalledTimes(3))
    await act(async () => { await result.current.reload() })
    expect(bridge.getWorkspaceOverview).toHaveBeenCalledTimes(4)
  })

  it('does not read with no workspace open', () => {
    renderHook(() => useWorkspaceOverview({ workspacePath: '', enabled: true }))
    expect(bridge.getWorkspaceOverview).not.toHaveBeenCalled()
  })

  it('reports a failed read', async () => {
    bridge.getWorkspaceOverview.mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useWorkspaceOverview({ workspacePath: '/ws/shop', enabled: true }))
    await waitFor(() => expect(result.current.error).toBe('boom'))
  })
})
