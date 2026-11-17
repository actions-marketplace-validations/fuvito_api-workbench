import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import App from './App'
import * as bridge from '@/lib/bridge'

vi.mock('@/lib/bridge')
vi.mock('@/components/HelpPanel', () => ({ HelpPanel: () => <div data-testid="help" /> }))
vi.mock('@/components/ScenarioViewer', () => ({ ScenarioViewer: () => <div data-testid="scenario-viewer" /> }))

const b = vi.mocked(bridge)

const settings = { theme: 'dark', fontSize: 13, defaultTimeoutMs: 30000, proxyUrl: '', tlsSkipVerify: false, telemetryEnabled: false }

function primeWorkspace() {
  b.getWorkspacePath.mockResolvedValue('/ws' as never)
  b.listRequests.mockResolvedValue([])
  b.listEnvironments.mockResolvedValue([])
  b.getActiveEnvironment.mockResolvedValue(null as never)
  b.getEnvironment.mockResolvedValue({ path: '', name: '', variables: [] } as never)
  b.listScenarios.mockResolvedValue([])
  b.getRecentWorkspaces.mockResolvedValue(['/ws'] as never)
  b.getHistory.mockResolvedValue([])
  b.getSettings.mockResolvedValue(settings as never)
  b.loadResponseCache.mockResolvedValue(null as never)
  b.setHasDirtyTabs.mockResolvedValue(undefined as never)
  b.listAccumulatedCoverage.mockResolvedValue([])
  b.getScenarioRunHistory.mockResolvedValue([])
  b.previewBundle.mockResolvedValue({ title: '', specFile: '', issues: [], entries: [], hasErrors: false } as never)
}

const exported = {
  path: 'C:/out/ws-bundle.yaml', environments: 2, requests: 1, scenarios: 3, redactedSecrets: 0, skipped: [], warnings: [],
}

async function openExportMenu() {
  render(<App />)
  await screen.findByRole('navigation', { name: 'Activity' })
  fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'File' }))
  fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'Export' }))
}

const bundleItem = () => screen.getByRole('menuitem', { name: 'Bundle (environments + requests + scenarios)…' })

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  Element.prototype.scrollIntoView = vi.fn()
  vi.stubGlobal('ResizeObserver', class { observe() {} unobserve() {} disconnect() {} })
  primeWorkspace()
})

describe('File → Export → Bundle', () => {
  it('is available without a response, while saving a response body is not', async () => {
    await openExportMenu()
    expect(bundleItem()).not.toBeDisabled()
    expect(screen.getByRole('menuitem', { name: 'Save Response Body…' })).toBeDisabled()
  })

  it('exports and reports what was written', async () => {
    b.exportBundle.mockResolvedValue(exported as never)
    await openExportMenu()
    fireEvent.mouseDown(bundleItem())
    expect(await screen.findByText('Exported 2 environments, 1 request, 3 scenarios to C:/out/ws-bundle.yaml.')).toBeInTheDocument()
    expect(b.exportBundle).toHaveBeenCalledTimes(1)
  })

  it('says when secrets were replaced and when files were skipped', async () => {
    b.exportBundle.mockResolvedValue({
      ...exported, environments: 1, requests: 0, scenarios: 1, redactedSecrets: 1,
      skipped: ['requests/broken.yaml — parsing yaml'], warnings: ['scenarios/a.yaml: requestRef "x" was left unchanged'],
    } as never)
    await openExportMenu()
    fireEvent.mouseDown(bundleItem())
    const notice = await screen.findByRole('status')
    expect(notice).toHaveTextContent('Exported 1 environment, 0 requests, 1 scenario')
    expect(notice).toHaveTextContent('1 secret value was replaced with CHANGE_ME.')
    expect(notice).toHaveTextContent('Skipped 1 file that could not be read: requests/broken.yaml — parsing yaml.')
    expect(notice).toHaveTextContent('requestRef "x" was left unchanged')
  })

  it('pluralises the secret count', async () => {
    b.exportBundle.mockResolvedValue({ ...exported, redactedSecrets: 3 } as never)
    await openExportMenu()
    fireEvent.mouseDown(bundleItem())
    expect(await screen.findByRole('status')).toHaveTextContent('3 secret values were replaced with CHANGE_ME.')
  })

  it('can dismiss the notice', async () => {
    b.exportBundle.mockResolvedValue(exported as never)
    await openExportMenu()
    fireEvent.mouseDown(bundleItem())
    const notice = await screen.findByRole('status')
    fireEvent.click(screen.getAllByRole('button', { name: 'Dismiss' }).at(-1)!)
    await waitFor(() => expect(notice).not.toBeInTheDocument())
  })

  it('shows nothing when the save dialog is cancelled', async () => {
    b.exportBundle.mockResolvedValue({ ...exported, path: '' } as never)
    await openExportMenu()
    fireEvent.mouseDown(bundleItem())
    await waitFor(() => expect(b.exportBundle).toHaveBeenCalled())
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('shows a backend error in the error banner', async () => {
    b.exportBundle.mockResolvedValue({ ...exported, path: '', error: 'the workspace has no environments, requests or scenarios to export' } as never)
    await openExportMenu()
    fireEvent.mouseDown(bundleItem())
    expect(await screen.findByText(/has no environments, requests or scenarios/)).toBeInTheDocument()
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('shows an error when the call itself fails', async () => {
    b.exportBundle.mockRejectedValue(new Error('ipc down'))
    await openExportMenu()
    fireEvent.mouseDown(bundleItem())
    expect(await screen.findByText('ipc down')).toBeInTheDocument()
  })
})

describe('File → Import → Bundle', () => {
  it('opens the import dialog', async () => {
    render(<App />)
    await screen.findByRole('navigation', { name: 'Activity' })
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'File' }))
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'Import' }))
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'Bundle (environments + scenarios)…' }))
    expect(await screen.findByText('Import Bundle')).toBeInTheDocument()
  })
})
