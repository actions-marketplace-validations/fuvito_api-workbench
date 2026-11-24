import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/react'
import { WorkspaceTab, openSummary } from './WorkspaceTab'
import type { WorkspaceOverview } from '@/lib/bridge'

const bridge = vi.hoisted(() => ({
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
    { ref: '../specs/api.yaml', refs: ['../specs/api.yaml'], path: '/ws/shop/specs/api.yaml', state: 'found', scenarios: ['/ws/shop/scenarios/a.yaml'], coverage: [{ ref: '../specs/api.yaml', total: 10, covered: 6, notCovered: 4, passing: 5, failing: 1 }] },
    { ref: 'https://example.com/openapi.yaml', refs: ['https://example.com/openapi.yaml'], state: 'url', scenarios: ['/ws/shop/scenarios/b.yaml'], coverage: [] },
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
  onRefresh: vi.fn(),
}

function show(o: WorkspaceOverview, extra: Partial<React.ComponentProps<typeof WorkspaceTab>> = {}) {
  return render(<WorkspaceTab overview={o} error={null} loading={false} {...handlers} {...extra} />)
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

  it('shows every Contracts card of a spec that scenarios spell differently, each opening its own card', async () => {
    show({
      ...base,
      specs: [{
        ref: 'a/openapi.yaml', refs: ['a/openapi.yaml', 'b/openapi.yaml'], path: '/ws/shop/specs/openapi.yaml', state: 'found', scenarios: ['/ws/shop/scenarios/a.yaml', '/ws/shop/scenarios/b.yaml'],
        coverage: [
          { ref: 'a/openapi.yaml', total: 4, covered: 4, notCovered: 0, passing: 4, failing: 0 },
          { ref: 'b/openapi.yaml', total: 4, covered: 1, notCovered: 3, passing: 0, failing: 1 },
        ],
      }],
    })
    const specs = await screen.findByRole('region', { name: 'Specs' })
    const buttons = within(specs).getAllByRole('button', { name: /Covered/ })
    expect(buttons).toHaveLength(2)
    fireEvent.click(buttons[1])
    expect(handlers.onOpenContracts).toHaveBeenCalledWith('b/openapi.yaml')
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

  it('keeps a dismissed reminder dismissed when the tab is closed and opened again, per workspace', async () => {
    const mine = { ...base, path: '/ws/remount', name: 'remount', attention: [] }
    const first = show(mine)
    fireEvent.click(await screen.findByRole('button', { name: 'Dismiss reminder' }))
    first.unmount()
    const second = show(mine)
    await screen.findByText('remount')
    expect(screen.queryByTestId('workspace-reminder')).not.toBeInTheDocument()
    second.unmount()
    show({ ...mine, path: '/ws/other', name: 'other' })
    expect(await screen.findByTestId('workspace-reminder')).toBeInTheDocument()
  })

  it('marks a temporary workspace', async () => {
    show({ ...base, temporary: true, environmentSource: 'temporary' })
    expect(await screen.findByText('temporary')).toBeInTheDocument()
    expect(screen.getByTestId('env-source')).toHaveTextContent('temporary workspace, not remembered')
  })

  it('asks the app to read the workspace again on Refresh', async () => {
    show(base)
    fireEvent.click(await screen.findByRole('button', { name: /Refresh/ }))
    expect(handlers.onRefresh).toHaveBeenCalledTimes(1)
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

  it('scrolls to the attention card named by the sidebar', () => {
    const scrolled = vi.fn()
    Element.prototype.scrollIntoView = scrolled
    const item = { id: 'x', severity: 'info' as const, what: 'a', where: 'b', why: 'c', fix: 'd', target: { kind: 'scenario' as const, path: '' } }
    const o = { ...base, attention: [item, { ...item, what: 'second' }] }
    const { rerender } = show(o)
    expect(scrolled).not.toHaveBeenCalled()
    rerender(<WorkspaceTab overview={o} error={null} loading={false} focusAttention={{ index: 1, seq: 1 }} {...handlers} />)
    expect(scrolled).toHaveBeenCalledTimes(1)
    expect(scrolled.mock.instances[0]).toHaveAttribute('data-attention-index', '1')
  })

  it('reports a failed read instead of staying blank', async () => {
    render(<WorkspaceTab overview={null} error="boom" loading={false} {...handlers} />)
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

describe('WorkspaceTab project file card', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('suggests clrsig init when the workspace has no project file', () => {
    show({ ...base, projectFile: { found: false, keys: [] } })
    const card = screen.getByTestId('project-file-card')
    expect(card).toHaveTextContent('No clrsig.yaml here')
    expect(screen.getByTestId('init-command')).toHaveTextContent('clrsig init')
  })

  it('says nothing about it when the snapshot has no project file information (older data)', () => {
    show(base)
    expect(screen.getByTestId('project-file-card')).toHaveTextContent('No clrsig.yaml here')
  })

  it('lists what a usable project file gives', () => {
    show({ ...base, projectFile: { found: true, path: 'clrsig.yaml', keys: ['spec', 'baseUrl', 'conform.readOnly'] } })
    const card = screen.getByTestId('project-file-card')
    expect(card).toHaveTextContent('clrsig.yaml')
    expect(card).toHaveTextContent('Gives: spec, baseUrl, conform.readOnly.')
    expect(card).toHaveTextContent('clrsig config show')
    expect(card).not.toHaveTextContent('No clrsig.yaml here')
  })

  it('points to Needs attention when the file cannot be used, and offers to show it', () => {
    show({
      ...base,
      projectFile: { found: true, path: 'clrsig.yaml', error: 'unknown key "baseURL"', keys: [] },
      attention: [{ id: 'project-file-invalid', severity: 'warning', what: 'The project file cannot be used.', where: 'clrsig.yaml', why: 'unknown key "baseURL"', fix: 'Correct the key the message names.', target: { kind: 'projectfile', path: '/ws/shop/clrsig.yaml' } }],
    })
    expect(screen.getByTestId('project-file-card')).toHaveTextContent('Cannot be used: see Needs attention.')
    fireEvent.click(screen.getByRole('button', { name: 'Show project file' }))
    expect(bridge.openInExplorer).toHaveBeenCalledWith('/ws/shop/clrsig.yaml')
  })
})

describe('WorkspaceTab weak scenarios', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('shows each weak spot of a scenario on its own line, as information, with a button to open it', () => {
    show({
      ...base,
      attention: [{
        id: 'scenario-weak', severity: 'info', what: 'The scenario passes its checks but is weak: 2 things to improve.', where: 'scenarios/weak.yaml',
        why: '[step 1] Ping: has no assertions and extracts nothing\n[step 2] Login: extracts "spare", but no later step uses {{spare}}',
        fix: 'Add an assertion, then run clrsig validate --strict to check it again.', target: { kind: 'scenario', path: '/ws/shop/scenarios/weak.yaml' },
      }],
    })
    const card = screen.getByRole('article', { name: /is weak/ })
    expect(card).toHaveTextContent('[step 1] Ping: has no assertions')
    expect(card).toHaveTextContent('extracts "spare"')
    expect(within(card).getByText(/\[step 1\] Ping/).className).toContain('whitespace-pre-line')
    fireEvent.click(within(card).getByRole('button', { name: 'Open scenario' }))
    expect(handlers.onOpenScenario).toHaveBeenCalledWith('/ws/shop/scenarios/weak.yaml', 'weak')
  })
})
