import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/react'
import { ConformanceSidebarPanel } from './ConformanceSidebarPanel'

const bridge = vi.hoisted(() => ({ getConformContext: vi.fn() }))
vi.mock('@/lib/bridge', () => bridge)

const ctx = {
  workspace: '/ws',
  specs: [{ path: '/ws/specs/a.yaml', label: 'specs/a.yaml' }, { path: '/ws/specs/b.yaml', label: 'specs/b.yaml' }],
  environments: [{ path: '/ws/environments/local.yaml', label: 'local' }],
}
const handlers = { onSelectSpec: vi.fn(), onSelectEnv: vi.fn() }

function show(over: Partial<React.ComponentProps<typeof ConformanceSidebarPanel>> = {}) {
  return render(<ConformanceSidebarPanel workspacePath="/ws" selectedSpec="" selectedEnv="" lastResult={null} {...handlers} {...over} />)
}

describe('ConformanceSidebarPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getConformContext.mockResolvedValue(ctx)
  })

  it('lists the workspace specs and environments and picks them on the page', async () => {
    show()
    const specs = await screen.findByRole('list', { name: 'Workspace specs' })
    fireEvent.click(within(specs).getByRole('button', { name: 'specs/b.yaml' }))
    expect(handlers.onSelectSpec).toHaveBeenCalledWith('/ws/specs/b.yaml')
    const envs = screen.getByRole('list', { name: 'Workspace environments' })
    fireEvent.click(within(envs).getByRole('button', { name: 'local' }))
    expect(handlers.onSelectEnv).toHaveBeenCalledWith('/ws/environments/local.yaml')
    fireEvent.click(within(envs).getByRole('button', { name: 'Active environment' }))
    expect(handlers.onSelectEnv).toHaveBeenCalledWith('')
  })

  it('marks the spec and environment chosen on the page', async () => {
    show({ selectedSpec: '/ws/specs/a.yaml', selectedEnv: '' })
    const specs = await screen.findByRole('list', { name: 'Workspace specs' })
    expect(within(specs).getByRole('button', { name: 'specs/a.yaml' })).toHaveAttribute('aria-pressed', 'true')
    expect(within(specs).getByRole('button', { name: 'specs/b.yaml' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: 'Active environment' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('says when the workspace has no spec', async () => {
    bridge.getConformContext.mockResolvedValue({ ...ctx, specs: [] })
    show()
    expect(await screen.findByText(/No OpenAPI spec was found/)).toBeInTheDocument()
  })

  it('shows the last result and shows the page again when it is clicked', async () => {
    show({ lastResult: { specPath: '/ws/specs/a.yaml', specTitle: 'Shop', total: 7, covered: 5, passing: 4, failing: 1, failed: true } })
    const result = await screen.findByTestId('sidebar-last-result')
    expect(result).toHaveTextContent('Shop')
    expect(result).toHaveTextContent('Covered 5, Passing 4, Failing 1 of 7')
    fireEvent.click(result)
    expect(handlers.onSelectSpec).toHaveBeenCalledWith('/ws/specs/a.yaml')
  })

  it('reads the workspace again when it changes and reports a failed read', async () => {
    const { rerender } = show()
    await screen.findByRole('list', { name: 'Workspace specs' })
    rerender(<ConformanceSidebarPanel workspacePath="/other" selectedSpec="" selectedEnv="" lastResult={null} {...handlers} />)
    expect(bridge.getConformContext).toHaveBeenCalledTimes(2)
    bridge.getConformContext.mockRejectedValue(new Error('boom'))
    rerender(<ConformanceSidebarPanel workspacePath="/third" selectedSpec="" selectedEnv="" lastResult={null} {...handlers} />)
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })
})
