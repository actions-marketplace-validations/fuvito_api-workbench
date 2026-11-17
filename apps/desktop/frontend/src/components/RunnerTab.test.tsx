import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { RunnerTab } from './RunnerTab'
import type { RunAllState } from '@/types'

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
