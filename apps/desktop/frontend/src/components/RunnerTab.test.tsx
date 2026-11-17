import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { RunnerTab } from './RunnerTab'
import type { RunAllState } from '@/types'
import * as bridge from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({
  exportRunReport: vi.fn(),
  saveContentToFile: vi.fn(),
  openInSystemViewer: vi.fn(),
}))

const b = vi.mocked(bridge)

beforeEach(() => {
  vi.clearAllMocks()
  b.exportRunReport.mockImplementation(async (format) => ({ content: `report:${format}` }))
  b.saveContentToFile.mockResolvedValue(undefined as never)
  b.openInSystemViewer.mockResolvedValue('' as never)
})

function state(over: Partial<RunAllState> = {}): RunAllState {
  return {
    running: false, done: 3, total: 3, passed: 2,
    results: [
      { path: '/a', name: 'smoke', passed: true, durationMs: 40 },
      { path: '/b', name: 'login', passed: false, durationMs: 60 },
      { path: '/c', name: 'broken', passed: false, durationMs: 0, error: 'spec missing' },
    ],
    ...over,
  }
}

describe('RunnerTab', () => {
  it('shows an idle state and offers to run all', () => {
    const onRunAll = vi.fn()
    render(<RunnerTab runAllState={null} onRunAll={onRunAll} />)
    expect(screen.getByText('No run in progress.')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Run All Scenarios' }))
    expect(onRunAll).toHaveBeenCalledTimes(1)
  })

  it('has no run button when idle without a handler', () => {
    render(<RunnerTab runAllState={null} />)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  describe('running', () => {
    it('shows 1-based progress, a progress bar and stop', () => {
      const onCancel = vi.fn()
      const { container } = render(<RunnerTab runAllState={state({ running: true, done: 1, total: 4, passed: 1, results: [] })} onCancel={onCancel} />)
      expect(screen.getByText('Running 2 / 4…')).toBeInTheDocument()
      expect(screen.getByText('1 passed so far')).toBeInTheDocument()
      expect(screen.getByText('Waiting for first result…')).toBeInTheDocument()
      expect((container.querySelector('.bg-\\[var\\(--color-accent\\)\\].transition-all') as HTMLElement).style.width).toBe('25%')
      fireEvent.click(screen.getByRole('button', { name: 'Stop' }))
      expect(onCancel).toHaveBeenCalledTimes(1)
    })

    it('handles a zero total', () => {
      const { container } = render(<RunnerTab runAllState={state({ running: true, done: 0, total: 0, results: [] })} />)
      expect((container.querySelector('.transition-all.duration-300') as HTMLElement).style.width).toBe('0%')
    })

    it('hides stop, run again and clear while running without handlers', () => {
      render(<RunnerTab runAllState={state({ running: true })} onRunAll={vi.fn()} onDismiss={vi.fn()} />)
      expect(screen.queryByRole('button', { name: 'Run Again' })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: 'Clear' })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: 'Stop' })).not.toBeInTheDocument()
    })

    it('does not show the waiting message once results exist', () => {
      render(<RunnerTab runAllState={state({ running: true })} />)
      expect(screen.queryByText('Waiting for first result…')).not.toBeInTheDocument()
    })
  })

  describe('finished', () => {
    it('summarises failures and total duration', () => {
      render(<RunnerTab runAllState={state()} />)
      expect(screen.getByText('2 failed')).toBeInTheDocument()
      expect(screen.getByText('2/3 scenarios · 100ms total')).toBeInTheDocument()
    })

    it('celebrates when everything passed', () => {
      render(<RunnerTab runAllState={state({ passed: 1, results: [{ path: '/a', name: 'smoke', passed: true, durationMs: 5 }], total: 1 })} />)
      expect(screen.getByText('All passed')).toBeInTheDocument()
    })

    it('offers run again and clear', () => {
      const onRunAll = vi.fn(), onDismiss = vi.fn()
      render(<RunnerTab runAllState={state()} onRunAll={onRunAll} onDismiss={onDismiss} />)
      fireEvent.click(screen.getByRole('button', { name: 'Run Again' }))
      fireEvent.click(screen.getByRole('button', { name: 'Clear' }))
      expect(onRunAll).toHaveBeenCalledTimes(1)
      expect(onDismiss).toHaveBeenCalledTimes(1)
    })

    it('lists scenarios with durations and errors, and opens on click', () => {
      const onSelectScenario = vi.fn()
      render(<RunnerTab runAllState={state()} onSelectScenario={onSelectScenario} />)
      expect(screen.getByText('40ms')).toBeInTheDocument()
      expect(screen.getByText('spec missing')).toBeInTheDocument()
      expect(screen.getByTitle('spec missing')).toBeInTheDocument()
      fireEvent.click(screen.getByText('login'))
      expect(onSelectScenario).toHaveBeenCalledWith('/b', 'login')
    })

    it('tolerates clicks without a handler and missing durations', () => {
      render(<RunnerTab runAllState={state({ results: [{ path: '/a', name: 'x', passed: true, durationMs: undefined as never }] })} />)
      expect(screen.getByText('—')).toBeInTheDocument()
      expect(() => fireEvent.click(screen.getByText('x'))).not.toThrow()
    })
  })
})

describe('RunnerTab — export report', () => {
  const paths = ['/a', '/b', '/c']

  function openMenu() {
    render(<RunnerTab runAllState={state()} />)
    fireEvent.click(screen.getByRole('button', { name: /Export report/ }))
  }

  it('is offered only once a run has finished with results', () => {
    const { unmount } = render(<RunnerTab runAllState={state({ running: true, results: [] })} />)
    expect(screen.queryByRole('button', { name: /Export report/ })).not.toBeInTheDocument()
    unmount()
    const { unmount: u2 } = render(<RunnerTab runAllState={state({ running: true })} />)
    expect(screen.queryByRole('button', { name: /Export report/ })).not.toBeInTheDocument()
    u2()
    const { unmount: u3 } = render(<RunnerTab runAllState={null} />)
    expect(screen.queryByRole('button', { name: /Export report/ })).not.toBeInTheDocument()
    u3()
    render(<RunnerTab runAllState={state()} />)
    expect(screen.getByRole('button', { name: /Export report/ })).toBeInTheDocument()
  })

  it('lists the formats and closes on an outside click', () => {
    openMenu()
    for (const label of ['HTML report', 'JUnit XML', 'JSON', 'Markdown', 'Plain text']) {
      expect(screen.getByRole('button', { name: label })).toBeInTheDocument()
    }
    fireEvent.mouseDown(document.body)
    expect(screen.queryByRole('button', { name: 'JUnit XML' })).not.toBeInTheDocument()
  })

  it('opens the HTML report in the system viewer', async () => {
    openMenu()
    fireEvent.click(screen.getByRole('button', { name: 'HTML report' }))
    await waitFor(() => expect(b.openInSystemViewer).toHaveBeenCalledWith('report:html', 'html'))
    expect(b.exportRunReport).toHaveBeenCalledWith('html', paths)
    expect(b.saveContentToFile).not.toHaveBeenCalled()
  })

  it.each([
    ['JUnit XML', 'junit', 'run-results.xml'],
    ['JSON', 'json', 'run-results.json'],
    ['Markdown', 'markdown', 'run-results.md'],
    ['Plain text', 'text', 'run-results.txt'],
  ])('saves %s with a sensible file name', async (label, format, file) => {
    openMenu()
    fireEvent.click(screen.getByRole('button', { name: label }))
    await waitFor(() => expect(b.saveContentToFile).toHaveBeenCalledWith(`report:${format}`, file))
    expect(b.exportRunReport).toHaveBeenCalledWith(format, paths)
  })

  it('shows why a report could not be made and saves nothing', async () => {
    b.exportRunReport.mockResolvedValue({ content: '', error: 'there are no results to export' })
    openMenu()
    fireEvent.click(screen.getByRole('button', { name: 'JSON' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('no results to export')
    expect(b.saveContentToFile).not.toHaveBeenCalled()
  })

  it('shows an error when the call fails or the viewer cannot open', async () => {
    b.exportRunReport.mockRejectedValueOnce(new Error('boom'))
    openMenu()
    fireEvent.click(screen.getByRole('button', { name: 'JSON' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')

    b.openInSystemViewer.mockResolvedValue('no viewer' as never)
    fireEvent.click(screen.getByRole('button', { name: /Export report/ }))
    fireEvent.click(screen.getByRole('button', { name: 'HTML report' }))
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('no viewer'))
  })
})
