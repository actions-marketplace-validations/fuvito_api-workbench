import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ActivityRail } from './ActivityRail'

function setup(over: Partial<React.ComponentProps<typeof ActivityRail>> = {}) {
  const props: React.ComponentProps<typeof ActivityRail> = {
    activeTab: 'explorer',
    sidebarOpen: true,
    expanded: false,
    onTabClick: vi.fn(),
    onToggleExpanded: vi.fn(),
    onShowSettings: vi.fn(),
    onShowHelp: vi.fn(),
    ...over,
  }
  const utils = render(<ActivityRail {...props} />)
  return { props, ...utils }
}

describe('ActivityRail', () => {
  it('renders the three primary tabs, settings, help and toggle', () => {
    setup()
    for (const name of ['Requests', 'Scenarios', 'Contracts', 'Settings', 'Help']) {
      expect(screen.getByRole('button', { name })).toBeInTheDocument()
    }
    expect(screen.getByTitle('Expand rail')).toBeInTheDocument()
  })

  it('shows the CLI Builder button only when it can be opened', () => {
    const { unmount } = setup()
    expect(screen.queryByRole('button', { name: 'CLI Command Builder' })).not.toBeInTheDocument()
    unmount()
    const onOpenCliBuilder = vi.fn()
    setup({ onOpenCliBuilder })
    fireEvent.click(screen.getByRole('button', { name: 'CLI Command Builder' }))
    expect(onOpenCliBuilder).toHaveBeenCalledTimes(1)
  })

  it('marks only the active tab as current when the sidebar is open', () => {
    setup({ activeTab: 'coverage' })
    expect(screen.getByRole('button', { name: 'Contracts' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('button', { name: 'Requests' })).not.toHaveAttribute('aria-current')
    expect(screen.queryByRole('button', { name: 'History' })).not.toBeInTheDocument()
  })

  it('marks nothing as current when the sidebar is closed', () => {
    setup({ sidebarOpen: false })
    expect(screen.getByRole('button', { name: 'Requests' })).not.toHaveAttribute('aria-current')
  })

  it('reports tab clicks', () => {
    const { props } = setup()
    fireEvent.click(screen.getByRole('button', { name: 'Scenarios' }))
    fireEvent.click(screen.getByRole('button', { name: 'Contracts' }))
    expect(props.onTabClick).toHaveBeenNthCalledWith(1, 'scenarios')
    expect(props.onTabClick).toHaveBeenNthCalledWith(2, 'coverage')
  })

  it('opens settings and help', () => {
    const { props } = setup()
    fireEvent.click(screen.getByRole('button', { name: 'Settings' }))
    fireEvent.click(screen.getByRole('button', { name: 'Help' }))
    expect(props.onShowSettings).toHaveBeenCalledTimes(1)
    expect(props.onShowHelp).toHaveBeenCalledTimes(1)
  })

  it('toggles expansion', () => {
    const { props } = setup()
    fireEvent.click(screen.getByTitle('Expand rail'))
    expect(props.onToggleExpanded).toHaveBeenCalledTimes(1)
  })

  it('is narrow and label-free when collapsed', () => {
    const { container } = setup()
    expect((container.firstChild as HTMLElement).style.width).toBe('44px')
    expect(screen.queryByText('Collapse')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Requests' })).toHaveAttribute('title', 'Requests')
  })

  it('shows labels and no tooltips when expanded', () => {
    const { container } = setup({ expanded: true })
    expect((container.firstChild as HTMLElement).style.width).toBe('160px')
    expect(screen.getByText('Requests')).toBeInTheDocument()
    expect(screen.getByText('Contracts')).toBeInTheDocument()
    expect(screen.getByText('Collapse')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Requests' })).not.toHaveAttribute('title')
    expect(screen.getByTitle('Collapse rail')).toBeInTheDocument()
  })

  it('hides the demo indicator unless the demo server is running', () => {
    setup()
    expect(screen.queryByRole('button', { name: 'Stop demo server' })).not.toBeInTheDocument()
  })

  it('shows a stop-demo control while the demo runs', () => {
    const onStopDemo = vi.fn()
    setup({ demoRunning: true, onStopDemo })
    fireEvent.click(screen.getByRole('button', { name: 'Stop demo server' }))
    expect(onStopDemo).toHaveBeenCalledTimes(1)
    expect(screen.queryByText('Stop Demo')).not.toBeInTheDocument()
  })

  it('labels the stop-demo control when expanded', () => {
    setup({ demoRunning: true, expanded: true })
    expect(screen.getByText('Stop Demo')).toBeInTheDocument()
  })
})
