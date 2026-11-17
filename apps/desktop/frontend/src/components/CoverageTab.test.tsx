import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { CoverageTab } from './CoverageTab'
import * as bridge from '@/lib/bridge'
import type { AccumulatedCoverage } from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({
  getAccumulatedCoverage: vi.fn(),
  resetAccumulatedCoverage: vi.fn(),
  saveContentToFile: vi.fn(),
  openInSystemViewer: vi.fn(),
  exportCoverageReport: vi.fn(),
}))

const b = vi.mocked(bridge)

function cov(over: Partial<AccumulatedCoverage> = {}): AccumulatedCoverage {
  return {
    specFile: 'specs/users-api.yaml',
    covered: 2,
    total: 3,
    percent: 66.6,
    totalRuns: 4,
    passedRuns: 3,
    failedRuns: 1,
    lastRunAt: '2026-11-15T10:00:00Z',
    exercised: ['POST /users', 'GET /users'],
    missed: ['DELETE /users/{id}'],
    operationResults: {
      'GET /users': { lastPassed: true, lastRunAt: new Date().toISOString() },
      'POST /users': { lastPassed: false, lastRunAt: new Date(Date.now() - 2 * 3_600_000).toISOString() },
    },
    ...over,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  b.getAccumulatedCoverage.mockResolvedValue(cov())
  b.resetAccumulatedCoverage.mockResolvedValue(undefined as never)
  b.saveContentToFile.mockResolvedValue(undefined as never)
  b.openInSystemViewer.mockResolvedValue('' as never)
  b.exportCoverageReport.mockImplementation(async (_spec, format) => ({ content: `report:${format}` }))
})

describe('CoverageTab — empty states', () => {
  it('asks for a workspace when none is open', () => {
    render(<CoverageTab workspacePath="" specFile="a.yaml" />)
    expect(screen.getByText('No workspace open')).toBeInTheDocument()
    expect(b.getAccumulatedCoverage).not.toHaveBeenCalled()
  })

  it('asks to select a spec when none is chosen', () => {
    render(<CoverageTab workspacePath="/ws" />)
    expect(screen.getByText('Select a spec from the sidebar')).toBeInTheDocument()
    expect(screen.getByText('Contracts')).toBeInTheDocument()
    expect(b.getAccumulatedCoverage).not.toHaveBeenCalled()
  })

  it('shows a loading state while fetching', async () => {
    let release!: (c: AccumulatedCoverage) => void
    b.getAccumulatedCoverage.mockReturnValue(new Promise(r => { release = r }))
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    expect(screen.getByText('Loading…')).toBeInTheDocument()
    await act(async () => { release(cov()) })
    expect(screen.queryByText('Loading…')).not.toBeInTheDocument()
  })

  it('shows an error with a retry that refetches', async () => {
    b.getAccumulatedCoverage.mockRejectedValueOnce(new Error('disk gone'))
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    expect(await screen.findByText(/disk gone/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByText('67%', { selector: 'span.text-4xl' })).toBeInTheDocument()
    expect(b.getAccumulatedCoverage).toHaveBeenCalledTimes(2)
  })
})

describe('CoverageTab — header', () => {
  it('shows the spec file name and its full path when they differ', async () => {
    render(<CoverageTab workspacePath="/ws" specFile="specs/users-api.yaml" />)
    expect(await screen.findByText('users-api.yaml')).toBeInTheDocument()
    expect(screen.getByTitle('specs/users-api.yaml')).toHaveTextContent('(specs/users-api.yaml)')
  })

  it('omits the path hint when the spec is a bare filename', async () => {
    b.getAccumulatedCoverage.mockResolvedValue(cov({ specFile: 'api.yaml' }))
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    await screen.findByText('67%', { selector: 'span.text-4xl' })
    expect(screen.queryByText('(api.yaml)')).not.toBeInTheDocument()
  })

  it('refreshes on the Refresh button and when refreshTrigger changes', async () => {
    const { rerender } = render(<CoverageTab workspacePath="/ws" specFile="api.yaml" refreshTrigger={0} />)
    await screen.findByText('67%', { selector: 'span.text-4xl' })
    expect(b.getAccumulatedCoverage).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByTitle('Refresh'))
    await waitFor(() => expect(b.getAccumulatedCoverage).toHaveBeenCalledTimes(2))
    rerender(<CoverageTab workspacePath="/ws" specFile="api.yaml" refreshTrigger={5} />)
    await waitFor(() => expect(b.getAccumulatedCoverage).toHaveBeenCalledTimes(3))
  })

  it('shows the no-data state when the spec has no entry', async () => {
    b.getAccumulatedCoverage.mockResolvedValue(null as never)
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    expect(await screen.findByText('No data for this spec yet')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Export' })).not.toBeInTheDocument()
  })
})

describe('CoverageTab — detail', () => {
  it('summarises coverage, runs and the last run time', async () => {
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    expect(await screen.findByText('67%', { selector: 'span.text-4xl' })).toBeInTheDocument()
    expect(screen.getByText('2/3 operations covered')).toBeInTheDocument()
    expect(screen.getByText('4 runs')).toBeInTheDocument()
    expect(screen.getByText('3 passed')).toBeInTheDocument()
    expect(screen.getByText('1 failed')).toBeInTheDocument()
    expect(screen.getByText(/Last run:/)).toBeInTheDocument()
  })

  it('uses the singular for one run and hides failed/last-run when absent', async () => {
    b.getAccumulatedCoverage.mockResolvedValue(cov({ totalRuns: 1, passedRuns: 1, failedRuns: 0, lastRunAt: undefined }))
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    expect(await screen.findByText('1 run')).toBeInTheDocument()
    expect(screen.queryByText(/failed/)).not.toBeInTheDocument()
    expect(screen.queryByText(/Last run:/)).not.toBeInTheDocument()
  })

  it.each([
    [90, 'text-emerald-400'],
    [50, 'text-amber-400'],
    [10, 'text-red-400'],
  ])('colours %i%% coverage with %s', async (percent, cls) => {
    b.getAccumulatedCoverage.mockResolvedValue(cov({ percent }))
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    expect((await screen.findByText(`${percent}%`, { selector: 'span.text-4xl' })).className).toContain(cls)
  })

  it('lists covered and missed operations sorted by path then method, with results', async () => {
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    await screen.findByText('Covered')
    expect(screen.getByText('Not covered')).toBeInTheDocument()
    const methods = screen.getAllByText(/^(GET|POST|DELETE)$/).map(e => e.textContent)
    expect(methods).toEqual(['GET', 'POST', 'DELETE'])
    expect(screen.getByText('✓ Pass')).toBeInTheDocument()
    expect(screen.getByText('✗ Fail')).toBeInTheDocument()
    expect(screen.getByText('just now')).toBeInTheDocument()
    expect(screen.getByText('2h ago')).toBeInTheDocument()
    expect(screen.getByText('Add these to your scenarios')).toBeInTheDocument()
  })

  it('collapses and expands the covered and missed sections', async () => {
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    await screen.findByText('Covered')
    fireEvent.click(screen.getByText('Covered'))
    expect(screen.queryByText('POST')).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('Covered'))
    expect(screen.getByText('POST')).toBeInTheDocument()

    fireEvent.click(screen.getByText('Not covered'))
    expect(screen.queryByText('DELETE')).not.toBeInTheDocument()
    expect(screen.queryByText('Add these to your scenarios')).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('Not covered'))
    expect(screen.getByText('DELETE')).toBeInTheDocument()
  })

  it('explains when a spec has no operation details', async () => {
    b.getAccumulatedCoverage.mockResolvedValue(cov({ exercised: [], missed: [] }))
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    expect(await screen.findByText(/Run a scenario with this spec to populate/)).toBeInTheDocument()
  })

  it('tolerates missing exercised/missed arrays', async () => {
    b.getAccumulatedCoverage.mockResolvedValue(cov({ exercised: undefined as never, missed: undefined as never }))
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    expect(await screen.findByText(/Run a scenario with this spec to populate/)).toBeInTheDocument()
  })

  it('renders the quality panel when quality data is present', async () => {
    b.getAccumulatedCoverage.mockResolvedValue(cov({
      quality: { neverExercised: [], happyPathOnly: ['GET /users'], securedNeverAuthed: [], errorsNeverTriggered: [] },
    }))
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    expect(await screen.findByText('Happy-path only')).toBeInTheDocument()
  })

  it('formats relative times in days for old runs', async () => {
    b.getAccumulatedCoverage.mockResolvedValue(cov({
      operationResults: { 'GET /users': { lastPassed: true, lastRunAt: new Date(Date.now() - 3 * 86_400_000).toISOString() } },
    }))
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    expect(await screen.findByText('3d ago')).toBeInTheDocument()
  })

  it('formats relative times in minutes', async () => {
    b.getAccumulatedCoverage.mockResolvedValue(cov({
      operationResults: { 'GET /users': { lastPassed: true, lastRunAt: new Date(Date.now() - 7 * 60_000).toISOString() } },
    }))
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    expect(await screen.findByText('7m ago')).toBeInTheDocument()
  })
})

describe('CoverageTab — reset', () => {
  it('requires confirmation, then resets and clears the view', async () => {
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    fireEvent.click(await screen.findByRole('button', { name: /Reset data/ }))
    expect(b.resetAccumulatedCoverage).not.toHaveBeenCalled()
    expect(screen.getByText('Clear all data?')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Yes, clear' }))
    await waitFor(() => expect(b.resetAccumulatedCoverage).toHaveBeenCalledWith('api.yaml'))
    expect(await screen.findByText('No data for this spec yet')).toBeInTheDocument()
  })

  it('cancelling the confirmation keeps the data', async () => {
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    fireEvent.click(await screen.findByRole('button', { name: /Reset data/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(b.resetAccumulatedCoverage).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: /Reset data/ })).toBeInTheDocument()
  })

  it('still clears the view when the backend reset fails', async () => {
    b.resetAccumulatedCoverage.mockRejectedValue(new Error('nope'))
    render(<CoverageTab workspacePath="/ws" specFile="api.yaml" />)
    fireEvent.click(await screen.findByRole('button', { name: /Reset data/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Yes, clear' }))
    expect(await screen.findByText('No data for this spec yet')).toBeInTheDocument()
  })
})

describe('CoverageTab — export', () => {
  const origCreate = URL.createObjectURL
  const origRevoke = URL.revokeObjectURL
  let anchorClick: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    URL.createObjectURL = vi.fn(() => 'blob:mock')
    URL.revokeObjectURL = vi.fn()
    anchorClick = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
  })
  afterEach(() => {
    URL.createObjectURL = origCreate
    URL.revokeObjectURL = origRevoke
    anchorClick.mockRestore()
  })

  async function openMenu() {
    render(<CoverageTab workspacePath="/ws" specFile="specs/users-api.yaml" />)
    fireEvent.click(await screen.findByRole('button', { name: 'Export' }))
  }

  it('toggles the menu and closes on outside click', async () => {
    await openMenu()
    expect(screen.getByText('Export CSV')).toBeInTheDocument()
    fireEvent.mouseDown(document.body)
    expect(screen.queryByText('Export CSV')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Export' }))
    fireEvent.click(screen.getByRole('button', { name: 'Export' }))
    expect(screen.queryByText('Export CSV')).not.toBeInTheDocument()
  })

  it('stays open when mousedown is inside the menu', async () => {
    await openMenu()
    fireEvent.mouseDown(screen.getByText('Export CSV'))
    expect(screen.getByText('Export CSV')).toBeInTheDocument()
  })

  it('downloads the CSV the Go core produced, named after the spec', async () => {
    await openMenu()
    fireEvent.click(screen.getByText('Export CSV'))
    await waitFor(() => expect(URL.createObjectURL).toHaveBeenCalledTimes(1))
    expect(b.exportCoverageReport).toHaveBeenCalledWith('specs/users-api.yaml', 'csv')
    const blob = vi.mocked(URL.createObjectURL).mock.calls[0][0] as Blob
    expect(blob.type).toBe('text/csv')
    expect(await blob.text()).toBe('report:csv')
    expect(anchorClick).toHaveBeenCalledTimes(1)
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:mock')
    expect(screen.queryByText('Export CSV')).not.toBeInTheDocument()
  })

  it('saves the Markdown report with a derived filename', async () => {
    await openMenu()
    fireEvent.click(screen.getByText('Export Markdown'))
    await waitFor(() => expect(b.saveContentToFile).toHaveBeenCalledTimes(1))
    expect(b.exportCoverageReport).toHaveBeenCalledWith('specs/users-api.yaml', 'markdown')
    expect(b.saveContentToFile).toHaveBeenCalledWith('report:markdown', 'users-api-contracts.md')
  })

  it('opens the HTML report in the system viewer', async () => {
    await openMenu()
    fireEvent.click(screen.getByText('Export HTML'))
    await waitFor(() => expect(b.openInSystemViewer).toHaveBeenCalledTimes(1))
    expect(b.exportCoverageReport).toHaveBeenCalledWith('specs/users-api.yaml', 'html')
    expect(b.openInSystemViewer).toHaveBeenCalledWith('report:html', 'html')
  })

  it('logs when the viewer reports an error', async () => {
    const err = vi.spyOn(console, 'error').mockImplementation(() => {})
    b.openInSystemViewer.mockResolvedValue('no viewer' as never)
    await openMenu()
    fireEvent.click(screen.getByText('Export HTML'))
    await waitFor(() => expect(err).toHaveBeenCalledWith('open report:', 'no viewer'))
    err.mockRestore()
  })

  it('shows the reason when the report cannot be made, and saves nothing', async () => {
    b.exportCoverageReport.mockResolvedValue({ content: '', error: 'no coverage has been recorded for this spec yet' })
    await openMenu()
    fireEvent.click(screen.getByText('Export Markdown'))
    expect(await screen.findByRole('alert')).toHaveTextContent('no coverage has been recorded')
    expect(b.saveContentToFile).not.toHaveBeenCalled()
    expect(URL.createObjectURL).not.toHaveBeenCalled()
  })

  it('shows an error when the call itself fails', async () => {
    b.exportCoverageReport.mockRejectedValue(new Error('boom'))
    await openMenu()
    fireEvent.click(screen.getByText('Export HTML'))
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
    expect(b.openInSystemViewer).not.toHaveBeenCalled()
  })

  it('clears an earlier error on the next export', async () => {
    b.exportCoverageReport.mockResolvedValueOnce({ content: '', error: 'first problem' })
    await openMenu()
    fireEvent.click(screen.getByText('Export CSV'))
    expect(await screen.findByRole('alert')).toHaveTextContent('first problem')
    fireEvent.click(screen.getByRole('button', { name: 'Export' }))
    fireEvent.click(screen.getByText('Export CSV'))
    await waitFor(() => expect(screen.queryByRole('alert')).not.toBeInTheDocument())
  })
})
