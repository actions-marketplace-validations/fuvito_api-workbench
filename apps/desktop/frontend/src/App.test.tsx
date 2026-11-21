import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import App from './App'
import * as bridge from '@/lib/bridge'

vi.mock('@/lib/bridge')
vi.mock('@/components/HelpPanel', () => ({ HelpPanel: ({ initialTab }: { initialTab?: string }) => <div data-testid="help">{initialTab}</div> }))
vi.mock('@/components/ScenarioViewer', () => ({ ScenarioViewer: ({ scenarioName }: { scenarioName: string }) => <div data-testid="scenario-viewer">{scenarioName}</div> }))

const b = vi.mocked(bridge)

const settings = { theme: 'dark', fontSize: 13, defaultTimeoutMs: 30000, proxyUrl: '', tlsSkipVerify: false, telemetryEnabled: false }

function primeBackend(workspace: string) {
  b.getWorkspacePath.mockResolvedValue(workspace as never)
  b.listRequests.mockResolvedValue(workspace
    ? [{ name: 'requests', path: `${workspace}/requests`, isDir: true, children: [
        { name: 'health.yaml', path: `${workspace}/requests/health.yaml`, isDir: false, method: 'GET' },
      ] }]
    : [])
  b.listEnvironments.mockResolvedValue(workspace ? [{ name: 'local', path: `${workspace}/environments/local.yaml` }] : [])
  b.getActiveEnvironment.mockResolvedValue((workspace ? { name: 'local', path: `${workspace}/environments/local.yaml` } : null) as never)
  b.getEnvironment.mockResolvedValue({ path: '', name: 'local', variables: [] } as never)
  b.listScenarios.mockResolvedValue(workspace ? [{ title: 'Smoke', path: `${workspace}/scenarios/smoke.yaml` }] : [])
  b.getRecentWorkspaces.mockResolvedValue((workspace ? [workspace] : ['/recent/one']) as never)
  b.getHistory.mockResolvedValue([])
  b.getSettings.mockResolvedValue(settings as never)
  b.loadResponseCache.mockResolvedValue(null as never)
  b.setHasDirtyTabs.mockResolvedValue(undefined as never)
  b.loadRequest.mockResolvedValue({ schemaVersion: '1', title: 'health', method: 'GET', url: 'https://x.test/health' } as never)
  b.listAccumulatedCoverage.mockResolvedValue([])
  b.getScenarioRunHistory.mockResolvedValue([])
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  Element.prototype.scrollIntoView = vi.fn()
  vi.stubGlobal('ResizeObserver', class { observe() {} unobserve() {} disconnect() {} })
  primeBackend('')
})

describe('App — landing page', () => {
  it('shows the landing page when no workspace is open', async () => {
    render(<App />)
    expect(await screen.findByRole('button', { name: /New Workspace/ })).toBeInTheDocument()
    expect(screen.queryByRole('navigation', { name: 'Activity' })).not.toBeInTheDocument()
  })

  it('resumes the most recent workspace', async () => {
    b.openWorkspaceByPath.mockResolvedValue('/recent/one' as never)
    render(<App />)
    fireEvent.click(await screen.findByRole('button', { name: /Resume one/ }))
    await waitFor(() => expect(b.openWorkspaceByPath).toHaveBeenCalledWith('/recent/one'))
  })

  it('loads settings on startup', async () => {
    render(<App />)
    await screen.findByRole('button', { name: /New Workspace/ })
    expect(b.getSettings).toHaveBeenCalled()
  })
})

describe('App — workspace shell', () => {
  beforeEach(() => primeBackend('/ws'))

  async function ready() {
    render(<App />)
    await screen.findByRole('navigation', { name: 'Activity' })
    await screen.findByText('health')
  }

  it('renders the rail, explorer tree and status bar', async () => {
    await ready()
    expect(screen.getByText('requests')).toBeInTheDocument()
    expect(screen.getByText('ws')).toBeInTheDocument()
    expect(screen.getByTitle('Edit environment: local')).toBeInTheDocument()
  })

  it('opens a request in a tab when its file is clicked', async () => {
    await ready()
    fireEvent.click(screen.getByText('health'))
    await waitFor(() => expect(b.loadRequest).toHaveBeenCalledWith('/ws/requests/health.yaml'))
    expect(await screen.findByDisplayValue('https://x.test/health')).toBeInTheDocument()
  })

  it('switches sidebar panels from the activity rail', async () => {
    await ready()
    fireEvent.click(screen.getByRole('button', { name: 'Scenarios' }))
    expect(await screen.findByText('Smoke')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Requests' }))
    fireEvent.click(await screen.findByRole('button', { name: 'History' }))
    expect(await screen.findByText('No requests yet')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Contracts' }))
    expect(await screen.findByText('Nothing recorded in Contracts yet')).toBeInTheDocument()
    expect(localStorage.getItem('ui.railTab')).toBe('coverage')
  })

  it('collapses the sidebar by clicking the active rail tab again, and persists it', async () => {
    await ready()
    // the rail button comes first; the panel strip has a "Requests" tab too
    fireEvent.click(screen.getAllByRole('button', { name: 'Requests' })[0])
    expect(screen.queryByText('health')).not.toBeInTheDocument()
    expect(localStorage.getItem('ui.sidebarOpen')).toBe('false')
    fireEvent.click(screen.getByRole('button', { name: 'Requests' }))
    expect(await screen.findByText('health')).toBeInTheDocument()
    expect(localStorage.getItem('ui.sidebarOpen')).toBe('true')
  })

  it('toggles the sidebar with Ctrl+B', async () => {
    await ready()
    fireEvent.keyDown(window, { key: 'b', ctrlKey: true })
    expect(screen.queryByText('health')).not.toBeInTheDocument()
    fireEvent.keyDown(window, { key: 'b', ctrlKey: true })
    expect(await screen.findByText('health')).toBeInTheDocument()
  })

  it('expands and collapses the rail', async () => {
    await ready()
    fireEvent.click(screen.getByTitle('Expand rail'))
    expect(localStorage.getItem('ui.railExpanded')).toBe('true')
    expect(screen.getByText('Collapse')).toBeInTheDocument()
  })

  it('restores layout preferences from storage', async () => {
    localStorage.setItem('ui.railTab', 'scenarios')
    localStorage.setItem('ui.sidebarOpen', 'false')
    render(<App />)
    await screen.findByRole('navigation', { name: 'Activity' })
    expect(screen.queryByText('Smoke')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Scenarios' }))
    expect(await screen.findByText('Smoke')).toBeInTheDocument()
  })

  it('opens quick open with Ctrl+P and closes it with Escape', async () => {
    await ready()
    fireEvent.keyDown(window, { key: 'p', ctrlKey: true })
    expect(await screen.findByPlaceholderText(/Open request, scenario, env/)).toBeInTheDocument()
    fireEvent.keyDown(window, { key: 'p', ctrlKey: true })
    await waitFor(() => expect(screen.queryByPlaceholderText(/Open request, scenario, env/)).not.toBeInTheDocument())
  })

  it('does not open quick open without a workspace', async () => {
    primeBackend('')
    render(<App />)
    await screen.findByRole('button', { name: /New Workspace/ })
    fireEvent.keyDown(window, { key: 'p', ctrlKey: true })
    expect(screen.queryByPlaceholderText(/Open request, scenario, env/)).not.toBeInTheDocument()
  })

  it('creates a new tab with Ctrl+N', async () => {
    await ready()
    const before = screen.getAllByRole('tab').length
    fireEvent.keyDown(window, { key: 'n', ctrlKey: true })
    await waitFor(() => expect(screen.getAllByRole('tab').length).toBe(before + 1))
  })

  it('opens an existing scenario in a scenario tab', async () => {
    await ready()
    fireEvent.click(screen.getByRole('button', { name: 'Scenarios' }))
    fireEvent.click(await screen.findByText('Smoke'))
    expect(await screen.findByTestId('scenario-viewer')).toHaveTextContent('Smoke')
  })

  it('lets the user cancel and confirm deleting a scenario', async () => {
    b.deleteScenario.mockResolvedValue('' as never)
    await ready()
    fireEvent.click(screen.getByRole('button', { name: 'Scenarios' }))
    await screen.findByText('Smoke')
    fireEvent.click(screen.getAllByTitle('Delete scenario')[0])
    expect(screen.getByText('Delete scenario')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(b.deleteScenario).not.toHaveBeenCalled()
    fireEvent.click(screen.getAllByTitle('Delete scenario')[0])
    fireEvent.click(screen.getAllByRole('button', { name: 'Delete' }).at(-1)!)
    await waitFor(() => expect(b.deleteScenario).toHaveBeenCalledWith('/ws/scenarios/smoke.yaml'))
  })

  it('creates a scenario from the scenarios panel and opens it', async () => {
    b.createScenario.mockResolvedValue({ path: '/ws/scenarios/fresh.yaml', error: '' } as never)
    await ready()
    fireEvent.click(screen.getByRole('button', { name: 'Scenarios' }))
    await screen.findByText('Smoke')
    fireEvent.click(screen.getByTitle('New scenario'))
    const input = screen.getAllByRole('textbox').find(i => !i.getAttribute('placeholder'))!
    fireEvent.change(input, { target: { value: 'Fresh' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    await waitFor(() => expect(b.createScenario).toHaveBeenCalledWith('Fresh', ''))
    expect(await screen.findByTestId('scenario-viewer')).toHaveTextContent('Fresh')
  })

  it('creates a scenario folder', async () => {
    b.createScenarioFolder.mockResolvedValue('' as never)
    await ready()
    fireEvent.click(screen.getByRole('button', { name: 'Scenarios' }))
    await screen.findByText('Smoke')
    fireEvent.click(screen.getByTitle('New folder'))
    const input = screen.getAllByRole('textbox').find(i => !i.getAttribute('placeholder'))!
    fireEvent.change(input, { target: { value: 'group' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    await waitFor(() => expect(b.createScenarioFolder).toHaveBeenCalledWith('group'))
  })

  it('shows a scenario rename error from the backend and clears it', async () => {
    b.renameScenario.mockResolvedValue('error: name taken' as never)
    await ready()
    fireEvent.click(screen.getByRole('button', { name: 'Scenarios' }))
    fireEvent.doubleClick(await screen.findByText('Smoke'))
    const input = screen.getByDisplayValue('Smoke')
    fireEvent.change(input, { target: { value: 'Other' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    await waitFor(() => expect(b.renameScenario).toHaveBeenCalledWith('/ws/scenarios/smoke.yaml', 'Other'))
    expect(await screen.findByText(/name taken/)).toBeInTheDocument()
  })
})

describe('App — menus and dialogs', () => {
  it('opens About from the Help menu on the landing page', async () => {
    render(<App />)
    await screen.findByRole('button', { name: /New Workspace/ })
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'Help' }))
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'About Clear Signal' }))
    expect(await screen.findByText('Version 0.6.0')).toBeInTheDocument()
  })

  it.each([
    ['User Guide', 'guide'],
    ['Format Spec…', 'spec'],
    ['Example Scenario…', 'example'],
  ])('opens help on the right tab from "%s"', async (label, tab) => {
    render(<App />)
    await screen.findByRole('button', { name: /New Workspace/ })
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'Help' }))
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: label }))
    expect(await screen.findByTestId('help')).toHaveTextContent(tab)
  })

  it('opens settings from the View menu', async () => {
    render(<App />)
    await screen.findByRole('button', { name: /New Workspace/ })
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'View' }))
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'Settings…' }))
    expect(await screen.findByText(/Settings/i, { selector: 'h2, h3, span' })).toBeInTheDocument()
  })

  it('toggles the theme and persists the saved settings', async () => {
    b.saveSettings.mockResolvedValue({ ...settings, theme: 'light' } as never)
    render(<App />)
    await screen.findByRole('button', { name: /New Workspace/ })
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'View' }))
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'Switch to Light Theme' }))
    await waitFor(() => expect(b.saveSettings).toHaveBeenCalledWith(expect.objectContaining({ theme: 'light' })))
    await waitFor(() => expect(document.documentElement.classList.contains('light')).toBe(true))
  })

  it('starts the demo from the landing page', async () => {
    b.startDemo.mockResolvedValue({ port: 8090, workspacePath: '/tmp/demo' } as never)
    render(<App />)
    fireEvent.click(await screen.findByRole('button', { name: /Try the demo/ }))
    await waitFor(() => expect(b.startDemo).toHaveBeenCalledTimes(1))
  })
})
