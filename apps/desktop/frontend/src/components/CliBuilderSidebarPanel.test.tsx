import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { CliBuilderSidebarPanel } from './CliBuilderSidebarPanel'

const bridge = vi.hoisted(() => ({ getCliCatalog: vi.fn() }))
vi.mock('@/lib/bridge', () => bridge)

const catalog = [
  { name: 'run', goal: 'Run scenarios' },
  { name: 'conform', goal: 'Check a server against its spec' },
  { name: 'diff', goal: 'Compare two spec versions' },
]

describe('CliBuilderSidebarPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    bridge.getCliCatalog.mockResolvedValue(catalog)
  })

  it('lists every command with its goal and marks the selected one', async () => {
    render(<CliBuilderSidebarPanel selected="conform" onSelect={vi.fn()} />)
    expect(await screen.findByRole('button', { name: /clrsig run/ })).toHaveTextContent('Run scenarios')
    expect(screen.getByRole('button', { name: /clrsig conform/ })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: /clrsig diff/ })).toHaveAttribute('aria-pressed', 'false')
  })

  it('selects a command on the page', async () => {
    const onSelect = vi.fn()
    render(<CliBuilderSidebarPanel selected="" onSelect={onSelect} />)
    fireEvent.click(await screen.findByRole('button', { name: /clrsig diff/ }))
    expect(onSelect).toHaveBeenCalledWith('diff')
  })

  it('reports a failed load', async () => {
    bridge.getCliCatalog.mockRejectedValue(new Error('boom'))
    render(<CliBuilderSidebarPanel selected="" onSelect={vi.fn()} />)
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })
})
