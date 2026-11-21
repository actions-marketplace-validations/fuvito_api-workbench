import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { WorkspaceTab, openSummary } from './WorkspaceTab'
import type { WorkspaceOverview } from '@/lib/bridge'

const bridge = vi.hoisted(() => ({
  getWorkspaceOverview: vi.fn(),
  openInExplorer: vi.fn(),
}))
vi.mock('@/lib/bridge', () => bridge)

const base: WorkspaceOverview = {
  open: true, path: '/ws/shop', name: 'shop', temporary: false,
  requests: 12, scenarios: 5, environments: 2,
  environmentName: 'Staging', environmentPath: '/ws/shop/environments/staging.yaml', environmentSource: 'remembered',
  envVars: [
    { name: 'baseUrl', secret: false, set: true },
    { name: 'token', secret: true, set: false, unsetRef: 'API_TOKEN' },
  ],
  specs: [
    { ref: '../specs/api.yaml', state: 'found', scenarios: ['/ws/shop/scenarios/a.yaml'], coverage: { total: 10, covered: 6, notCovered: 4, passing: 5, failing: 1 } },
    { ref: 'https://example.com/openapi.yaml', state: 'url', scenarios: ['/ws/shop/scenarios/b.yaml'] },
  ],
  lastRun: { at: '2026-11-21T10:00:00Z', scenario: 'Checkout', passed: false, scenarios: 5, failingNow: 1 },
  attention: [],
  snapshotAt: '2026-11-21T10:05:00Z',
}

const handlers = {
  onOpenScenario: vi.fn(),
  onOpenContracts: vi.fn(),
  onShowPanel: vi.fn(),
  onCloseWorkspace: vi.fn(),
}

function show(o: WorkspaceOverview) {
  bridge.getWorkspaceOverview.mockResolvedValue(o)
  return render(<WorkspaceTab workspacePath={o.path} {...handlers} />)
}

describe('WorkspaceTab', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('shows the contents, the environment and where its choice came from', async () => {
    show(base)
    expect(await screen.findByText('shop')).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Contents' })).toHaveTextContent('12 requests')
    expect(screen.getByRole('region', { name: 'Contents' })).toHaveTextContent('5 scenarios')
    expect(screen.getByTestId('env-source')).toHaveTextContent('Staging')
    expect(screen.getByTestId('env-source')).toHaveTextContent('remembered for this workspace')
  })

  it('lists environment variables as set or unset and never shows a value', async () => {
    show(base)
    const list = await screen.findByRole('list', { name: 'Environment variables' })
    expect(within(list).getByText('baseUrl').closest('li')).toHaveTextContent('set')
    expect(within(list).getByText('token').closest('li')).toHaveTextContent('unset ${API_TOKEN}')
    expect(within(list).getByLabelText('secret')).toBeInTheDocument()
  })

  it('says when nothing needs attention', async () => {
    show(base)
    expect(await screen.findByTestId('attention-empty')).toHaveTextContent('Nothing needs attention')
  })

  it('shows each attention item as what, where, why and fix, with a button that goes to the fix', async () => {
    show({
      ...base,
      attention: [{
        id: 'spec-missing', severity: 'warning',
        what: 'The spec file "../specs/gone.yaml" does not exist.',
        where: 'used by scenarios/b.yaml', why: 'Responses are not checked against a spec.', fix: 'Choose the spec file in the Setup row.',
        target: { kind: 'scenario', path: '/ws/shop/scenarios/b.yaml' },
      }],
    })
    const card = await screen.findByRole('article', { name: /does not exist/ })
    expect(card).toHaveTextContent('used by scenarios/b.yaml')
    expect(card).toHaveTextContent('Responses are not checked')
    expect(card).toHaveTextContent('Choose the spec file')
    fireEvent.click(within(card).getByRole('button', { name: 'Open scenario' }))
    expect(handlers.onOpenScenario).toHaveBeenCalledWith('/ws/shop/scenarios/b.yaml', 'b')
  })

  it('offers Contracts for a failing spec and the file manager for an environment file', async () => {
    show({
      ...base,
      attention: [
        { id: 'spec-failing', severity: 'warning', what: '1 of 6 failed', where: 'spec ../specs/api.yaml', why: 'w', fix: 'f', target: { kind: 'contracts', path: '../specs/api.yaml' } },
        { id: 'unset-env-var', severity: 'warning', what: 'API_TOKEN is not set', where: 'environment', why: 'w', fix: 'f', target: { kind: 'environment', path: '/ws/shop/environments/staging.yaml' } },
        { id: 'no-environment', severity: 'warning', what: 'No environment is selected.', where: 'w', why: 'w', fix: 'Choose an environment in the top bar.', target: { kind: 'environment', path: '' } },
      ],
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Open Contracts' }))
    expect(handlers.onOpenContracts).toHaveBeenCalledWith('../specs/api.yaml')
    fireEvent.click(screen.getByRole('button', { name: 'Show environment file' }))
    expect(bridge.openInExplorer).toHaveBeenCalledWith('/ws/shop/environments/staging.yaml')
    expect(screen.getAllByRole('button', { name: /Show environment file/ })).toHaveLength(1)
  })

  it('shows specs with their state and the four coverage totals', async () => {
    show(base)
    const specs = await screen.findByRole('region', { name: 'Specs' })
    expect(specs).toHaveTextContent('found')
    expect(specs).toHaveTextContent('URL, not checked')
    expect(specs).toHaveTextContent('Covered 6, Not covered 4, Passing 5, Failing 1')
    expect(specs).toHaveTextContent('no coverage recorded yet')
    fireEvent.click(within(specs).getByRole('button', { name: /Covered 6/ }))
    expect(handlers.onOpenContracts).toHaveBeenCalledWith('../specs/api.yaml')
  })

  it('shows the last run, and an empty message when there is none', async () => {
    const { unmount } = show(base)
    expect(await screen.findByTestId('last-run')).toHaveTextContent('FAILED')
    expect(screen.getByTestId('last-run')).toHaveTextContent('Checkout')
    expect(screen.getByTestId('last-run')).toHaveTextContent('1 failed the last time')
    unmount()
    show({ ...base, lastRun: undefined })
    expect(await screen.findByText('No scenario has run in this workspace yet.')).toBeInTheDocument()
  })

  it('opens with a reminder naming the environment source and the number of items, and it can be dismissed', async () => {
    show({ ...base, attention: [{ id: 'x', severity: 'info', what: 'a', where: 'b', why: 'c', fix: 'd', target: { kind: 'scenario', path: '' } }] })
    const reminder = await screen.findByTestId('workspace-reminder')
    expect(reminder).toHaveTextContent('Environment: Staging (remembered) · 1 thing needs attention')
    fireEvent.click(screen.getByRole('button', { name: 'Dismiss reminder' }))
    expect(screen.queryByTestId('workspace-reminder')).not.toBeInTheDocument()
  })

  it('marks a temporary workspace', async () => {
    show({ ...base, temporary: true, environmentSource: 'temporary' })
    expect(await screen.findByText('temporary')).toBeInTheDocument()
    expect(screen.getByTestId('env-source')).toHaveTextContent('temporary workspace, not remembered')
  })

  it('reads the workspace again on Refresh and when the workspace changes', async () => {
    const { rerender } = show(base)
    await screen.findByText('shop')
    expect(bridge.getWorkspaceOverview).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByRole('button', { name: /Refresh/ }))
    await waitFor(() => expect(bridge.getWorkspaceOverview).toHaveBeenCalledTimes(2))
    rerender(<WorkspaceTab workspacePath="/ws/other" {...handlers} />)
    await waitFor(() => expect(bridge.getWorkspaceOverview).toHaveBeenCalledTimes(3))
  })

  it('shows contents links, close workspace and a load error', async () => {
    show(base)
    fireEvent.click(await screen.findByRole('button', { name: '12 requests' }))
    expect(handlers.onShowPanel).toHaveBeenCalledWith('explorer')
    fireEvent.click(screen.getByRole('button', { name: '5 scenarios' }))
    expect(handlers.onShowPanel).toHaveBeenCalledWith('scenarios')
    fireEvent.click(screen.getByRole('button', { name: 'Close workspace' }))
    expect(handlers.onCloseWorkspace).toHaveBeenCalled()
  })

  it('reports a failed read instead of staying blank', async () => {
    bridge.getWorkspaceOverview.mockRejectedValue(new Error('boom'))
    render(<WorkspaceTab workspacePath="/ws/shop" {...handlers} />)
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })
})

describe('openSummary', () => {
  it('words each environment source and the attention count', () => {
    expect(openSummary({ ...base, environmentSource: 'chosen' })).toBe('Environment: Staging (chosen) · nothing needs attention')
    expect(openSummary({ ...base, environmentName: undefined, environmentSource: 'none' })).toBe('Environment: none chosen · nothing needs attention')
    const two = { ...base, attention: [base.attention, base.attention].flat().concat([{ id: 'a' }, { id: 'b' }] as never) }
    expect(openSummary(two)).toContain('2 things need attention')
  })
})
