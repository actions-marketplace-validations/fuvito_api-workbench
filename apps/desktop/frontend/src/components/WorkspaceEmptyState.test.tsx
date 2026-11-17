import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { WorkspaceEmptyState } from './WorkspaceEmptyState'

describe('WorkspaceEmptyState', () => {
  it('offers new and open workspace', () => {
    const onNewWorkspace = vi.fn(), onOpenWorkspace = vi.fn()
    render(<WorkspaceEmptyState onNewWorkspace={onNewWorkspace} onOpenWorkspace={onOpenWorkspace} />)
    fireEvent.click(screen.getByRole('button', { name: 'New Workspace' }))
    fireEvent.click(screen.getByRole('button', { name: 'Open Existing' }))
    expect(onNewWorkspace).toHaveBeenCalledTimes(1)
    expect(onOpenWorkspace).toHaveBeenCalledTimes(1)
  })

  it('hides the demo card without a demo handler', () => {
    render(<WorkspaceEmptyState onNewWorkspace={vi.fn()} onOpenWorkspace={vi.fn()} />)
    expect(screen.queryByText('Try the demo')).not.toBeInTheDocument()
  })

  it('starts the demo', () => {
    const onTryDemo = vi.fn()
    render(<WorkspaceEmptyState onTryDemo={onTryDemo} onNewWorkspace={vi.fn()} onOpenWorkspace={vi.fn()} />)
    fireEvent.click(screen.getByRole('button', { name: /Try the Demo/ }))
    expect(onTryDemo).toHaveBeenCalledTimes(1)
  })

  it('disables the demo button while starting', () => {
    render(<WorkspaceEmptyState onTryDemo={vi.fn()} demoStarting onNewWorkspace={vi.fn()} onOpenWorkspace={vi.fn()} />)
    expect(screen.getByRole('button', { name: /Starting/ })).toBeDisabled()
  })
})
