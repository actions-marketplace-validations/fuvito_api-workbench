import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { Sidebar, type SidebarProps } from './Sidebar'
import type { HistoryEntry } from '@/lib/bridge'

vi.mock('@/components/WorkspacePanel', () => ({
  WorkspacePanel: (p: { workspacePath: string; onSelectFile: (s: string) => void; onPasteToRoot?: () => void; onPasteToFolder?: (s: string) => void }) => (
    <div data-testid="workspace-panel">
      <span>{p.workspacePath}</span>
      <button onClick={() => p.onSelectFile('/ws/a.yaml')}>pick-file</button>
      <button onClick={() => p.onPasteToRoot?.()}>paste-root</button>
      <button onClick={() => p.onPasteToFolder?.('/ws/dir')}>paste-folder</button>
    </div>
  ),
}))
vi.mock('@/components/ScenariosPanel', () => ({
  ScenariosPanel: (p: { scenarios: unknown[]; onSelectScenario?: (a: string, b: string) => void; onNewFolder?: (a: string, b: string) => void; onRunFolder?: (s: string) => void }) => (
    <div data-testid="scenarios-panel">
      <span data-testid="scenario-count">{p.scenarios.length}</span>
      <button onClick={() => p.onSelectScenario?.('/s.yaml', 'S')}>pick-scenario</button>
      <button onClick={() => p.onNewFolder?.('/p', 'n')}>new-scenario-folder</button>
      <button onClick={() => p.onRunFolder?.('/f')}>run-scenario-folder</button>
    </div>
  ),
}))
vi.mock('@/components/CoverageSidebarPanel', () => ({
  CoverageSidebarPanel: (p: { selectedSpecFile: string | null; onSelectSpec: (s: string) => void }) => (
    <div data-testid="coverage-panel">
      <span data-testid="coverage-selected">{String(p.selectedSpecFile)}</span>
      <button onClick={() => p.onSelectSpec('spec.yaml')}>pick-spec</button>
    </div>
  ),
}))

function entry(over: Partial<HistoryEntry> = {}): HistoryEntry {
  return {
    id: 1, executedAt: new Date().toISOString(), method: 'GET', url: 'https://api.test/users',
    statusCode: 200, statusText: 'OK', durationMs: 42, sizeBytes: 10,
    headers: [], bodyType: 'none', bodyContent: '', ...over,
  }
}

function setup(over: Partial<SidebarProps> = {}) {
  const props: SidebarProps = {
    tab: 'history',
    workspacePath: '/ws',
    fileTree: [],
    history: [],
    onOpenWorkspace: vi.fn(), onNewWorkspace: vi.fn(), onSelectFile: vi.fn(), onDeleteFile: vi.fn(),
    onDuplicateFile: vi.fn(), onDeleteFolder: vi.fn(), onNewRequestInFolder: vi.fn(), onCreateFolder: vi.fn(),
    onRenameFile: vi.fn(), onRenameFolder: vi.fn(), onMoveFile: vi.fn(), onSelectHistory: vi.fn(),
    onClearHistory: vi.fn(), onImportOpenAPI: vi.fn(),
    ...over,
  }
  render(<Sidebar {...props} />)
  return props
}

describe('Sidebar — tab routing', () => {
  it('renders the explorer panel for the explorer tab and wires callbacks', () => {
    const p = setup({ tab: 'explorer', onPasteToRequestRoot: vi.fn(), onPasteToRequestFolder: vi.fn() })
    expect(screen.getByTestId('workspace-panel')).toHaveTextContent('/ws')
    fireEvent.click(screen.getByText('pick-file'))
    fireEvent.click(screen.getByText('paste-root'))
    fireEvent.click(screen.getByText('paste-folder'))
    expect(p.onSelectFile).toHaveBeenCalledWith('/ws/a.yaml')
    expect(p.onPasteToRequestRoot).toHaveBeenCalledTimes(1)
    expect(p.onPasteToRequestFolder).toHaveBeenCalledWith('/ws/dir')
    expect(screen.queryByTestId('scenarios-panel')).not.toBeInTheDocument()
  })

  it('renders the scenarios panel and defaults scenarios to an empty list', () => {
    const p = setup({ tab: 'scenarios', onSelectScenario: vi.fn(), onNewScenarioFolder: vi.fn(), onRunScenarioFolder: vi.fn() })
    expect(screen.getByTestId('scenario-count')).toHaveTextContent('0')
    fireEvent.click(screen.getByText('pick-scenario'))
    fireEvent.click(screen.getByText('new-scenario-folder'))
    fireEvent.click(screen.getByText('run-scenario-folder'))
    expect(p.onSelectScenario).toHaveBeenCalledWith('/s.yaml', 'S')
    expect(p.onNewScenarioFolder).toHaveBeenCalledWith('/p', 'n')
    expect(p.onRunScenarioFolder).toHaveBeenCalledWith('/f')
  })

  it('passes provided scenarios through', () => {
    setup({ tab: 'scenarios', scenarios: [{ title: 'A', path: '/a' }, { title: 'B', path: '/b' }] })
    expect(screen.getByTestId('scenario-count')).toHaveTextContent('2')
  })

  it('renders the coverage panel with selection state', () => {
    const onSelectCoverageSpec = vi.fn()
    setup({ tab: 'coverage', selectedCoverageSpecFile: 'api.yaml', onSelectCoverageSpec })
    expect(screen.getByTestId('coverage-selected')).toHaveTextContent('api.yaml')
    fireEvent.click(screen.getByText('pick-spec'))
    expect(onSelectCoverageSpec).toHaveBeenCalledWith('spec.yaml')
  })

  it('treats a missing coverage selection as null and tolerates a missing handler', () => {
    setup({ tab: 'coverage' })
    expect(screen.getByTestId('coverage-selected')).toHaveTextContent('null')
    expect(() => fireEvent.click(screen.getByText('pick-spec'))).not.toThrow()
  })
})

describe('Sidebar — history', () => {
  beforeEach(() => { vi.useFakeTimers(); vi.setSystemTime(new Date('2026-11-15T12:00:00')) })
  afterEach(() => vi.useRealTimers())

  it('shows an empty state with no entries and no clear control', () => {
    setup()
    expect(screen.getByText('No requests yet')).toBeInTheDocument()
    expect(screen.queryByText('Clear history')).not.toBeInTheDocument()
  })

  it('renders entries with method, status, duration and url', () => {
    setup({ history: [entry({ method: 'POST', statusCode: 201, durationMs: 7, url: 'https://x/y' })] })
    expect(screen.getByText('POST')).toBeInTheDocument()
    expect(screen.getByText('201')).toBeInTheDocument()
    expect(screen.getByText('7ms')).toBeInTheDocument()
    expect(screen.getByText('https://x/y')).toBeInTheDocument()
    expect(screen.getByText(/auth not stored/)).toBeInTheDocument()
  })

  it('shows a dash when there is no status code', () => {
    setup({ history: [entry({ statusCode: 0 })] })
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('groups entries as Today, Yesterday and older dates', () => {
    const now = new Date('2026-11-15T12:00:00')
    const yesterday = new Date(now); yesterday.setDate(now.getDate() - 1)
    const older = new Date(now); older.setDate(now.getDate() - 10)
    const lastYear = new Date('2025-03-02T09:00:00')
    setup({
      history: [
        entry({ id: 1, executedAt: now.toISOString() }),
        entry({ id: 2, executedAt: yesterday.toISOString() }),
        entry({ id: 3, executedAt: older.toISOString() }),
        entry({ id: 4, executedAt: lastYear.toISOString() }),
      ],
    })
    expect(screen.getByText('Today')).toBeInTheDocument()
    expect(screen.getByText('Yesterday')).toBeInTheDocument()
    expect(screen.getByText(new Date(older).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }))).toBeInTheDocument()
    expect(screen.getByText(lastYear.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' }))).toBeInTheDocument()
  })

  it('keeps entries on the same day in one group', () => {
    setup({ history: [entry({ id: 1 }), entry({ id: 2, url: 'https://x/2' })] })
    expect(screen.getAllByText('Today')).toHaveLength(1)
  })

  it('selects a history entry on click', () => {
    const e = entry()
    const p = setup({ history: [e] })
    fireEvent.click(screen.getByText('https://api.test/users'))
    expect(p.onSelectHistory).toHaveBeenCalledWith(e)
  })

  it('omits the re-run control when no handler is supplied', () => {
    setup({ history: [entry()] })
    expect(screen.queryByTitle('Re-run this request')).not.toBeInTheDocument()
  })

  it('re-runs an entry', () => {
    const e = entry()
    const onRerun = vi.fn()
    setup({ history: [e], onRerun })
    fireEvent.click(screen.getByTitle('Re-run this request'))
    expect(onRerun).toHaveBeenCalledWith(e)
  })

  it('requires confirmation before clearing history', () => {
    const p = setup({ history: [entry()] })
    fireEvent.click(screen.getByText('Clear history'))
    expect(p.onClearHistory).not.toHaveBeenCalled()
    expect(screen.getByText('Clear all history?')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Clear' }))
    expect(p.onClearHistory).toHaveBeenCalledTimes(1)
    expect(screen.queryByText('Clear all history?')).not.toBeInTheDocument()
  })

  it('cancelling the confirmation leaves history intact', () => {
    const p = setup({ history: [entry()] })
    fireEvent.click(screen.getByText('Clear history'))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(p.onClearHistory).not.toHaveBeenCalled()
    expect(screen.getByText('Clear history')).toBeInTheDocument()
  })
})
