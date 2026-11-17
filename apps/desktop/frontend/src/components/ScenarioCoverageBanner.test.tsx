import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { CoverageBanner } from './ScenarioCoverageBanner'
import type { ScenarioCoverage, AccumulatedCoverage } from '@/lib/bridge'

const run: ScenarioCoverage = { covered: 1, total: 4, percent: 25, exercised: ['GET /a'], missed: ['GET /b', 'POST /c', 'GET /d'] }
const acc: AccumulatedCoverage = {
  specFile: 's.yaml', covered: 3, total: 4, percent: 75, totalRuns: 5, passedRuns: 4, failedRuns: 1,
  exercised: ['GET /a', 'GET /b', 'POST /c'], missed: ['GET /d'],
}

describe('CoverageBanner', () => {
  it('renders nothing without data', () => {
    const { container } = render(<CoverageBanner />)
    expect(container).toBeEmptyDOMElement()
  })

  it('shows a single run as "Spec Coverage" with what is missing', () => {
    render(<CoverageBanner coverage={run} />)
    expect(screen.getByText('Spec Coverage')).toBeInTheDocument()
    expect(screen.getByText('1 / 4 operations (25%)')).toBeInTheDocument()
    expect(screen.getByText(/Not yet covered: GET \/b · POST \/c · GET \/d/)).toBeInTheDocument()
    expect(screen.queryByTitle('Reset accumulated coverage')).not.toBeInTheDocument()
  })

  it('prefers accumulated data and summarises runs', () => {
    render(<CoverageBanner coverage={run} accumulated={acc} />)
    expect(screen.getByText('Accumulated Coverage')).toBeInTheDocument()
    expect(screen.getByText('3 / 4 operations (75%)')).toBeInTheDocument()
    expect(screen.getByText('5 runs · 4 passed · 1 failed')).toBeInTheDocument()
    expect(screen.getByText('This run: GET /a')).toBeInTheDocument()
    expect(screen.getByText('Not yet covered: GET /d')).toBeInTheDocument()
  })

  it('uses the singular for one run and omits "This run" when nothing was exercised', () => {
    render(<CoverageBanner coverage={{ ...run, exercised: [] }} accumulated={{ ...acc, totalRuns: 1 }} />)
    expect(screen.getByText(/^1 run ·/)).toBeInTheDocument()
    expect(screen.queryByText(/This run/)).not.toBeInTheDocument()
  })

  it('offers reset only for accumulated coverage', () => {
    const onReset = vi.fn()
    const { rerender } = render(<CoverageBanner coverage={run} onReset={onReset} />)
    expect(screen.queryByTitle('Reset accumulated coverage')).not.toBeInTheDocument()
    rerender(<CoverageBanner accumulated={acc} onReset={onReset} />)
    fireEvent.click(screen.getByTitle('Reset accumulated coverage'))
    expect(onReset).toHaveBeenCalledTimes(1)
  })

  it.each([
    [100, 'text-emerald-400', 'bg-emerald-500'],
    [50, 'text-amber-400', 'bg-amber-500'],
    [49.4, 'text-red-400', 'bg-red-500'],
  ])('colours %s%% correctly', (percent, text, bar) => {
    const { container } = render(<CoverageBanner coverage={{ ...run, percent, missed: [] }} />)
    expect(container.querySelector(`.${text}`)).not.toBeNull()
    expect(container.querySelector(`.${bar}`)).not.toBeNull()
    expect((container.querySelector(`.${bar}`) as HTMLElement).style.width).toBe(`${percent}%`)
  })

  it('hides the missing line at full coverage', () => {
    render(<CoverageBanner coverage={{ ...run, covered: 4, percent: 100, missed: [] }} />)
    expect(screen.queryByText(/Not yet covered/)).not.toBeInTheDocument()
  })
})
