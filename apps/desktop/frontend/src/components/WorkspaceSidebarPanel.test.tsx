import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/react'
import { WorkspaceSidebarPanel } from './WorkspaceSidebarPanel'
import type { WorkspaceOverview } from '@/lib/bridge'

const item = { id: 'spec-missing', severity: 'warning' as const, what: 'The spec file "../specs/gone.yaml" does not exist.', where: 'w', why: 'y', fix: 'Choose the spec file.', target: { kind: 'scenario' as const, path: '/ws/a.yaml' } }

const base = {
  open: true, path: '/ws/shop', name: 'shop', temporary: false,
  requests: 12, scenarios: 1, environments: 2,
  environmentName: 'Staging', environmentPath: '/ws/shop/environments/staging.yaml', environmentSource: 'remembered',
  envVars: [], specs: [{}, {}], attention: [], snapshotAt: '2026-11-21T10:00:00Z',
} as unknown as WorkspaceOverview

const handlers = { onRefresh: vi.fn(), onEnvChange: vi.fn(), onShowAttention: vi.fn(), onShowIcon: vi.fn() }
const envs = [{ name: 'Local', path: '/ws/shop/environments/local.yaml' }, { name: 'Staging', path: '/ws/shop/environments/staging.yaml' }]

function show(over: Partial<React.ComponentProps<typeof WorkspaceSidebarPanel>> = {}) {
  return render(<WorkspaceSidebarPanel overview={base} error={null} loading={false} environments={envs} activeEnvPath="/ws/shop/environments/staging.yaml" {...handlers} {...over} />)
}

describe('WorkspaceSidebarPanel', () => {
  beforeEach(() => vi.clearAllMocks())

  it('shows the workspace name, a temporary badge and where the environment choice came from', () => {
    show({ overview: { ...base, temporary: true } })
    expect(screen.getByText('shop')).toBeInTheDocument()
    expect(screen.getByText('temporary')).toBeInTheDocument()
    expect(screen.getByTestId('sidebar-env-source')).toHaveTextContent('remembered for this workspace')
  })

  it('selects an environment through the same handler as the status bar', () => {
    show()
    const select = screen.getByRole('combobox', { name: 'Environment' })
    expect(select).toHaveValue('/ws/shop/environments/staging.yaml')
    fireEvent.change(select, { target: { value: '/ws/shop/environments/local.yaml' } })
    expect(handlers.onEnvChange).toHaveBeenCalledWith('/ws/shop/environments/local.yaml')
    fireEvent.change(select, { target: { value: '' } })
    expect(handlers.onEnvChange).toHaveBeenCalledWith('')
  })

  it('says when nothing needs attention', () => {
    show()
    expect(screen.getByTestId('sidebar-attention-empty')).toHaveTextContent('Nothing needs attention')
  })

  it('lists attention items and shows the one that is clicked on the page', () => {
    show({ overview: { ...base, attention: [item, { ...item, severity: 'info', what: 'Second thing' }] } })
    const list = screen.getByRole('list', { name: 'Needs attention' })
    expect(within(list).getAllByRole('button')).toHaveLength(2)
    fireEvent.click(within(list).getByRole('button', { name: /Second thing/ }))
    expect(handlers.onShowAttention).toHaveBeenCalledWith(1)
  })

  it('links the counts to the matching rail icon', () => {
    show()
    fireEvent.click(screen.getByRole('button', { name: '12 requests' }))
    fireEvent.click(screen.getByRole('button', { name: '1 scenario' }))
    expect(handlers.onShowIcon).toHaveBeenNthCalledWith(1, 'explorer')
    expect(handlers.onShowIcon).toHaveBeenNthCalledWith(2, 'scenarios')
    expect(screen.getByText('2 specs')).toBeInTheDocument()
  })

  it('refreshes, and says when it is reading or failed', () => {
    const { unmount } = show()
    fireEvent.click(screen.getByRole('button', { name: 'Refresh workspace' }))
    expect(handlers.onRefresh).toHaveBeenCalledTimes(1)
    unmount()
    show({ overview: null })
    expect(screen.getByText('Reading the workspace…')).toBeInTheDocument()
  })

  it('shows a read error', () => {
    show({ overview: null, error: 'boom' })
    expect(screen.getByRole('alert')).toHaveTextContent('boom')
  })
})
