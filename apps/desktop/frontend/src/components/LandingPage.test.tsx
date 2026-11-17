import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { LandingPage } from './LandingPage'

function setup(over: Partial<React.ComponentProps<typeof LandingPage>> = {}) {
  const props: React.ComponentProps<typeof LandingPage> = {
    recentWorkspaces: [],
    demoStarting: false,
    demoRunning: false,
    onNewWorkspace: vi.fn(),
    onOpenWorkspace: vi.fn(),
    onOpenRecentWorkspace: vi.fn(),
    onTryDemo: vi.fn(),
    onStopDemo: vi.fn(),
    ...over,
  }
  render(<LandingPage {...props} />)
  return props
}

describe('LandingPage', () => {
  it('shows branding and primary actions without recents', () => {
    const p = setup()
    expect(screen.getByText('Clear Signal')).toBeInTheDocument()
    expect(screen.queryByText(/Resume/)).not.toBeInTheDocument()
    expect(screen.queryByText('Recent')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /New Workspace/ }))
    fireEvent.click(screen.getByRole('button', { name: /Open Workspace/ }))
    expect(p.onNewWorkspace).toHaveBeenCalledTimes(1)
    expect(p.onOpenWorkspace).toHaveBeenCalledTimes(1)
  })

  it('offers to resume the most recent workspace', () => {
    const p = setup({ recentWorkspaces: ['C:\\dev\\api-one', '/home/me/api-two'] })
    fireEvent.click(screen.getByRole('button', { name: /Resume api-one/ }))
    expect(p.onOpenRecentWorkspace).toHaveBeenCalledWith('C:\\dev\\api-one')
  })

  it('lists recents with name and parent directory and opens one', () => {
    const p = setup({ recentWorkspaces: ['C:\\dev\\api-one', '/home/me/api-two'] })
    expect(screen.getByText('api-two')).toBeInTheDocument()
    expect(screen.getByText('home/me')).toBeInTheDocument()
    fireEvent.click(screen.getByText('api-two'))
    expect(p.onOpenRecentWorkspace).toHaveBeenCalledWith('/home/me/api-two')
  })

  it('shows at most six recents', () => {
    setup({ recentWorkspaces: Array.from({ length: 9 }, (_, i) => `/w/ws${i}`) })
    expect(screen.getByText('ws5')).toBeInTheDocument()
    expect(screen.queryByText('ws6')).not.toBeInTheDocument()
  })

  it('falls back to the raw path for a single-segment workspace', () => {
    setup({ recentWorkspaces: ['solo'] })
    expect(screen.getAllByText('solo').length).toBeGreaterThan(0)
  })

  it('starts the demo', () => {
    const p = setup()
    fireEvent.click(screen.getByRole('button', { name: /Try the demo/ }))
    expect(p.onTryDemo).toHaveBeenCalledTimes(1)
  })

  it('disables the demo button while starting', () => {
    setup({ demoStarting: true })
    expect(screen.getByRole('button', { name: /Starting demo/ })).toBeDisabled()
  })

  it('offers to stop a running demo', () => {
    const p = setup({ demoRunning: true })
    expect(screen.queryByRole('button', { name: /Try the demo/ })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /Stop demo/ }))
    expect(p.onStopDemo).toHaveBeenCalledTimes(1)
  })
})
