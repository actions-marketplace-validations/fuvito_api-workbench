import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ScenarioStatsDialog } from './ScenarioStatsDialog'
import type { ScenarioResult, StepResult } from '@/lib/bridge'

const step = (over: Partial<StepResult> = {}): StepResult => ({
  label: 'step', passed: true, skipped: false, statusCode: 200, durationMs: 100, extracted: {}, assertions: [], ...over,
})

const result = (steps: StepResult[], over: Partial<ScenarioResult> = {}): ScenarioResult => ({
  title: 't', passed: true, durationMs: 250, steps, ...over,
})

describe('ScenarioStatsDialog', () => {
  it('summarises counts and total time', () => {
    render(<ScenarioStatsDialog onClose={vi.fn()} result={result([
      step(), step({ passed: false }), step({ skipped: true, passed: false }), step(),
    ], { passed: false, durationMs: 777 })} />)
    expect(screen.getByText('FAILED')).toBeInTheDocument()
    expect(screen.getByText('2 passed · 1 failed · 1 skipped')).toBeInTheDocument()
    expect(screen.getByText('777 ms total')).toBeInTheDocument()
  })

  it('shows PASSED for a passing run', () => {
    render(<ScenarioStatsDialog onClose={vi.fn()} result={result([step()])} />)
    expect(screen.getByText('PASSED')).toBeInTheDocument()
  })

  it('lists steps with label fallback, status code and assertion tallies', () => {
    render(<ScenarioStatsDialog onClose={vi.fn()} result={result([
      step({ label: 'login', assertions: [{ type: 'status', passed: true, message: '' }, { type: 'jsonpath', passed: false, message: 'x' }] }),
      step({ label: '', statusCode: 404, passed: false }),
    ])} />)
    expect(screen.getByText('login')).toBeInTheDocument()
    expect(screen.getByText('Step 2')).toBeInTheDocument()
    expect(screen.getByText('1/2 assertions')).toBeInTheDocument()
    expect(screen.getByText('0/0 assertions')).toBeInTheDocument()
    expect(screen.getByText('200')).toBeInTheDocument()
    expect(screen.getByText('404')).toBeInTheDocument()
  })

  it('labels skipped steps and hides status and duration for them', () => {
    render(<ScenarioStatsDialog onClose={vi.fn()} result={result([step({ skipped: true, passed: false, statusCode: 0, durationMs: 0 })])} />)
    expect(screen.getByText('skipped')).toBeInTheDocument()
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('scales duration bars relative to the slowest step', () => {
    const { container } = render(<ScenarioStatsDialog onClose={vi.fn()} result={result([
      step({ durationMs: 50 }), step({ durationMs: 200 }), step({ durationMs: 100, passed: false }),
    ])} />)
    const bars = Array.from(container.querySelectorAll<HTMLElement>('.h-1 > div')).map(b => b.style.width)
    expect(bars).toEqual(['25%', '100%', '50%'])
    expect(container.querySelectorAll('.h-1 > div.bg-red-500')).toHaveLength(1)
  })

  it('copes with missing step data', () => {
    render(<ScenarioStatsDialog onClose={vi.fn()} result={{ ...result([]), steps: undefined as never }} />)
    expect(screen.getByText('0 passed · 0 failed · 0 skipped')).toBeInTheDocument()
    const noAssertions = step({ assertions: undefined as never })
    render(<ScenarioStatsDialog onClose={vi.fn()} result={result([noAssertions])} />)
    expect(screen.getByText('0/0 assertions')).toBeInTheDocument()
  })

  it('closes from the X button and the backdrop, but not from inside the dialog', () => {
    const onClose = vi.fn()
    const { container } = render(<ScenarioStatsDialog onClose={onClose} result={result([step()])} />)
    fireEvent.click(screen.getByText('Run Statistics'))
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button'))
    fireEvent.click(container.firstElementChild!)
    expect(onClose).toHaveBeenCalledTimes(2)
  })
})
