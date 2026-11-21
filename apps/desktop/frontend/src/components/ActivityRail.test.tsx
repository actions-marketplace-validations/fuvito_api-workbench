import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ActivityRail } from './ActivityRail'

function setup(over: Partial<React.ComponentProps<typeof ActivityRail>> = {}) {
  const props: React.ComponentProps<typeof ActivityRail> = {
    activeTab: 'explorer',
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

  it('renders all six icons, Workspace first', () => {
    setup()
    for (const name of ['Workspace', 'Requests', 'Scenarios', 'Contracts', 'Check spec conformance', 'CLI Command Builder']) {
      expect(screen.getByRole('button', { name })).toBeInTheDocument()
    }
    const first = screen.getByRole('navigation', { name: 'Activity' }).querySelector('button')
    expect(first).toHaveAttribute('aria-label', 'Workspace')
  })

  it('lights exactly the icon the app names, whatever kind it is', () => {
    for (const [tab, name] of [['workspace', 'Workspace'], ['explorer', 'Requests'], ['scenarios', 'Scenarios'], ['coverage', 'Contracts'], ['conformance', 'Check spec conformance'], ['cli', 'CLI Command Builder']] as const) {
      const { unmount } = setup({ activeTab: tab })
      expect(document.querySelectorAll('[aria-current="page"]')).toHaveLength(1)
      expect(screen.getByRole('button', { name })).toHaveAttribute('aria-current', 'page')
      unmount()
    }
    setup({ activeTab: 'coverage' })
    expect(screen.queryByRole('button', { name: 'History' })).not.toBeInTheDocument()
  })

  it('reports tab clicks', () => {
    const { props } = setup()
    fireEvent.click(screen.getByRole('button', { name: 'Scenarios' }))
    fireEvent.click(screen.getByRole('button', { name: 'Contracts' }))
    fireEvent.click(screen.getByRole('button', { name: 'Workspace' }))
    fireEvent.click(screen.getByRole('button', { name: 'Check spec conformance' }))
    fireEvent.click(screen.getByRole('button', { name: 'CLI Command Builder' }))
    expect(props.onTabClick).toHaveBeenNthCalledWith(1, 'scenarios')
    expect(props.onTabClick).toHaveBeenNthCalledWith(2, 'coverage')
    expect(props.onTabClick).toHaveBeenNthCalledWith(3, 'workspace')
    expect(props.onTabClick).toHaveBeenNthCalledWith(4, 'conformance')
    expect(props.onTabClick).toHaveBeenNthCalledWith(5, 'cli')
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
